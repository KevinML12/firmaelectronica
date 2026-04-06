package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	Env         string
	// BaseURL pública del backend (validación QR, enlaces en PDF)
	PublicAPIURL string
	// Orígenes CORS separados por coma (ej. http://localhost:5173,https://*.vercel.app)
	CORSOrigins []string
	// Directorio para PDFs subidos
	StoragePath string
	// PIN servidor para firmar / cerrar (debe coincidir con PUBLIC_SIGN_PIN del frontend)
	SignPin string
	// URL del frontend para enlaces en QR (ej. https://mi-app.vercel.app)
	PublicFrontendURL string
	// Si true (defecto), aplica migraciones SQL al arrancar el servidor.
	AutoMigrate bool
}

func Load() (Config, error) {
	_ = os.Setenv("TZ", "America/Guatemala")

	defCORS := "http://localhost:5173,http://127.0.0.1:5173"
	corsRaw := strings.TrimSpace(os.Getenv("CORS_ORIGINS"))
	if corsRaw == "" {
		corsRaw = strings.TrimSpace(os.Getenv("CORS_ORIGIN"))
	}
	if corsRaw == "" {
		corsRaw = defCORS
	}

	c := Config{
		HTTPAddr:          getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		Env:               getEnv("ENV", "development"),
		PublicAPIURL:      normalizeHTTPURL(getEnv("PUBLIC_API_URL", "http://localhost:8080")),
		CORSOrigins:       splitComma(corsRaw),
		StoragePath:       getEnv("STORAGE_PATH", "./data/storage"),
		SignPin:           getEnv("SIGN_PIN", "2026"),
		PublicFrontendURL: strings.TrimRight(getEnv("PUBLIC_FRONTEND_URL", "http://localhost:5173"), "/"),
		AutoMigrate:       autoMigrateDefault(getEnv("AUTO_MIGRATE", "true")),
	}
	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" {
		c.HTTPAddr = ":" + p
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL es obligatorio")
	}
	return c, nil
}

// normalizeHTTPURL añade https:// si falta esquema (evita enlaces rotos en PDF/QR).
func normalizeHTTPURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return u
	}
	if strings.HasPrefix(strings.ToLower(u), "http://") || strings.HasPrefix(strings.ToLower(u), "https://") {
		return strings.TrimRight(u, "/")
	}
	return "https://" + strings.TrimRight(strings.TrimPrefix(u, "/"), "/")
}

func autoMigrateDefault(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func splitComma(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
