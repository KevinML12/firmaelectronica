package httpapi

import (
	"database/sql"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/firmaelectronica/expedientes-oj/internal/hashutil"
	"github.com/firmaelectronica/expedientes-oj/internal/pdfpages"
	"github.com/firmaelectronica/expedientes-oj/internal/pdfstamp"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (d *RouterDeps) postProcesarDocumento(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	expID := chi.URLParam(r, "eid")
	docID := chi.URLParam(r, "did")
	if _, err := uuid.Parse(expID); err != nil {
		httpError(w, http.StatusBadRequest, "expediente inválido")
		return
	}
	if _, err := uuid.Parse(docID); err != nil {
		httpError(w, http.StatusBadRequest, "documento inválido")
		return
	}

	var storageKey, expOfDoc string
	var pageCount int
	err := d.Pool.QueryRow(ctx, `
		SELECT storage_key, expediente_id::text, page_count FROM documentos WHERE id = $1::uuid
	`, docID).Scan(&storageKey, &expOfDoc, &pageCount)
	if err == pgx.ErrNoRows {
		httpError(w, http.StatusNotFound, "documento no encontrado")
		return
	}
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if expOfDoc != expID {
		httpError(w, http.StatusBadRequest, "el documento no pertenece a este expediente")
		return
	}

	var ya string
	_ = d.Pool.QueryRow(ctx, `SELECT id::text FROM documentos_procesados WHERE documento_id = $1::uuid LIMIT 1`, docID).Scan(&ya)
	if ya != "" {
		httpError(w, http.StatusConflict, "este documento ya fue procesado; use firmar sobre el procesado")
		return
	}

	rows, err := d.Pool.Query(ctx, `
		SELECT folio_numero FROM expediente_folio_hoja
		WHERE documento_id = $1::uuid ORDER BY numero_hoja
	`, docID)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var pages []pdfstamp.PageFolio
	for rows.Next() {
		var fn int64
		if err := rows.Scan(&fn); err != nil {
			rows.Close()
			httpError(w, http.StatusInternalServerError, err.Error())
			return
		}
		pages = append(pages, pdfstamp.PageFolio{FolioNumero: fn})
	}
	rows.Close()
	if len(pages) != pageCount {
		httpError(w, http.StatusInternalServerError, "folios no coinciden con las hojas; reordene o vuelva a subir")
		return
	}

	codigo, err := hashutil.CodigoVerificacion()
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	procID := uuid.New().String()
	qrTok := uuid.New().String()

	srcPath := filepath.Join(d.StoragePath, filepath.FromSlash(storageKey))
	outRel := filepath.ToSlash(filepath.Join(expID, "proc_"+procID+".pdf"))
	dstPath := filepath.Join(d.StoragePath, filepath.FromSlash(outRel))

	if err := os.MkdirAll(filepath.Dir(dstPath), 0750); err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	validURL := d.PublicFrontendURL + "/validar/" + qrTok
	if err := pdfstamp.Apply(srcPath, dstPath, pages, codigo, validURL); err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}

	shaOut, err := hashutil.FileSHA256(dstPath)
	if err != nil {
		_ = os.Remove(dstPath)
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var fi, ff int64
	_ = d.Pool.QueryRow(ctx, `
		SELECT MIN(folio_numero), MAX(folio_numero) FROM expediente_folio_hoja WHERE documento_id = $1::uuid
	`, docID).Scan(&fi, &ff)

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO documentos_procesados (
			id, documento_id, expediente_id, storage_key_salida, codigo_verificacion, qr_token, sha256_salida, folio_inicio, folio_fin
		) VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6::uuid, $7, $8, $9)
	`, procID, docID, expID, outRel, codigo, qrTok, shaOut, fi, ff)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO timbres_cang (documento_procesado_id, tipo, hash_verificacion_sha256, detalle)
		VALUES ($1::uuid, 'forense', $2, '{"origen":"procesar"}'::jsonb)
	`, procID, shaOut)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO tokens_validacion_publica (documento_procesado_id, token, activo)
		VALUES ($1::uuid, $2::uuid, true)
	`, procID, qrTok)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = tx.Exec(ctx, `
		UPDATE expedientes SET checklist = COALESCE(checklist, '{}'::jsonb) || '{"pdf_procesado": true}'::jsonb WHERE id = $1::uuid
	`, expID)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(ctx); err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"documento_procesado_id": procID,
		"codigo_verificacion":    codigo,
		"qr_token":               qrTok,
		"url_validar":            validURL,
		"url_descarga":           fmt.Sprintf("/api/public/documentos-procesados/%s/pdf?token=%s", procID, qrTok),
		"mensaje":                "PDF estampado con folios, código, QR y rúbricas.",
	})
}

type firmarBody struct {
	PIN        string `json:"pin"`
	Rol        string `json:"rol"`
	NombreActa string `json:"nombre_acta"`
}

func (d *RouterDeps) postFirmarProcesado(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	procID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(procID); err != nil {
		httpError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var body firmarBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.PIN != d.SignPin {
		httpError(w, http.StatusUnauthorized, "PIN incorrecto")
		return
	}

	uid, ok := usuarioDemoPorRol(body.Rol)
	if !ok {
		httpError(w, http.StatusBadRequest, "rol inválido: ver lista de roles en documentación del tramitador")
		return
	}

	var storageKey, expID string
	err := d.Pool.QueryRow(ctx, `
		SELECT storage_key_salida, expediente_id::text
		FROM documentos_procesados WHERE id = $1::uuid
	`, procID).Scan(&storageKey, &expID)
	if err == pgx.ErrNoRows {
		httpError(w, http.StatusNotFound, "documento procesado no encontrado")
		return
	}
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	inPath := filepath.Join(d.StoragePath, filepath.FromSlash(storageKey))
	tmpPath := inPath + ".firmando.pdf"
	n, err := pdfpages.Count(inPath)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	nombreMostrar := strings.TrimSpace(body.NombreActa)
	if nombreMostrar == "" {
		_ = d.Pool.QueryRow(ctx, `SELECT nombre_completo FROM usuarios WHERE id = $1::uuid`, uid).Scan(&nombreMostrar)
	}
	if nombreMostrar == "" {
		nombreMostrar = "Firmante"
	}

	sum := md5.Sum([]byte(procID + nombreMostrar + body.Rol + time.Now().UTC().Format(time.RFC3339Nano)))
	hashInterno := strings.ToUpper(hex.EncodeToString(sum[:]))

	var jNom, jDep string
	_ = d.Pool.QueryRow(ctx, `
		SELECT COALESCE(j.nombre, ''), COALESCE(j.departamento, '')
		FROM expedientes e JOIN juzgados j ON j.id = e.juzgado_id
		WHERE e.id = $1::uuid
	`, expID).Scan(&jNom, &jDep)
	dependenciaPDF := strings.TrimSpace(jNom)
	if t := strings.TrimSpace(jDep); t != "" {
		if dependenciaPDF != "" {
			dependenciaPDF += " · "
		}
		dependenciaPDF += t
	}

	if err := pdfstamp.FirmaElectrónicaBloque(inPath, tmpPath, etiquetaRol(body.Rol), nombreMostrar, hashInterno, dependenciaPDF, n); err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.Remove(inPath); err != nil {
		_ = os.Remove(tmpPath)
		httpError(w, http.StatusInternalServerError, "no se pudo actualizar el archivo")
		return
	}
	if err := os.Rename(tmpPath, inPath); err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	shaNew, err := hashutil.FileSHA256(inPath)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	payload := sha256.Sum256([]byte(procID + nombreMostrar + body.Rol + string(shaNew)))
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `UPDATE documentos_procesados SET sha256_salida = $2 WHERE id = $1::uuid`, procID, shaNew)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var naPtr *string
	if s := strings.TrimSpace(body.NombreActa); s != "" {
		naPtr = &s
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO firmas_documento (documento_procesado_id, usuario_id, rol_firma, hash_firma_interna, sha256_contenido_firmado, nombre_acta, orden)
		VALUES ($1::uuid, $2::uuid, $3::rol_firma_oj, $4, $5, $6,
			(SELECT COALESCE(MAX(orden),0)+1 FROM firmas_documento f WHERE f.documento_procesado_id = $1::uuid))
	`, procID, uid, strings.ToLower(strings.TrimSpace(body.Rol)), hashInterno, payload[:], naPtr)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, _ = tx.Exec(ctx, `
		UPDATE expedientes SET checklist = COALESCE(checklist, '{}'::jsonb) || '{"firmado": true}'::jsonb WHERE id = $1::uuid
	`, expID)

	if err := tx.Commit(ctx); err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"ok":     "firma registrada y bloque añadido a la última hoja",
		"nombre": nombreMostrar,
	})
}

func usuarioDemoPorRol(rol string) (uuid.UUID, bool) {
	switch strings.ToLower(strings.TrimSpace(rol)) {
	case "juez":
		return uuid.MustParse("33333333-3333-4333-8333-333333333301"), true
	case "secretario":
		return uuid.MustParse("33333333-3333-4333-8333-333333333302"), true
	case "oficial_v":
		return uuid.MustParse("33333333-3333-4333-8333-333333333303"), true
	case "parte_actora":
		return uuid.MustParse("33333333-3333-4333-8333-333333333304"), true
	case "patrono_abogado":
		return uuid.MustParse("33333333-3333-4333-8333-333333333305"), true
	case "inspectora_trabajo":
		return uuid.MustParse("33333333-3333-4333-8333-333333333306"), true
	case "representante_demandada":
		return uuid.MustParse("33333333-3333-4333-8333-333333333307"), true
	case "magistrado":
		return uuid.MustParse("33333333-3333-4333-8333-333333333308"), true
	case "notificador":
		return uuid.MustParse("33333333-3333-4333-8333-333333333309"), true
	case "ministro_ejecutor":
		return uuid.MustParse("33333333-3333-4333-8333-333333333310"), true
	default:
		return uuid.Nil, false
	}
}

func etiquetaRol(rol string) string {
	switch strings.ToLower(strings.TrimSpace(rol)) {
	case "juez":
		return "JUEZ"
	case "secretario":
		return "SECRETARIO(A)"
	case "oficial_v":
		return "OFICIAL V"
	case "parte_actora":
		return "PARTE ACTORA / COMPARECIENTE"
	case "patrono_abogado":
		return "ABOGADO PATRONO"
	case "inspectora_trabajo":
		return "INSPECTORA DE TRABAJO"
	case "representante_demandada":
		return "REPRESENTANTE DEMANDADA"
	case "magistrado":
		return "MAGISTRADO(A)"
	case "notificador":
		return "NOTIFICADOR OJ"
	case "ministro_ejecutor":
		return "MINISTRO EJECUTOR"
	default:
		return strings.ToUpper(strings.ReplaceAll(rol, "_", " "))
	}
}

type cerrarBody struct {
	PIN string `json:"pin"`
}

func (d *RouterDeps) postCerrarExpediente(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	expID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(expID); err != nil {
		httpError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body cerrarBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.PIN != d.SignPin {
		httpError(w, http.StatusUnauthorized, "PIN incorrecto")
		return
	}

	var cerradoEn sql.NullTime
	err := d.Pool.QueryRow(ctx, `SELECT cerrado_en FROM expedientes WHERE id = $1::uuid`, expID).Scan(&cerradoEn)
	if err == pgx.ErrNoRows {
		httpError(w, http.StatusNotFound, "expediente no encontrado")
		return
	}
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cerradoEn.Valid {
		httpError(w, http.StatusBadRequest, "el expediente ya estaba cerrado")
		return
	}

	tag, err := d.Pool.Exec(ctx, `
		UPDATE expedientes SET estado = 'cerrado', cerrado_en = now(),
			checklist = COALESCE(checklist, '{}'::jsonb) || '{"expediente_cerrado": true}'::jsonb
		WHERE id = $1::uuid AND estado <> 'cerrado'
	`, expID)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		httpError(w, http.StatusBadRequest, "no se pudo cerrar")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "expediente cerrado"})
}

func (d *RouterDeps) getValidarPublico(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tok := chi.URLParam(r, "token")
	if _, err := uuid.Parse(tok); err != nil {
		httpError(w, http.StatusBadRequest, "token inválido")
		return
	}

	var procID, codigo, expNum, docTitulo string
	var sha []byte
	err := d.Pool.QueryRow(ctx, `
		SELECT p.id::text, p.codigo_verificacion, e.numero_unico, COALESCE(d.titulo, ''), p.sha256_salida
		FROM tokens_validacion_publica t
		JOIN documentos_procesados p ON p.id = t.documento_procesado_id
		JOIN expedientes e ON e.id = p.expediente_id
		JOIN documentos d ON d.id = p.documento_id
		WHERE t.token = $1::uuid AND t.activo = true
	`, tok).Scan(&procID, &codigo, &expNum, &docTitulo, &sha)
	if err == pgx.ErrNoRows {
		httpError(w, http.StatusNotFound, "documento no encontrado o token inválido")
		return
	}
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rows, err := d.Pool.Query(ctx, `
		SELECT rol_firma::text, COALESCE(nombre_acta, u.nombre_completo), hash_firma_interna, firmado_en::text
		FROM firmas_documento f
		JOIN usuarios u ON u.id = f.usuario_id
		WHERE f.documento_procesado_id = $1::uuid ORDER BY f.orden
	`, procID)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var firmas []map[string]string
	for rows.Next() {
		var rol, nom, hh, fe string
		if err := rows.Scan(&rol, &nom, &hh, &fe); err != nil {
			httpError(w, http.StatusInternalServerError, err.Error())
			return
		}
		firmas = append(firmas, map[string]string{"rol": rol, "nombre": nom, "hash_interno": hh, "fecha": fe})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"valido":                  true,
		"documento_procesado_id": procID,
		"expediente_numero":       expNum,
		"documento_titulo":        docTitulo,
		"codigo_verificacion":     codigo,
		"sha256":                  hex.EncodeToString(sha),
		"firmas":                  firmas,
		"mensaje":                 "Documento registrado en el sistema de expediente digital (demostración).",
	})
}

func (d *RouterDeps) getProcessedPDFDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	procID := chi.URLParam(r, "id")
	tok := r.URL.Query().Get("token")
	if _, err := uuid.Parse(procID); err != nil || tok == "" {
		httpError(w, http.StatusBadRequest, "falta token o id inválido")
		return
	}
	if _, err := uuid.Parse(tok); err != nil {
		httpError(w, http.StatusBadRequest, "token inválido")
		return
	}

	var key string
	err := d.Pool.QueryRow(ctx, `
		SELECT p.storage_key_salida FROM documentos_procesados p
		JOIN tokens_validacion_publica t ON t.documento_procesado_id = p.id
		WHERE p.id = $1::uuid AND t.token = $2::uuid AND t.activo = true
	`, procID, tok).Scan(&key)
	if err == pgx.ErrNoRows {
		httpError(w, http.StatusNotFound, "no autorizado")
		return
	}
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	path := filepath.Join(d.StoragePath, filepath.FromSlash(key))
	w.Header().Set("Content-Type", "application/pdf")
	http.ServeFile(w, r, path)
}
