-- Identificación de usuario (tipo + número) y firma gráfica propia
-- Mapeo folio ↔ hoja: un folio correlativo = una HOJA del documento (no “página” abstracta)

CREATE TYPE tipo_identificacion_usuario AS ENUM ('dpi', 'pasaporte', 'licencia', 'otro');

ALTER TABLE usuarios
    ADD COLUMN tipo_identificacion tipo_identificacion_usuario,
    ADD COLUMN numero_identificacion TEXT,
    ADD COLUMN identificador_funcional TEXT,
    ADD COLUMN firma_grafica_storage_key TEXT,
    ADD COLUMN firma_grafica_metadata JSONB NOT NULL DEFAULT '{}';

CREATE UNIQUE INDEX idx_usuarios_identificador_funcional
    ON usuarios (identificador_funcional)
    WHERE identificador_funcional IS NOT NULL;

-- Mismo número de documento no duplicado por tipo (ej. dos DPI iguales)
CREATE UNIQUE INDEX idx_usuarios_tipo_numero_identificacion
    ON usuarios (tipo_identificacion, numero_identificacion)
    WHERE tipo_identificacion IS NOT NULL AND numero_identificacion IS NOT NULL;

COMMENT ON COLUMN usuarios.identificador_funcional IS 'Código corto de funcionario para actuaciones (distinto del UUID interno)';
COMMENT ON COLUMN usuarios.firma_grafica_storage_key IS 'Objeto en almacenamiento (PNG/SVG) de la firma manuscrita o rubrica';

-- Cada fila = una hoja del PDF (cara impresa) con su número de folio correlativo en el expediente
CREATE TABLE expediente_folio_hoja (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expediente_id   UUID NOT NULL REFERENCES expedientes (id) ON DELETE CASCADE,
    documento_id    UUID NOT NULL REFERENCES documentos (id) ON DELETE CASCADE,
    numero_hoja     INT NOT NULL CHECK (numero_hoja >= 1),
    folio_numero    BIGINT NOT NULL CHECK (folio_numero >= 1),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (expediente_id, folio_numero),
    UNIQUE (documento_id, numero_hoja)
);

CREATE INDEX idx_folio_hoja_expediente ON expediente_folio_hoja (expediente_id, folio_numero);
CREATE INDEX idx_folio_hoja_documento ON expediente_folio_hoja (documento_id);

COMMENT ON TABLE expediente_folio_hoja IS 'Correlación expediente: cada hoja del documento tiene un folio; el render estampa folio arriba-derecha y rúbricas en márgenes por hoja';
COMMENT ON COLUMN expediente_folio_hoja.numero_hoja IS 'Orden de la hoja dentro del PDF (1 = primera hoja del documento)';

-- Historial de reordenaciones (quién, cuándo, snapshot opcional)
CREATE TABLE folio_reorden_eventos (
    id              BIGSERIAL PRIMARY KEY,
    expediente_id   UUID NOT NULL REFERENCES expedientes (id) ON DELETE CASCADE,
    actor_id        UUID REFERENCES usuarios (id),
    motivo          TEXT,
    detalle         JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_folio_reorden_expediente ON folio_reorden_eventos (expediente_id, created_at DESC);
