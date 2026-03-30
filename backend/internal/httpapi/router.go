package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RouterDeps struct {
	Pool              *pgxpool.Pool
	Env               string
	StoragePath       string
	CORSOrigins       []string
	SignPin           string
	PublicFrontendURL string
}

func NewRouter(d RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(180 * time.Second))

	origins := d.CORSOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173"}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", healthHandler(d))
	r.Get("/ready", readyHandler(d))

	r.Route("/api", func(r chi.Router) {
		r.Get("/expedientes", d.listExpedientes)
		r.Get("/expedientes/{id}", d.getExpediente)
		r.Post("/expedientes/{id}/documentos", d.postDocumento)
		r.Post("/expedientes/{id}/folios/reordenar", d.postReordenarFolios)
		r.Post("/expedientes/{eid}/documentos/{did}/procesar", d.postProcesarDocumento)
		r.Post("/expedientes/{id}/cerrar", d.postCerrarExpediente)

		r.Post("/documentos-procesados/{id}/firmar", d.postFirmarProcesado)
		r.Get("/public/validar/{token}", d.getValidarPublico)
		r.Get("/public/documentos-procesados/{id}/pdf", d.getProcessedPDFDownload)
	})

	return r
}

func healthHandler(d RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"env":    d.Env,
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func readyHandler(d RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		if d.Pool == nil {
			http.Error(w, `{"status":"no_db"}`, http.StatusServiceUnavailable)
			return
		}
		if err := d.Pool.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	}
}
