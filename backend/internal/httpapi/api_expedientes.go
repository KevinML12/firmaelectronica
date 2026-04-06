package httpapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/firmaelectronica/expedientes-oj/internal/folio"
	"github.com/firmaelectronica/expedientes-oj/internal/hashutil"
	"github.com/firmaelectronica/expedientes-oj/internal/pdfpages"
	"github.com/firmaelectronica/expedientes-oj/internal/plantillas"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const maxUploadBytes = 40 << 20

func enrichTipoPlantilla(tipoOut *string, etiquetaOut *string, plantillaOut *string, rolesOut *[]string, tipoDB string) {
	inf := plantillas.PorTipo(tipoDB)
	*tipoOut = inf.CodigoTipo
	*etiquetaOut = inf.Etiqueta
	*plantillaOut = inf.Plantilla
	if len(inf.Roles) > 0 {
		cp := make([]string, len(inf.Roles))
		copy(cp, inf.Roles)
		*rolesOut = cp
	} else {
		*rolesOut = nil
	}
}

type expedienteListItem struct {
	ID          string `json:"id"`
	NumeroUnico string `json:"numero_unico"`
	TipoProceso string `json:"tipo_proceso,omitempty"`
	Estado      string `json:"estado"`
}

type expedienteDetalle struct {
	ID          string         `json:"id"`
	NumeroUnico string         `json:"numero_unico"`
	TipoProceso string         `json:"tipo_proceso,omitempty"`
	Estado      string         `json:"estado"`
	CerradoEn   *string        `json:"cerrado_en,omitempty"`
	Checklist   map[string]any `json:"checklist"`
	Juzgado     map[string]any `json:"juzgado"`
	Hojas       []hojaItem     `json:"hojas"`
	Documentos  []docResumen   `json:"documentos"`
	Procesados  []procResumen  `json:"documentos_procesados"`
}

type docResumen struct {
	ID             string   `json:"id"`
	Titulo         string   `json:"titulo,omitempty"`
	Tipo           string   `json:"tipo,omitempty"`
	TipoEtiqueta   string   `json:"tipo_etiqueta,omitempty"`
	PlantillaDOCX  string   `json:"plantilla_docx,omitempty"`
	RolesSugeridos []string `json:"roles_sugeridos,omitempty"`
	PageCount      int      `json:"page_count"`
	StorageKey     string   `json:"storage_key"`
	CreatedAt      string   `json:"created_at"`
}

type procResumen struct {
	ID                 string   `json:"id"`
	DocumentoID        string   `json:"documento_id"`
	CodigoVerificacion string   `json:"codigo_verificacion"`
	QRToken            string   `json:"qr_token"`
	StorageKeySalida   string   `json:"storage_key_salida"`
	CreatedAt          string   `json:"created_at"`
	Tipo               string   `json:"tipo,omitempty"`
	TipoEtiqueta       string   `json:"tipo_etiqueta,omitempty"`
	PlantillaDOCX      string   `json:"plantilla_docx,omitempty"`
	RolesSugeridos     []string `json:"roles_sugeridos,omitempty"`
}

type hojaItem struct {
	ID          string `json:"id"`
	FolioNumero int64  `json:"folio_numero"`
	NumeroHoja  int    `json:"numero_hoja"`
	DocumentoID string `json:"documento_id"`
	Titulo      string `json:"titulo,omitempty"`
	Tipo        string `json:"tipo,omitempty"`
}

func (d *RouterDeps) listExpedientes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := d.Pool.Query(ctx, `
		SELECT id::text, numero_unico, COALESCE(tipo_proceso, ''), estado
		FROM expedientes
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var items []expedienteListItem
	for rows.Next() {
		var it expedienteListItem
		if err := rows.Scan(&it.ID, &it.NumeroUnico, &it.TipoProceso, &it.Estado); err != nil {
			httpError(w, http.StatusInternalServerError, err.Error())
			return
		}
		items = append(items, it)
	}
	if items == nil {
		items = []expedienteListItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (d *RouterDeps) getExpediente(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		httpError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var det expedienteDetalle
	det.Checklist = map[string]any{}
	det.Documentos = []docResumen{}
	det.Procesados = []procResumen{}
	var jCodigo, jNombre, jDep, jMun string
	var cerrado sql.NullTime
	var checklistRaw []byte
	err := d.Pool.QueryRow(ctx, `
		SELECT e.id::text, e.numero_unico, COALESCE(e.tipo_proceso, ''), e.estado,
		       e.cerrado_en, COALESCE(e.checklist::text, '{}'),
		       j.codigo, j.nombre, COALESCE(j.departamento, ''), COALESCE(j.municipio, '')
		FROM expedientes e
		JOIN juzgados j ON j.id = e.juzgado_id
		WHERE e.id = $1::uuid
	`, id).Scan(
		&det.ID, &det.NumeroUnico, &det.TipoProceso, &det.Estado,
		&cerrado, &checklistRaw,
		&jCodigo, &jNombre, &jDep, &jMun,
	)
	if err != nil {
		httpError(w, http.StatusNotFound, "expediente no encontrado")
		return
	}
	det.Juzgado = map[string]any{
		"codigo": jCodigo, "nombre": jNombre, "departamento": jDep, "municipio": jMun,
	}
	if cerrado.Valid {
		s := cerrado.Time.UTC().Format(time.RFC3339)
		det.CerradoEn = &s
	}
	if len(checklistRaw) > 0 {
		_ = json.Unmarshal(checklistRaw, &det.Checklist)
	}
	if det.Checklist == nil {
		det.Checklist = map[string]any{}
	}

	hrows, err := d.Pool.Query(ctx, `
		SELECT f.id::text, f.folio_numero, f.numero_hoja, f.documento_id::text,
		       COALESCE(d.titulo, ''), d.tipo::text
		FROM expediente_folio_hoja f
		JOIN documentos d ON d.id = f.documento_id
		WHERE f.expediente_id = $1::uuid
		ORDER BY f.folio_numero
	`, id)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer hrows.Close()

	for hrows.Next() {
		var h hojaItem
		if err := hrows.Scan(&h.ID, &h.FolioNumero, &h.NumeroHoja, &h.DocumentoID, &h.Titulo, &h.Tipo); err != nil {
			httpError(w, http.StatusInternalServerError, err.Error())
			return
		}
		det.Hojas = append(det.Hojas, h)
	}
	if det.Hojas == nil {
		det.Hojas = []hojaItem{}
	}

	docs, err := d.Pool.Query(ctx, `
		SELECT id::text, COALESCE(titulo, ''), page_count, storage_key, created_at::text, tipo::text
		FROM documentos WHERE expediente_id = $1::uuid ORDER BY created_at
	`, id)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer docs.Close()
	for docs.Next() {
		var dr docResumen
		var tipoDB string
		if err := docs.Scan(&dr.ID, &dr.Titulo, &dr.PageCount, &dr.StorageKey, &dr.CreatedAt, &tipoDB); err != nil {
			httpError(w, http.StatusInternalServerError, err.Error())
			return
		}
		enrichTipoPlantilla(&dr.Tipo, &dr.TipoEtiqueta, &dr.PlantillaDOCX, &dr.RolesSugeridos, tipoDB)
		det.Documentos = append(det.Documentos, dr)
	}

	prows, err := d.Pool.Query(ctx, `
		SELECT p.id::text, p.documento_id::text, p.codigo_verificacion, p.qr_token::text, p.storage_key_salida, p.created_at::text,
		       COALESCE(d.tipo::text, 'otro')
		FROM documentos_procesados p
		JOIN documentos d ON d.id = p.documento_id
		WHERE p.expediente_id = $1::uuid ORDER BY p.created_at
	`, id)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer prows.Close()
	for prows.Next() {
		var pr procResumen
		var tipoDB string
		if err := prows.Scan(&pr.ID, &pr.DocumentoID, &pr.CodigoVerificacion, &pr.QRToken, &pr.StorageKeySalida, &pr.CreatedAt, &tipoDB); err != nil {
			httpError(w, http.StatusInternalServerError, err.Error())
			return
		}
		enrichTipoPlantilla(&pr.Tipo, &pr.TipoEtiqueta, &pr.PlantillaDOCX, &pr.RolesSugeridos, tipoDB)
		det.Procesados = append(det.Procesados, pr)
	}

	writeJSON(w, http.StatusOK, det)
}

func (d *RouterDeps) postDocumento(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	expID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(expID); err != nil {
		httpError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var exists bool
	_ = d.Pool.QueryRow(ctx, `SELECT true FROM expedientes WHERE id = $1::uuid`, expID).Scan(&exists)
	if !exists {
		httpError(w, http.StatusNotFound, "expediente no encontrado")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		httpError(w, http.StatusBadRequest, "archivo demasiado grande o formulario inválido")
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		httpError(w, http.StatusBadRequest, "falta el campo file (PDF)")
		return
	}
	defer file.Close()

	if hdr.Size > maxUploadBytes {
		httpError(w, http.StatusBadRequest, "archivo demasiado grande")
		return
	}

	sub := filepath.Join(d.StoragePath, expID)
	if err := os.MkdirAll(sub, 0750); err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	docID := uuid.New().String()
	filename := docID + ".pdf"
	fullPath := filepath.Join(sub, filename)
	out, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := out.Close(); err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	numHojas, err := pdfpages.Count(fullPath)
	if err != nil {
		_ = os.Remove(fullPath)
		httpError(w, http.StatusBadRequest, fmt.Sprintf("no es un PDF válido: %v", err))
		return
	}

	sha, err := hashutil.FileSHA256(fullPath)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	relKey := filepath.ToSlash(filepath.Join(expID, filename))
	titulo := r.FormValue("titulo")
	if titulo == "" {
		titulo = hdr.Filename
	}
	tipoDoc := strings.TrimSpace(strings.ToLower(r.FormValue("tipo")))
	if tipoDoc == "" {
		tipoDoc = "otro"
	}
	if !plantillas.ValidTipo(tipoDoc) {
		httpError(w, http.StatusBadRequest, "tipo de documento inválido: use GET /api/catalogo/tipos-documento")
		return
	}

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO documentos (id, expediente_id, tipo, titulo, mime, storage_key, page_count, sha256_original)
		VALUES ($1::uuid, $2::uuid, $3::tipo_documento_oj, $4, 'application/pdf', $5, $6, $7)
	`, docID, expID, tipoDoc, titulo, relKey, numHojas, sha)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	inicio, fin, err := folio.AltaDocumento(ctx, d.Pool, expID, docID, numHojas)
	if err != nil {
		_, _ = d.Pool.Exec(ctx, `DELETE FROM documentos WHERE id = $1::uuid`, docID)
		_ = os.Remove(fullPath)
		httpError(w, http.StatusInternalServerError, fmt.Sprintf("folios: %v", err))
		return
	}

	_, _ = d.Pool.Exec(ctx, `
		UPDATE expedientes SET checklist = COALESCE(checklist, '{}'::jsonb) || '{"subio_pdf": true}'::jsonb WHERE id = $1::uuid
	`, expID)

	writeJSON(w, http.StatusCreated, map[string]any{
		"documento_id":  docID,
		"storage_key":   relKey,
		"hojas":         numHojas,
		"folio_inicio":  inicio,
		"folio_fin":     fin,
		"numero_unico":  expID,
		"mensaje_corto": fmt.Sprintf("Listo: %d hojas, folios %d al %d", numHojas, inicio, fin),
	})
}

type reorderBody struct {
	Orden  []string `json:"orden"`
	Motivo string   `json:"motivo"`
}

func (d *RouterDeps) postReordenarFolios(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	expID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(expID); err != nil {
		httpError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var body reorderBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if len(body.Orden) == 0 {
		httpError(w, http.StatusBadRequest, "orden vacío")
		return
	}

	if err := folio.ReordenarFolios(ctx, d.Pool, expID, body.Orden, nil, body.Motivo); err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"ok": "orden actualizado"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
