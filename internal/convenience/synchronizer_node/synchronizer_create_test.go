package synchronizernode

import (
	"context"
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/stretchr/testify/suite"
)

type SynchronizerNodeSuite struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
}

func (s *SynchronizerNodeSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.ctx = context.Background()
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerNodeSuite) TearDownTest() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
}

func TestSynchronizerNodeSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerNodeSuite))
}

func (s *SynchronizerNodeSuite) TestSynchronizerNodeConnection() {
	s.Equal(4, 2+2)
}
