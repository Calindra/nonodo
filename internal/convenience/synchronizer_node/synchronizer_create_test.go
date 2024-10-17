package synchronizernode

import (
	"context"
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/stretchr/testify/suite"
)

type SynchorizerNodeSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *SynchorizerNodeSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.ctx = context.Background()

	err := raw.RunDockerCompose(s.ctx)
	s.NoError(err)
}

func (s *SynchorizerNodeSuite) TearDownTest() {
	err := raw.StopDockerCompose(s.ctx)
	s.NoError(err)
}

func TestSynchronizerNodeSuite(t *testing.T) {
	suite.Run(t, new(SynchorizerNodeSuite))
}

func (s *SynchorizerNodeSuite) TestSynchronizerNodeConnection() {
	s.Equal(4, 2+2)
}
