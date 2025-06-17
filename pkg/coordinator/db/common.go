package db

import (
	"embed"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/sirupsen/logrus"

	// sql backend drivers
	_ "github.com/glebarez/go-sqlite"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
)

//go:embed schema/pgsql/*.sql
var EmbedPgsqlSchema embed.FS

//go:embed schema/sqlite/*.sql
var EmbedSqliteSchema embed.FS

type DatabaseConfig struct {
	Engine string                `yaml:"engine"`
	Sqlite *SqliteDatabaseConfig `yaml:"sqlite"`
	Pgsql  *PgsqlDatabaseConfig  `yaml:"pgsql"`
}

type EngineType int

const (
	EngineAny    EngineType = 0
	EngineSqlite EngineType = 1
	EnginePgsql  EngineType = 2
)

type Database struct {
	engine      EngineType
	logger      logrus.FieldLogger
	reader      *sqlx.DB
	writer      *sqlx.DB
	writerMutex sync.Mutex
}

func NewDatabase(logger logrus.FieldLogger) *Database {
	return &Database{
		logger: logger,
	}
}

func (db *Database) checkDBConn(dbConn *sqlx.DB, dataBaseName string) {
	// The golang sql driver does not properly implement PingContext
	// therefore we use a timer to catch db connection timeouts
	dbConnectionTimeout := time.NewTimer(15 * time.Second)

	go func() {
		<-dbConnectionTimeout.C
		db.logger.Fatalf("timeout while connecting to %s", dataBaseName)
	}()

	err := dbConn.Ping()
	if err != nil {
		db.logger.Fatalf("unable to Ping %s: %s", dataBaseName, err)
	}

	dbConnectionTimeout.Stop()
}

type SqliteDatabaseConfig struct {
	File         string `yaml:"file"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

func (db *Database) initSqlite(config *SqliteDatabaseConfig) error {
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 50
	}

	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 10
	}

	if config.MaxOpenConns < config.MaxIdleConns {
		config.MaxIdleConns = config.MaxOpenConns
	}

	db.logger.Infof("initializing sqlite connection to %v with %v/%v conn limit", config.File, config.MaxIdleConns, config.MaxOpenConns)

	dbConn, err := sqlx.Open("sqlite", fmt.Sprintf("%s?_pragma=journal_mode(WAL)", config.File))
	if err != nil {
		return fmt.Errorf("error opening sqlite database: %v", err)
	}

	db.checkDBConn(dbConn, "database")
	dbConn.SetConnMaxIdleTime(0)
	dbConn.SetConnMaxLifetime(0)
	dbConn.SetMaxOpenConns(config.MaxOpenConns)
	dbConn.SetMaxIdleConns(config.MaxIdleConns)

	dbConn.MustExec("PRAGMA journal_mode = WAL")

	db.reader = dbConn
	db.writer = dbConn

	return nil
}

type PgsqlDatabaseConfig struct {
	Host         string `yaml:"host"`
	Port         string `yaml:"port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

func (db *Database) initPgsql(config *PgsqlDatabaseConfig) error {
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 50
	}

	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 10
	}

	if config.MaxOpenConns < config.MaxIdleConns {
		config.MaxIdleConns = config.MaxOpenConns
	}

	db.logger.Infof("initializing pgsql writer connection to %v with %v/%v conn limit", config.Host, config.MaxIdleConns, config.MaxOpenConns)

	dbConn, err := sqlx.Open("pgx", fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", config.Username, config.Password, config.Host, config.Port, config.Database))
	if err != nil {
		return fmt.Errorf("error getting pgsql writer database: %v", err)
	}

	db.checkDBConn(dbConn, "database")
	dbConn.SetConnMaxIdleTime(time.Second * 30)
	dbConn.SetConnMaxLifetime(time.Second * 60)
	dbConn.SetMaxOpenConns(config.MaxOpenConns)
	dbConn.SetMaxIdleConns(config.MaxIdleConns)

	db.writer = dbConn
	db.reader = dbConn

	return nil
}

func (db *Database) InitDB(config *DatabaseConfig) error {
	switch config.Engine {
	case "sqlite":
		err := db.initSqlite(config.Sqlite)
		if err != nil {
			return err
		}

		db.engine = EngineSqlite

	case "pgsql":
		err := db.initPgsql(config.Pgsql)
		if err != nil {
			return err
		}

		db.engine = EnginePgsql

	default:
		return fmt.Errorf("unknown database engine type: %s", config.Engine)
	}

	return nil
}

func (db *Database) CloseDB() error {
	err := db.writer.Close()
	if err != nil {
		db.logger.Errorf("Error closing writer db connection: %v", err)
	}

	err = db.writer.Close()
	if err != nil {
		db.logger.Errorf("Error closing reader db connection: %v", err)
	}

	return nil
}

func (db *Database) RunTransaction(handler func(tx *sqlx.Tx) error) error {
	if db.engine == EngineSqlite {
		db.writerMutex.Lock()
		defer db.writerMutex.Unlock()
	}

	tx, err := db.writer.Beginx()
	if err != nil {
		return fmt.Errorf("error starting db transactions: %v", err)
	}

	defer func() {
		//nolint:errcheck // ignore error
		tx.Rollback()
	}()

	err = handler(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error committing db transaction: %v", err)
	}

	return err
}

func (db *Database) ApplySchema(version int64) error {
	var engineDialect, schemaDirectory string

	switch db.engine {
	case EnginePgsql:
		goose.SetBaseFS(EmbedPgsqlSchema)

		engineDialect = "postgres"
		schemaDirectory = "schema/pgsql"

	case EngineSqlite:
		goose.SetBaseFS(EmbedSqliteSchema)

		engineDialect = "sqlite3"
		schemaDirectory = "schema/sqlite"

	case EngineAny:
		db.logger.Fatalf("database engine not initialized")
	default:
		db.logger.Fatalf("unknown database engine")
	}

	if err := goose.SetDialect(engineDialect); err != nil {
		return err
	}

	switch version {
	case -2:
		if err := goose.Up(db.writer.DB, schemaDirectory, goose.WithAllowMissing()); err != nil {
			return err
		}
	case -1:
		if err := goose.UpByOne(db.writer.DB, schemaDirectory, goose.WithAllowMissing()); err != nil {
			return err
		}
	default:
		if err := goose.UpTo(db.writer.DB, schemaDirectory, version, goose.WithAllowMissing()); err != nil {
			return err
		}
	}

	return nil
}

func (db *Database) EngineQuery(queryMap map[EngineType]string) string {
	if queryMap[db.engine] != "" {
		return queryMap[db.engine]
	}

	return queryMap[EngineAny]
}
