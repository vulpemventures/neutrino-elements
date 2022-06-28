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
}

func (e *E2ESuite) BeforeTest(suiteName, testName string) {

}

func (e *E2ESuite) AfterTest(suiteName, testName string) {

}
