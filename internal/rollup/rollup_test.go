package rollup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	cModel "github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience"
	"github.com/ethereum/go-ethereum/common"
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
	model      *model.NonodoModel
	container  *convenience.Container
}

type SequencerMock struct {
	mock.Mock
}

// FinishAndGetNext implements Sequencer.
func (s *SequencerMock) FinishAndGetNext(accept bool) (cModel.Input, error) {
	return nil, errors.New("finish and get next unimplemented")
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
	sqliteFileName = filepath.Join(tempDir, sqliteFileName)

	// NoNodoModel
	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	container := convenience.NewContainer(*db, false)
	decoder := container.GetOutputDecoder()
	nonodoModel := model.NewNonodoModel(decoder,
		container.GetReportRepository(),
		container.GetInputRepository(),
		container.GetVoucherRepository(),
		container.GetNoticeRepository(),
	)

	// Sequencer
	var sequencer model.Sequencer = model.NewInputBoxSequencer(nonodoModel)

	// Server
	s.server = echo.New()
	s.server.Use(middleware.Logger())
	s.server.Use(middleware.Recover())
	s.rollupsAPI = &RollupAPI{model: nonodoModel, sequencer: sequencer}
	s.model = nonodoModel
	s.container = container
	RegisterHandlers(s.server, s.rollupsAPI)
}

func TestRollupSuite(t *testing.T) {
	suite.Run(t, new(RollupSuite))
}

func (s *RollupSuite) TearDownTest() {
	defer os.RemoveAll(s.tempDir)

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

func (s *RollupSuite) TestEncodeVoucher() {

	abiParsed, err := contracts.OutputsMetaData.GetAbi()
	s.NoError(err)
	destination := common.HexToAddress(devnet.ApplicationAddress)
	valueHex := "0x0000000000000000000000000000000000000000000000000000000000000002"
	payloadHex := "0xdeadbeef"
	payload := common.Hex2Bytes(payloadHex[2:])
	value := new(big.Int)
	value, ok := value.SetString(valueHex[2:], 16)
	s.True(ok)
	s.addNewAdvanceInput(1)
	s.addNewAdvanceInput(2)
	s.hitFinish()
	voucherJsonReqBody := AddVoucherJSONRequestBody{
		Destination: destination.Hex(),
		Payload:     payloadHex,
		Value:       valueHex,
	}
	body, err := json.Marshal(voucherJsonReqBody)
	s.NoError(err)
	bodyReader := bytes.NewReader(body)
	req := httptest.NewRequest(http.MethodPost, "/voucher", bodyReader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.server.NewContext(req, rec)

	res := s.rollupsAPI.AddVoucher(c)
	s.NoError(res, "AddVoucher should not return an error")
	s.Assert().Equal(http.StatusOK, rec.Result().StatusCode)

	encoded, err := abiParsed.Methods["Voucher"].Inputs.Pack(destination, value, payload)
	s.NoError(err)

	values, err := abiParsed.Methods["Voucher"].Inputs.Unpack(encoded[:])
	s.NoError(err)

	deb, err := abiParsed.Pack("Voucher", destination, value, payload)
	s.NoError(err)
	slog.Debug("encoded", "encoded", common.Bytes2Hex(deb))
	expectedAddress := devnet.ApplicationAddress
	s.Equal(expectedAddress, values[0].(common.Address).Hex())
	s.Equal(value, values[1].(*big.Int))
	s.Equal(payload, values[2])

	s.hitFinish()

	ctx := context.Background()
	vouchersResp, err := s.container.GetVoucherRepository().FindAllVouchers(
		ctx, nil, nil, nil, nil, nil,
	)
	s.NoError(err)
	s.Equal(fmt.Sprintf("0x%s", common.Bytes2Hex(deb)), vouchersResp.Rows[0].Payload)
}

func (s *RollupSuite) TestEncodeNotice() {
	abiParsed, err := contracts.OutputsMetaData.GetAbi()
	s.NoError(err)
	payloadHex := "0xdeadbeef"
	payload := common.Hex2Bytes(payloadHex[2:])
	s.addNewAdvanceInput(1)
	s.addNewAdvanceInput(2)
	s.hitFinish()
	voucherJsonReqBody := AddNoticeJSONRequestBody{
		Payload: payloadHex,
	}
	body, err := json.Marshal(voucherJsonReqBody)
	s.NoError(err)
	bodyReader := bytes.NewReader(body)
	req := httptest.NewRequest(http.MethodPost, "/notice", bodyReader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := s.server.NewContext(req, rec)

	res := s.rollupsAPI.AddNotice(c)
	s.NoError(res, "AddNotice should not return an error")
	s.Assert().Equal(http.StatusOK, rec.Result().StatusCode)

	deb, err := abiParsed.Pack("Notice", payload)
	s.NoError(err)
	slog.Debug("encoded", "encoded", common.Bytes2Hex(deb))
	s.hitFinish()

	ctx := context.Background()
	noticesResp, err := s.container.GetNoticeRepository().FindAllNotices(
		ctx, nil, nil, nil, nil, nil,
	)
	s.NoError(err)
	s.Equal(fmt.Sprintf("0x%s", common.Bytes2Hex(deb)), noticesResp.Rows[0].Payload)
}

func (s *RollupSuite) addNewAdvanceInput(inputBoxIndex int) {
	destination := common.HexToAddress("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e")
	payloadHex := "0xdeadbeef"
	payload := payloadHex[2:]
	err := s.model.AddAdvanceInput(
		common.HexToAddress("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e"),
		payload, uint64(1), time.Now(), inputBoxIndex, "0x", destination, "31337",
	)
	s.NoError(err)
}

func (s *RollupSuite) hitFinish() {
	finishReq := FinishJSONRequestBody{
		Status: Accept,
	}
	body1, err := json.Marshal(finishReq)
	s.NoError(err)
	bodyReader1 := bytes.NewReader(body1)
	req1 := httptest.NewRequest(http.MethodPost, "/finish", bodyReader1)
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec1 := httptest.NewRecorder()
	c1 := s.server.NewContext(req1, rec1)
	res1 := s.rollupsAPI.Finish(c1)
	s.NoError(res1, "Finish should not return an error")
	s.Assert().Equal(http.StatusOK, rec1.Result().StatusCode)
}
