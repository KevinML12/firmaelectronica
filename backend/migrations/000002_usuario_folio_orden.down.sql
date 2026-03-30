DROP TABLE IF EXISTS folio_reorden_eventos;
DROP TABLE IF EXISTS expediente_folio_hoja;

DROP INDEX IF EXISTS idx_usuarios_tipo_numero_identificacion;
DROP INDEX IF EXISTS idx_usuarios_identificador_funcional;

ALTER TABLE usuarios
    DROP COLUMN IF EXISTS firma_grafica_metadata,
    DROP COLUMN IF EXISTS firma_grafica_storage_key,
    DROP COLUMN IF EXISTS identificador_funcional,
    DROP COLUMN IF EXISTS numero_identificacion,
    DROP COLUMN IF EXISTS tipo_identificacion;

DROP TYPE IF EXISTS tipo_identificacion_usuario;
