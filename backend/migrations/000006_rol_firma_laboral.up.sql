-- Solo ampliar el enum (PG exige commit antes de usar valores nuevos → INSERT va en 000007).
ALTER TYPE rol_firma_oj ADD VALUE 'parte_actora';
ALTER TYPE rol_firma_oj ADD VALUE 'patrono_abogado';
ALTER TYPE rol_firma_oj ADD VALUE 'inspectora_trabajo';
ALTER TYPE rol_firma_oj ADD VALUE 'representante_demandada';
ALTER TYPE rol_firma_oj ADD VALUE 'magistrado';
ALTER TYPE rol_firma_oj ADD VALUE 'notificador';
ALTER TYPE rol_firma_oj ADD VALUE 'ministro_ejecutor';
