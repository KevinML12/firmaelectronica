-- Expedientes digitales judiciales (Organismo Judicial Guatemala) — esquema inicial
-- Roles de firma alineados al flujo OJ: Juez, Secretario, Oficial V

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE rol_firma_oj AS ENUM ('juez', 'secretario', 'oficial_v');

CREATE TYPE tipo_documento_oj AS ENUM (
    'resolucion',
    'notificacion',
    'auto',
    'oficio',
    'escrito',
    'otro'
);

CREATE TYPE tipo_timbre_cang AS ENUM ('forense', 'notarial');

CREATE TABLE juzgados (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo      TEXT NOT NULL,
    nombre      TEXT NOT NULL,
    departamento TEXT,
    municipio   TEXT,
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (codigo)
);

CREATE TABLE usuarios (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE,
    nombre_completo TEXT NOT NULL,
    rol_firma       rol_firma_oj NOT NULL,
    colegiado_no    TEXT,
    casillero_oj    TEXT,
    activo          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE expedientes (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    numero_unico     TEXT NOT NULL,
    juzgado_id       UUID NOT NULL REFERENCES juzgados (id),
    tipo_proceso     TEXT,
    estado           TEXT NOT NULL DEFAULT 'activo',
    metadata         JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (juzgado_id, numero_unico)
);

-- Contador correlativo de folios por expediente (bloqueo en transacción al asignar)
CREATE TABLE expediente_folio_contador (
    expediente_id UUID PRIMARY KEY REFERENCES expedientes (id) ON DELETE CASCADE,
    ultimo_folio  BIGINT NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE documentos (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expediente_id  UUID NOT NULL REFERENCES expedientes (id) ON DELETE CASCADE,
    tipo           tipo_documento_oj NOT NULL DEFAULT 'otro',
    titulo         TEXT,
    descripcion    TEXT,
    mime           TEXT NOT NULL DEFAULT 'application/pdf',
    storage_key    TEXT NOT NULL,
    page_count     INT NOT NULL DEFAULT 1, -- número de hojas del PDF (cada hoja = un folio al incorporar al expediente)
    sha256_original BYTEA,
    subido_por_id  UUID REFERENCES usuarios (id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Asignación de rango de folios al documento original subido
CREATE TABLE asignacion_folios (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    documento_id   UUID NOT NULL REFERENCES documentos (id) ON DELETE CASCADE,
    expediente_id  UUID NOT NULL REFERENCES expedientes (id) ON DELETE CASCADE,
    folio_inicio   BIGINT NOT NULL,
    folio_fin      BIGINT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (folio_fin >= folio_inicio)
);

-- Versión procesada: timbres, QR, código de verificación, PDF final
CREATE TABLE documentos_procesados (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    documento_id          UUID NOT NULL REFERENCES documentos (id) ON DELETE CASCADE,
    expediente_id         UUID NOT NULL REFERENCES expedientes (id) ON DELETE CASCADE,
    storage_key_salida    TEXT NOT NULL,
    codigo_verificacion   TEXT NOT NULL,
    qr_token              UUID NOT NULL DEFAULT gen_random_uuid(),
    sha256_salida         BYTEA NOT NULL,
    folio_inicio          BIGINT,
    folio_fin             BIGINT,
    metadata              JSONB NOT NULL DEFAULT '{}',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (codigo_verificacion)
);

CREATE INDEX idx_documentos_procesados_qr ON documentos_procesados (qr_token);

CREATE TABLE timbres_cang (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    documento_procesado_id  UUID NOT NULL REFERENCES documentos_procesados (id) ON DELETE CASCADE,
    tipo                    tipo_timbre_cang NOT NULL,
    hash_verificacion_sha256 BYTEA NOT NULL,
    detalle                 JSONB NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_timbres_doc ON timbres_cang (documento_procesado_id);

-- Firma electrónica interna + referencia a firma institucional (contenido sensible en storage o cifrado aparte)
CREATE TABLE firmas_documento (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    documento_procesado_id   UUID NOT NULL REFERENCES documentos_procesados (id) ON DELETE CASCADE,
    usuario_id               UUID NOT NULL REFERENCES usuarios (id),
    rol_firma                rol_firma_oj NOT NULL,
    hash_firma_interna       TEXT NOT NULL, -- en muestras OJ suele mostrarse como 32 hex (estilo MD5)
    sha256_contenido_firmado BYTEA NOT NULL,
    firma_institucional_ref  TEXT,
    firmado_en               TIMESTAMPTZ NOT NULL DEFAULT now(),
    orden                    INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_firmas_doc ON firmas_documento (documento_procesado_id);

-- Página pública de validación (token opaco; puede rotarse)
CREATE TABLE tokens_validacion_publica (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    documento_procesado_id   UUID NOT NULL REFERENCES documentos_procesados (id) ON DELETE CASCADE,
    token                    UUID NOT NULL DEFAULT gen_random_uuid(),
    activo                   BOOLEAN NOT NULL DEFAULT true,
    expira_en                TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (token)
);

CREATE INDEX idx_tokens_validacion ON tokens_validacion_publica (token) WHERE activo = true;

CREATE TABLE auditoria (
    id          BIGSERIAL PRIMARY KEY,
    tabla       TEXT NOT NULL,
    registro_id UUID,
    accion      TEXT NOT NULL,
    actor_id    UUID REFERENCES usuarios (id),
    detalle     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_auditoria_tabla ON auditoria (tabla, created_at DESC);

CREATE OR REPLACE FUNCTION expediente_init_folio_counter()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO expediente_folio_contador (expediente_id, ultimo_folio)
    VALUES (NEW.id, 0)
    ON CONFLICT (expediente_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_expediente_folio_init
    AFTER INSERT ON expedientes
    FOR EACH ROW
    EXECUTE PROCEDURE expediente_init_folio_counter();
