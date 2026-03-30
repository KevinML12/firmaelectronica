DROP TRIGGER IF EXISTS tr_expediente_folio_init ON expedientes;
DROP FUNCTION IF EXISTS expediente_init_folio_counter;

DROP TABLE IF EXISTS auditoria;
DROP TABLE IF EXISTS tokens_validacion_publica;
DROP TABLE IF EXISTS firmas_documento;
DROP TABLE IF EXISTS timbres_cang;
DROP TABLE IF EXISTS documentos_procesados;
DROP TABLE IF EXISTS asignacion_folios;
DROP TABLE IF EXISTS documentos;
DROP TABLE IF EXISTS expediente_folio_contador;
DROP TABLE IF EXISTS expedientes;
DROP TABLE IF EXISTS usuarios;
DROP TABLE IF EXISTS juzgados;

DROP TYPE IF EXISTS tipo_timbre_cang;
DROP TYPE IF EXISTS tipo_documento_oj;
DROP TYPE IF EXISTS rol_firma_oj;
