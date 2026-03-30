-- Expediente y juzgado fijos para desarrollo / demo (IDs estables para el frontend)
INSERT INTO juzgados (id, codigo, nombre, departamento, municipio)
VALUES (
    '11111111-1111-4111-8111-111111111111'::uuid,
    'DEMO-SEED',
    'Juzgado de Primera Instancia (demostración)',
    'Huehuetenango',
    'Huehuetenango'
)
ON CONFLICT (codigo) DO NOTHING;

INSERT INTO expedientes (id, numero_unico, juzgado_id, tipo_proceso, estado)
SELECT
    '22222222-2222-4222-8222-222222222222'::uuid,
    '13002-2025-00341',
    j.id,
    'Constitucional Amparo',
    'activo'
FROM juzgados j
WHERE j.codigo = 'DEMO-SEED'
ON CONFLICT (juzgado_id, numero_unico) DO NOTHING;
