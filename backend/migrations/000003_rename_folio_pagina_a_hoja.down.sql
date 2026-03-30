-- Reversión best-effort (solo si existía el nombre antiguo tras migrar con 000003)
DO $$
BEGIN
    IF to_regclass('public.expediente_folio_hoja') IS NOT NULL
       AND to_regclass('public.expediente_folio_pagina') IS NULL THEN
        ALTER TABLE expediente_folio_hoja RENAME COLUMN numero_hoja TO pagina;
        ALTER TABLE expediente_folio_hoja RENAME TO expediente_folio_pagina;
        IF to_regclass('public.idx_folio_hoja_expediente') IS NOT NULL THEN
            ALTER INDEX idx_folio_hoja_expediente RENAME TO idx_folio_pagina_expediente;
        END IF;
        IF to_regclass('public.idx_folio_hoja_documento') IS NOT NULL THEN
            ALTER INDEX idx_folio_hoja_documento RENAME TO idx_folio_pagina_documento;
        END IF;
    END IF;
END $$;
