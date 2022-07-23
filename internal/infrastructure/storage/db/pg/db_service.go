package dbpg

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/golang-migrate/migrate/v4"
	"sync"

	"github.com/golang-migrate/migrate/v4/database/postgres"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	postgresDriver             = "postgres"
	insecureDataSourceTemplate = "postgresql://%s:%s@%s:%d/%s?sslmode=disable"
	dataSourceTemplate         = "host=%s port=%d user=%s password=%s dbname=%s"
	postgresDialect            = "postgres"
	fixturesPath               = "../fixtures"
)

var (
	txContextKey contextKey = struct{}{}
)

type contextKey struct{}

type DbService struct {
	Db             *sqlx.DB
	fixturesLoader *testfixtures.Loader
	mutex          *sync.RWMutex
}

type DbConfig struct {
	DbUser             string
	DbPassword         string
	DbHost             string
	DbPort             int
	DbName             string
	MigrationSourceURL string
}

func NewDbService(dbConfig DbConfig) (*DbService, error) {
	db, err := connect(dbConfig)
	if err != nil {
		return nil, err
	}

	if err = migrateDb(db.DB, dbConfig.MigrationSourceURL); err != nil {
		return nil, err
	}

	return &DbService{
		Db: db,
	}, nil
}

func connect(dbConfig DbConfig) (*sqlx.DB, error) {
	dataSource := insecureDataSourceStr(dbConfig)

	db, err := sqlx.Connect(
		postgresDriver,
		dataSource,
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func migrateDb(db *sql.DB, migrationSourceUrl string) error {
	dbInstance, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationSourceUrl,
		postgresDriver,
		dbInstance,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

// insecureDataSourceStr converts database configuration params to connection string
func insecureDataSourceStr(dbConfig DbConfig) string {
	return fmt.Sprintf(
		insecureDataSourceTemplate,
		dbConfig.DbUser,
		dbConfig.DbPassword,
		dbConfig.DbHost,
		dbConfig.DbPort,
		dbConfig.DbName,
	)
}

// CreateLoader creates loader that is to be used to load fixtures from given
//folder
func (d *DbService) CreateLoader(db *sql.DB) error {
	f, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect(postgresDialect),
		testfixtures.Directory(fixturesPath),
	)
	if err != nil {
		return err
	}

	d.fixturesLoader = f
	d.mutex = new(sync.RWMutex)

	return nil
}

func (d *DbService) LoadFixtures() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	err := d.fixturesLoader.Load()
	if err != nil {
		return err
	}

	return nil
}

func (d *DbService) SetCtxTx(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey, tx)
}

func (d *DbService) GetTxFromCtx(ctx context.Context) (*sqlx.Tx, error) {
	if tx, ok := ctx.Value(contextKey{}).(*sqlx.Tx); ok {
		return tx, nil
	}

	return d.Db.Beginx()
}
