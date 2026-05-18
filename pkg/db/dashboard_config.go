package db

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

// dashboardConfigKey is the singleton row used to store the active
// dashboard config. The table is keyed in case we ever want
// per-user or named dashboards.
const dashboardConfigKey = "default"

type DashboardConfig struct {
	Key       string `db:"key"`
	Data      []byte `db:"data"`
	UpdatedAt int64  `db:"updated_at"`
}

// GetDashboardConfig returns the persisted dashboard config blob, or
// (nil, nil) when no config has been saved yet.
func (db *Database) GetDashboardConfig() (*DashboardConfig, error) {
	var cfg DashboardConfig

	err := db.reader.Get(&cfg, `
		SELECT key, data, updated_at FROM dashboard_config
		WHERE key = $1`,
		dashboardConfigKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &cfg, nil
}

// UpsertDashboardConfig stores `data` as the active dashboard config.
// The blob is opaque to the server — clients are responsible for
// validating the schema.
func (db *Database) UpsertDashboardConfig(data []byte) error {
	return db.RunTransaction(func(tx *sqlx.Tx) error {
		_, err := tx.Exec(db.EngineQuery(map[EngineType]string{
			EnginePgsql: `
				INSERT INTO dashboard_config (key, data, updated_at)
				VALUES ($1, $2, $3)
				ON CONFLICT (key) DO UPDATE SET
					data = excluded.data,
					updated_at = excluded.updated_at`,
			EngineSqlite: `
				INSERT OR REPLACE INTO dashboard_config (key, data, updated_at)
				VALUES ($1, $2, $3)`,
		}),
			dashboardConfigKey, data, time.Now().Unix())

		return err
	})
}
