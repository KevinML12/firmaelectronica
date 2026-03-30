package folio

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReordenarFolios aplica un nuevo orden total de HOJAS del expediente: los UUID en
// orderedFolioHojaIDs deben ser exactamente las filas de expediente_folio_hoja del
// expediente, en el orden deseado (primera posición = folio 1, etc.).
func ReordenarFolios(ctx context.Context, pool *pgxpool.Pool, expedienteID string, orderedFolioHojaIDs []string, actorID *string, motivo string) error {
	if len(orderedFolioHojaIDs) == 0 {
		return fmt.Errorf("lista de ids vacía")
	}
	seen := make(map[string]struct{}, len(orderedFolioHojaIDs))
	for _, id := range orderedFolioHojaIDs {
		if _, ok := seen[id]; ok {
			return fmt.Errorf("id duplicado en la lista: %s", id)
		}
		seen[id] = struct{}{}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var n int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM expediente_folio_hoja WHERE expediente_id = $1::uuid
	`, expedienteID).Scan(&n)
	if err != nil {
		return fmt.Errorf("contar hojas-folio: %w", err)
	}
	if n != len(orderedFolioHojaIDs) {
		return fmt.Errorf("se esperaban %d ids, se recibieron %d", n, len(orderedFolioHojaIDs))
	}

	snapshot, err := snapshotOrden(ctx, tx, expedienteID)
	if err != nil {
		return err
	}
	snapJSON, _ := json.Marshal(snapshot)

	for _, id := range orderedFolioHojaIDs {
		var ok bool
		err = tx.QueryRow(ctx, `
			SELECT true FROM expediente_folio_hoja
			WHERE id = $1::uuid AND expediente_id = $2::uuid
		`, id, expedienteID).Scan(&ok)
		if err == pgx.ErrNoRows {
			return fmt.Errorf("id %s no pertenece al expediente", id)
		}
		if err != nil {
			return fmt.Errorf("validar id: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE expediente_folio_hoja
		SET folio_numero = -folio_numero, updated_at = now()
		WHERE expediente_id = $1::uuid
	`, expedienteID)
	if err != nil {
		return fmt.Errorf("fase 1 reorden: %w", err)
	}

	for i, id := range orderedFolioHojaIDs {
		newFolio := int64(i + 1)
		_, err = tx.Exec(ctx, `
			UPDATE expediente_folio_hoja
			SET folio_numero = $3, updated_at = now()
			WHERE id = $1::uuid AND expediente_id = $2::uuid
		`, id, expedienteID, newFolio)
		if err != nil {
			return fmt.Errorf("fase 2 reorden folio %d: %w", newFolio, err)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE expediente_folio_contador
		SET ultimo_folio = (SELECT COALESCE(MAX(folio_numero), 0) FROM expediente_folio_hoja WHERE expediente_id = $1::uuid),
		    updated_at = now()
		WHERE expediente_id = $1::uuid
	`, expedienteID)
	if err != nil {
		return fmt.Errorf("sincronizar contador: %w", err)
	}

	if err := sincronizarAsignaciones(ctx, tx, expedienteID); err != nil {
		return err
	}

	detalle := map[string]json.RawMessage{
		"orden_anterior": snapJSON,
	}
	detBytes, _ := json.Marshal(detalle)

	var actor any
	if actorID != nil && *actorID != "" {
		actor = *actorID
	}
	var motivoAny any
	if motivo != "" {
		motivoAny = motivo
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO folio_reorden_eventos (expediente_id, actor_id, motivo, detalle)
		VALUES ($1::uuid, $2::uuid, $3, $4::jsonb)
	`, expedienteID, actor, motivoAny, string(detBytes))
	if err != nil {
		return fmt.Errorf("auditoría reorden: %w", err)
	}

	return tx.Commit(ctx)
}

type folioSnapshotRow struct {
	ID          string `json:"id"`
	DocumentoID string `json:"documento_id"`
	NumeroHoja  int    `json:"numero_hoja"`
	FolioNumero int64  `json:"folio_numero"`
}

func snapshotOrden(ctx context.Context, tx pgx.Tx, expedienteID string) ([]folioSnapshotRow, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, documento_id::text, numero_hoja, folio_numero
		FROM expediente_folio_hoja
		WHERE expediente_id = $1::uuid
		ORDER BY folio_numero
	`, expedienteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []folioSnapshotRow
	for rows.Next() {
		var r folioSnapshotRow
		if err := rows.Scan(&r.ID, &r.DocumentoID, &r.NumeroHoja, &r.FolioNumero); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func sincronizarAsignaciones(ctx context.Context, tx pgx.Tx, expedienteID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE asignacion_folios a
		SET folio_inicio = s.mn, folio_fin = s.mx
		FROM (
			SELECT documento_id, MIN(folio_numero) AS mn, MAX(folio_numero) AS mx
			FROM expediente_folio_hoja
			WHERE expediente_id = $1::uuid
			GROUP BY documento_id
		) s
		WHERE a.expediente_id = $1::uuid AND a.documento_id = s.documento_id
	`, expedienteID)
	if err != nil {
		return fmt.Errorf("sincronizar asignacion_folios: %w", err)
	}
	return nil
}
