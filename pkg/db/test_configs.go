package db

import (
	"github.com/jmoiron/sqlx"
)

type TestConfig struct {
	TestID           string `db:"test_id"`
	Source           string `db:"source"`
	Name             string `db:"name"`
	Timeout          int    `db:"timeout"`
	Config           string `db:"config"`
	ConfigVars       string `db:"config_vars"`
	ScheduleStartup  bool   `db:"schedule_startup"`
	ScheduleCronYaml string `db:"schedule_cron_yaml"`
	YamlSource       string `db:"yaml_source"`
}

// InsertTestConfig inserts a test config into the database.
func (db *Database) InsertTestConfig(tx *sqlx.Tx, config *TestConfig) error {
	_, err := tx.Exec(db.EngineQuery(map[EngineType]string{
		EnginePgsql: `
			INSERT INTO test_configs (
				test_id, source, name, timeout, config, config_vars, schedule_startup, schedule_cron_yaml, yaml_source
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (test_id) DO UPDATE SET
				source = excluded.source,
				name = excluded.name,
				timeout = excluded.timeout,
				config = excluded.config,
				config_vars = excluded.config_vars,
				schedule_startup = excluded.schedule_startup,
				schedule_cron_yaml = excluded.schedule_cron_yaml,
				yaml_source = excluded.yaml_source`,
		EngineSqlite: `
			INSERT OR REPLACE INTO test_configs (
				test_id, source, name, timeout, config, config_vars, schedule_startup, schedule_cron_yaml, yaml_source
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
	}),
		config.TestID, config.Source, config.Name, config.Timeout, config.Config, config.ConfigVars,
		config.ScheduleStartup, config.ScheduleCronYaml, config.YamlSource)
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

type TestRunStats struct {
	TestID  string `db:"test_id"`
	Count   int    `db:"count"`
	LastRun uint64 `db:"last_run"`
}

// GetTestRunStats returns the test run stats for all tests.
func (db *Database) GetTestRunStats() ([]*TestRunStats, error) {
	var stats []*TestRunStats

	err := db.reader.Select(&stats, `SELECT test_id, COUNT(*) AS count, MAX(start_time) AS last_run FROM test_runs GROUP BY test_id`)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetTestConfig returns a single test config by ID.
func (db *Database) GetTestConfig(testID string) (*TestConfig, error) {
	var config TestConfig

	err := db.reader.Get(&config, `SELECT * FROM test_configs WHERE test_id = $1`, testID)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// DeleteTestConfig deletes a test config from the database.
func (db *Database) DeleteTestConfig(tx *sqlx.Tx, testID string) error {
	_, err := tx.Exec(`DELETE FROM test_configs WHERE test_id = $1`, testID)
	if err != nil {
		return err
	}

	return nil
}
