package folio

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AsignarReserva solo actualiza el contador y no crea filas en expediente_folio_hoja.
// Para el flujo normal (estampado y reordenación), usar AltaDocumento.
func AsignarReserva(ctx context.Context, pool *pgxpool.Pool, expedienteID string, numHojas int) (inicio, fin int64, err error) {
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

	_, err = tx.Exec(ctx, `
		UPDATE expediente_folio_contador
		SET ultimo_folio = $2, updated_at = now()
		WHERE expediente_id = $1::uuid
	`, expedienteID, fin)
	if err != nil {
		return 0, 0, fmt.Errorf("actualizar contador: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return inicio, fin, nil
}
