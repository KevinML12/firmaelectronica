-- Roles de firma ampliados: expediente laboral (MTPS, partes, sala, notificador, etc.)
ALTER TYPE rol_firma_oj ADD VALUE 'parte_actora';
ALTER TYPE rol_firma_oj ADD VALUE 'patrono_abogado';
ALTER TYPE rol_firma_oj ADD VALUE 'inspectora_trabajo';
ALTER TYPE rol_firma_oj ADD VALUE 'representante_demandada';
ALTER TYPE rol_firma_oj ADD VALUE 'magistrado';
ALTER TYPE rol_firma_oj ADD VALUE 'notificador';
ALTER TYPE rol_firma_oj ADD VALUE 'ministro_ejecutor';

INSERT INTO usuarios (id, nombre_completo, rol_firma, activo)
VALUES
    ('33333333-3333-4333-8333-333333333304'::uuid, 'Parte actora / compareciente (demo)', 'parte_actora', true),
    ('33333333-3333-4333-8333-333333333305'::uuid, 'Abogado patrono (demo)', 'patrono_abogado', true),
    ('33333333-3333-4333-8333-333333333306'::uuid, 'Inspectora de Trabajo (demo)', 'inspectora_trabajo', true),
    ('33333333-3333-4333-8333-333333333307'::uuid, 'Representante demandada (demo)', 'representante_demandada', true),
    ('33333333-3333-4333-8333-333333333308'::uuid, 'Magistrado sala apelaciones (demo)', 'magistrado', true),
    ('33333333-3333-4333-8333-333333333309'::uuid, 'Notificador OJ (demo)', 'notificador', true),
    ('33333333-3333-4333-8333-333333333310'::uuid, 'Ministro ejecutor (demo)', 'ministro_ejecutor', true)
ON CONFLICT (id) DO NOTHING;
