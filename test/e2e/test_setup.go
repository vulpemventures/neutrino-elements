package e2etest

import (
	"github.com/stretchr/testify/suite"
	"github.com/vulpemventures/neutrino-elements/pkg/testutil"
	"os/exec"
	"time"
)

var (
	neutrinod *exec.Cmd
)

type E2ESuite struct {
	suite.Suite
}

func (e *E2ESuite) SetupSuite() {
	if err := testutil.SetupDB(); err != nil {
		e.FailNow(err.Error())
	}

	if err := testutil.TruncateDB(); err != nil {
		e.FailNow(err.Error())
	}

	e.T().Setenv("NEUTRINO_ELEMENTS_DB_NAME", "neutrino-elements-test")
	e.T().Setenv("NEUTRINO_ELEMENTS_DB_MIGRATION_PATH", "file://../../internal/infrastructure/storage/db/pg/migration")

	n, err := testutil.RunCommandDetached("../../bin/neutrinod")
	if err != nil {
		e.FailNow(err.Error())
	}

	neutrinod = n
	time.Sleep(time.Second * 3)
}

func (e *E2ESuite) TearDownSuite() {
	if err := neutrinod.Process.Kill(); err != nil {
		e.FailNow(err.Error())
	}

	if err := testutil.ShutdownDB(); err != nil {
		e.FailNow(err.Error())
	}
}

func (e *E2ESuite) BeforeTest(suiteName, testName string) {

}

func (e *E2ESuite) AfterTest(suiteName, testName string) {

}
