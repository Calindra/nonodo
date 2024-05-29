package rollup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const TestTimeout = 5 * time.Second

//
// Test Suite
//

type RollupSuite struct {
	suite.Suite
	ctx        context.Context
	cancel     context.CancelFunc
	rollupsAPI ServerInterface
	tempDir    string
	server     *echo.Echo
}

type SequencerMock struct {
	mock.Mock
}

// FinishAndGetNext implements Sequencer.
func (s *SequencerMock) FinishAndGetNext(accept bool) model.Input {
	panic("unimplemented")
}

func (s *RollupSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.ctx, s.cancel = context.WithTimeout(context.Background(), TestTimeout)
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	sqliteFileName := fmt.Sprintf("test_rollup%d.sqlite3", time.Now().UnixMilli())
	sqliteFileName = path.Join(tempDir, sqliteFileName)

	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	container := convenience.NewContainer(*db)
	decoder := container.GetOutputDecoder()
	nonodoModel := model.NewNonodoModel(decoder, db)

	// s.server = echo.New()
	// s.server.Use(middleware.Logger())
	// s.server.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
	// 	ErrorMessage: "Request timed out",
	// 	Timeout:      100 * time.Millisecond,
	// }))
	// Register(s.server, nonodoModel)
	sequencer := &SequencerMock{}
	s.rollupsAPI = &RollupAPI{model: nonodoModel, sequencer: sequencer}
}

func TestRollupSuite(t *testing.T) {
	suite.Run(t, new(RollupSuite))
}

func (s *RollupSuite) teardown() {
	// nothing to do
	// s.server.Close()
	select {
	case <-s.ctx.Done():
		s.T().Error(s.ctx.Err())
	default:
		s.cancel()
	}
}

func (s *RollupSuite) TestFetcher() {
	defer s.teardown()
	// ctx := s.server.AcquireContext()
	// defer s.server.ReleaseContext(ctx)
	// res := s.rollupsAPI.Gio(ctx)
	// s.NoError(res, "Gio should not return an error")
}
