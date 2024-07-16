package rollup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience"
	cModel "github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
func (s *SequencerMock) FinishAndGetNext(accept bool) cModel.Input {
	panic("unimplemented")
}

func (s *RollupSuite) SetupTest() {
	// Log
	commons.ConfigureLog(slog.LevelDebug)

	// Context
	s.ctx, s.cancel = context.WithTimeout(context.Background(), TestTimeout)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	// Database
	sqliteFileName := fmt.Sprintf("test_rollup%d.sqlite3", time.Now().UnixMilli())
	sqliteFileName = path.Join(tempDir, sqliteFileName)

	// NoNodoModel
	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	container := convenience.NewContainer(*db)
	decoder := container.GetOutputDecoder()
	nonodoModel := model.NewNonodoModel(decoder,
		container.GetReportRepository(),
		container.GetInputRepository(),
	)

	// Sequencer
	var sequencer model.Sequencer = &SequencerMock{}

	// Server
	s.server = echo.New()
	s.server.Use(middleware.Logger())
	s.server.Use(middleware.Recover())
	s.rollupsAPI = &RollupAPI{model: nonodoModel, sequencer: sequencer}
	RegisterHandlers(s.server, s.rollupsAPI)
}

func TestRollupSuite(t *testing.T) {
	suite.Run(t, new(RollupSuite))
}

func (s *RollupSuite) TearDownTest() {
	// nothing to do
	s.server.Close()
	select {
	case <-s.ctx.Done():
		s.T().Error(s.ctx.Err())
	default:
		s.cancel()
	}
}

func (s *RollupSuite) TestFetcherMissing() {
	gioJsonReqBody := GioJSONRequestBody{
		Domain: 0,
		Id:     "idontexist",
	}
	body, err := json.Marshal(gioJsonReqBody)
	s.NoError(err)
	bodyReader := bytes.NewReader(body)
	req := httptest.NewRequest(http.MethodGet, "/gio", bodyReader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.server.NewContext(req, rec)

	res := s.rollupsAPI.Gio(c)
	s.NoError(res, "Gio should not return an error")
	s.Assert().Equal(http.StatusBadRequest, rec.Result().StatusCode)
	s.Assert().Equal("Unsupported domain", rec.Body.String())
}
