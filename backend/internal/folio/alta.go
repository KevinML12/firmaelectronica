package folio

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AltaDocumento registra un documento en el expediente: reserva correlativo por HOJA,
// una fila en expediente_folio_hoja por cada hoja del PDF, y asignacion_folios.
// numHojas = número de hojas del PDF (cada hoja lleva un folio correlativo).
func AltaDocumento(ctx context.Context, pool *pgxpool.Pool, expedienteID, documentoID string, numHojas int) (inicio, fin int64, err error) {
	if numHojas < 1 {
		return 0, 0, fmt.Errorf("numHojas debe ser >= 1")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var ultimo int64
	err = tx.QueryRow(ctx, `
		SELECT ultimo_folio FROM expediente_folio_contador
		WHERE expediente_id = $1::uuid
		FOR UPDATE
	`, expedienteID).Scan(&ultimo)
	if err != nil {
		return 0, 0, fmt.Errorf("leer contador: %w", err)
	}

	inicio = ultimo + 1
	fin = ultimo + int64(numHojas)

	for h := 1; h <= numHojas; h++ {
		folio := inicio + int64(h-1)
		_, err = tx.Exec(ctx, `
			INSERT INTO expediente_folio_hoja (expediente_id, documento_id, numero_hoja, folio_numero)
			VALUES ($1::uuid, $2::uuid, $3, $4)
		`, expedienteID, documentoID, h, folio)
		if err != nil {
			return 0, 0, fmt.Errorf("insertar mapeo folio-hoja %d: %w", h, err)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE expediente_folio_contador
		SET ultimo_folio = $2, updated_at = now()
		WHERE expediente_id = $1::uuid
	`, expedienteID, fin)
	if err != nil {
		return 0, 0, fmt.Errorf("actualizar contador: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO asignacion_folios (documento_id, expediente_id, folio_inicio, folio_fin)
		VALUES ($1::uuid, $2::uuid, $3, $4)
	`, documentoID, expedienteID, inicio, fin)
	if err != nil {
		return 0, 0, fmt.Errorf("insertar asignacion_folios: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return inicio, fin, nil
}
