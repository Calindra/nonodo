package rollup

import (
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
)

//
// Test Suite
//

type RollupSuite struct {
	suite.Suite
	echo *echo.Echo
}

func (s *RollupSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
}

func TestRollupSuite(t *testing.T) {
	suite.Run(t, new(RollupSuite))
}

// func (s *RollupSuite) TestFetcher() {}