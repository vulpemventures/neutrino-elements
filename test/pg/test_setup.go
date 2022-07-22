package pgtest

import (
	"context"
	"github.com/stretchr/testify/suite"
	dbpg "github.com/vulpemventures/neutrino-elements/internal/infrastructure/storage/db/pg"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

var (
	dbSvc      *dbpg.DbService
	filterRepo repository.FilterRepository
	headerRepo repository.BlockHeaderRepository

	ctx = context.Background()
)

type PgDbTestSuite struct {
	suite.Suite
}

func (s *PgDbTestSuite) SetupSuite() {
	d, err := dbpg.NewDbService(dbpg.DbConfig{
		DbUser:     "root",
		DbPassword: "secret",
		DbHost:     "127.0.0.1",
		DbPort:     5432,
		DbName:     "neutrino-elements-test",
		MigrationSourceURL: "file://../.." +
			"/internal/infrastructure/storage/db/pg/migration",
	})
	if err != nil {
		s.FailNow(err.Error())
	}
	dbSvc = d

	if dbSvc != nil {
		err := dbSvc.CreateLoader(dbSvc.Db.DB)
		if err != nil {
			s.FailNow(err.Error())
		}
	}

	fr, err := dbpg.NewFilterRepositoryImpl(dbSvc)
	if err != nil {
		s.FailNow(err.Error())
	}
	filterRepo = fr

	hr, err := dbpg.NewHeaderRepositoryImpl(dbSvc)
	if err != nil {
		s.FailNow(err.Error())
	}
	headerRepo = hr
}

func (s *PgDbTestSuite) TearDownSuite() {
	err := dbSvc.Db.Close()
	if err != nil {
		s.FailNow(err.Error())
	}
}

func (s *PgDbTestSuite) BeforeTest(suiteName, testName string) {
	if err := dbSvc.LoadFixtures(); err != nil {
		s.FailNow(err.Error())
	}
}

func (s *PgDbTestSuite) AfterTest(suiteName, testName string) {

}
