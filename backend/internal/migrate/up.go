package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/firmaelectronica/expedientes-oj/migrations"
)

const advisoryLockKey int64 = 0x6578706f6a // "expoj" — evita dos réplicas a la vez

// Up aplica los *.up.sql embebidos que aún no consten en schema_migrations.
func Up(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("adquirir conexión: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockKey); err != nil {
		return fmt.Errorf("lock migraciones: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, advisoryLockKey)
	}()

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrations.UpSQL, ".")
	if err != nil {
		return fmt.Errorf("listar migraciones: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".up.sql") {
			names = append(names, n)
		}
	}
	sort.Strings(names)

	for _, name := range names {
		version := strings.SplitN(name, "_", 2)[0]
		if version == "" {
			continue
		}
		var n int
		if err := conn.QueryRow(ctx,
			`SELECT COUNT(*) FROM schema_migrations WHERE version = $1`, version,
		).Scan(&n); err != nil {
			return fmt.Errorf("consultar versión %s: %w", version, err)
		}
		if n > 0 {
			continue
		}

		sqlBytes, err := fs.ReadFile(migrations.UpSQL, name)
		if err != nil {
			return fmt.Errorf("leer %s: %w", name, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("ejecutar %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("marcar %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}
	}

	return nil
}
