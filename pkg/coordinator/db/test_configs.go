package db

import "github.com/jmoiron/sqlx"

type TestConfig struct {
	TestID           string `db:"test_id"`
	Source           string `db:"source"`
	Name             string `db:"name"`
	Timeout          int    `db:"timeout"`
	Config           string `db:"config"`
	ConfigVars       string `db:"config_vars"`
	ScheduleStartup  bool   `db:"schedule_startup"`
	ScheduleCronYaml string `db:"schedule_cron_yaml"`
}

// InsertTestConfig inserts a test config into the database.
func (db *Database) InsertTestConfig(tx *sqlx.Tx, config *TestConfig) error {
	_, err := tx.Exec(db.EngineQuery(map[EngineType]string{
		EnginePgsql: `
			INSERT INTO test_configs (
				test_id, source, name, timeout, config, config_vars, schedule_startup, schedule_cron_yaml
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (test_id) DO UPDATE SET
				source = excluded.source,
				name = excluded.name,
				timeout = excluded.timeout,
				config = excluded.config,
				config_vars = excluded.config_vars,
				schedule_startup = excluded.schedule_startup,
				schedule_cron_yaml = excluded.schedule_cron_yaml`,
		EngineSqlite: `
			INSERT OR REPLACE INTO test_configs (
				test_id, source, name, timeout, config, config_vars, schedule_startup, schedule_cron_yaml
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
	}),
		config.TestID, config.Source, config.Name, config.Timeout, config.Config, config.ConfigVars,
		config.ScheduleStartup, config.ScheduleCronYaml)
	if err != nil {
		return err
	}

	return nil
}

// GetTestConfigs returns all test configs.
func (db *Database) GetTestConfigs() ([]*TestConfig, error) {
	var configs []*TestConfig

	err := db.reader.Select(&configs, `SELECT * FROM test_configs`)
	if err != nil {
		return nil, err
	}

	return configs, nil
}
