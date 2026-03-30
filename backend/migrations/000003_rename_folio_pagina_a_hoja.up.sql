-- Si ya aplicaste una versión anterior de 000002 con expediente_folio_pagina / pagina
DO $$
BEGIN
    IF to_regclass('public.expediente_folio_pagina') IS NOT NULL
       AND to_regclass('public.expediente_folio_hoja') IS NULL THEN
        ALTER TABLE expediente_folio_pagina RENAME TO expediente_folio_hoja;
        ALTER TABLE expediente_folio_hoja RENAME COLUMN pagina TO numero_hoja;
        IF to_regclass('public.idx_folio_pagina_expediente') IS NOT NULL THEN
            ALTER INDEX idx_folio_pagina_expediente RENAME TO idx_folio_hoja_expediente;
        END IF;
        IF to_regclass('public.idx_folio_pagina_documento') IS NOT NULL THEN
            ALTER INDEX idx_folio_pagina_documento RENAME TO idx_folio_hoja_documento;
        END IF;
    END IF;
END $$;
