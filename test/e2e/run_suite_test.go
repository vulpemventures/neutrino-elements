package e2etest

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestE2E(t *testing.T) {
	suite.Run(t, new(E2ESuite))
}
