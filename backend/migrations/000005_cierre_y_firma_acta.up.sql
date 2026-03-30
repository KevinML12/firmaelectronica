ALTER TABLE expedientes
    ADD COLUMN IF NOT EXISTS cerrado_en TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS checklist JSONB NOT NULL DEFAULT '{}';

COMMENT ON COLUMN expedientes.checklist IS 'Progreso UI: subido, procesado, firmado, etc.';

ALTER TABLE firmas_documento
    ADD COLUMN IF NOT EXISTS nombre_acta TEXT;

COMMENT ON COLUMN firmas_documento.nombre_acta IS 'Nombre que se imprime en el acta si difiere del usuario';

-- Usuarios demo para firmar sin login (mapeo por rol en API)
INSERT INTO usuarios (id, nombre_completo, rol_firma, activo)
VALUES
    ('33333333-3333-4333-8333-333333333301'::uuid, 'Juez (demostración)', 'juez', true),
    ('33333333-3333-4333-8333-333333333302'::uuid, 'Secretario (demostración)', 'secretario', true),
    ('33333333-3333-4333-8333-333333333303'::uuid, 'Oficial V (demostración)', 'oficial_v', true)
ON CONFLICT (id) DO NOTHING;
