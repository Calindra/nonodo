package reader

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	cModel "github.com/calindra/nonodo/internal/convenience/model"
	cRepos "github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/reader/model"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/suite"
)

//
// Test suite
//

type AdapterSuite struct {
	suite.Suite
	reportRepository  *cRepos.ReportRepository
	inputRepository   *cRepos.InputRepository
	voucherRepository *cRepos.VoucherRepository
	noticeRepository  *cRepos.NoticeRepository
	adapter           Adapter
	dbFactory         *commons.DbFactory
}

func (s *AdapterSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.dbFactory = commons.NewDbFactory()
	db := s.dbFactory.CreateDb("adapterV1.sqlite3")
	s.reportRepository = &cRepos.ReportRepository{
		Db: db,
	}
	err := s.reportRepository.CreateTables()
	s.NoError(err)
	s.inputRepository = &cRepos.InputRepository{
		Db: *db,
	}
	err = s.inputRepository.CreateTables()
	s.NoError(err)

	s.voucherRepository = &cRepos.VoucherRepository{
		Db: *db,
	}
	err = s.voucherRepository.CreateTables()
	s.Require().NoError(err)

	s.noticeRepository = &cRepos.NoticeRepository{
		Db: *db,
	}
	err = s.noticeRepository.CreateTables()
	s.Require().NoError(err)
	s.adapter = &AdapterV1{
		reportRepository: s.reportRepository,
		inputRepository:  s.inputRepository,
		convenienceService: services.NewConvenienceService(
			s.voucherRepository, s.noticeRepository, nil, nil,
		),
	}
}

func (s *AdapterSuite) TearDownTest() {
	s.dbFactory.Cleanup()
}

func TestAdapterSuite(t *testing.T) {
	suite.Run(t, new(AdapterSuite))
}

func (s *AdapterSuite) TestCreateTables() {
	err := s.reportRepository.CreateTables()
	s.NoError(err)
}

func (s *AdapterSuite) TestGetReport() {
	ctx := context.Background()
	reportSaved, err := s.reportRepository.CreateReport(ctx, cModel.Report{
		InputIndex: 1,
		Index:      999,
		Payload:    "1122",
	})
	s.NoError(err)
	report, err := s.adapter.GetReport(ctx, reportSaved.Index)
	s.NoError(err)
	s.Equal("0x1122", report.Payload)
}

func (s *AdapterSuite) TestGetReports() {
	ctx := context.Background()
	s.createTestData(ctx)
	res, err := s.adapter.GetReports(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(3, res.TotalCount)

	inputIndex := 1
	res, err = s.adapter.GetReports(ctx, nil, nil, nil, nil, &inputIndex)
	s.NoError(err)
	s.Equal(1, res.TotalCount)
}

func (s *AdapterSuite) TestGetInputs() {
	ctx := context.Background()
	s.createTestData(ctx)
	res, err := s.adapter.GetInputs(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(3, res.TotalCount)

	msgSender := "0x0000000000000000000000000000000000000001"
	filter := model.InputFilter{
		MsgSender: &msgSender,
	}
	res, err = s.adapter.GetInputs(ctx, nil, nil, nil, nil, &filter)
	s.NoError(err)
	s.Equal(1, res.TotalCount)
	s.Equal(res.Edges[0].Node.MsgSender, msgSender)
}

func (s *AdapterSuite) TestGetInputsFilteredByAppContract() {
	ctx := context.Background()
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	s.createTestData(ctx)

	// without address
	res, err := s.adapter.GetInputs(ctx, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(3, res.TotalCount)

	// with inexistent address
	appContract2 := common.HexToAddress("0x000028bb862fb57e8a2bcd567a2e929a0be56a5e")
	ctx2 := context.WithValue(ctx, cModel.AppContractKey, appContract2.Hex())
	res2, err := s.adapter.GetInputs(ctx2, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(0, res2.TotalCount)

	// with correct address
	ctx3 := context.WithValue(ctx, cModel.AppContractKey, appContract.Hex())
	res3, err := s.adapter.GetInputs(ctx3, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(3, res3.TotalCount)
}

func (s *AdapterSuite) TestGetVouchersFilteredByAppContract() {
	ctx := context.Background()
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	s.createTestData(ctx)

	// without address
	res, err := s.adapter.GetVouchers(ctx, nil, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(3, res.TotalCount)

	// with inexistent address
	appContract2 := common.HexToAddress("0x000028bb862fb57e8a2bcd567a2e929a0be56a5e")
	ctx2 := context.WithValue(ctx, cModel.AppContractKey, appContract2.Hex())
	res2, err := s.adapter.GetVouchers(ctx2, nil, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(0, res2.TotalCount)

	// with correct address
	ctx3 := context.WithValue(ctx, cModel.AppContractKey, appContract.Hex())
	res3, err := s.adapter.GetVouchers(ctx3, nil, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(3, res3.TotalCount)
}

func (s *AdapterSuite) TestGetVoucherFilteredByAppContract() {
	ctx := context.Background()
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	s.createTestData(ctx)

	// without address
	res, err := s.adapter.GetVoucher(ctx, 1)
	s.Require().NoError(err)
	s.NotNil(res) // returns the notice

	// with inexistent address
	appContract2 := common.HexToAddress("0x000028bb862fb57e8a2bcd567a2e929a0be56a5e")
	ctx2 := context.WithValue(ctx, cModel.AppContractKey, appContract2.Hex())
	res2, err := s.adapter.GetVoucher(ctx2, 1)
	s.ErrorContains(err, "voucher not found")
	s.Nil(res2) // returns nothing

	// with correct address
	ctx3 := context.WithValue(ctx, cModel.AppContractKey, appContract.Hex())
	res3, err := s.adapter.GetVoucher(ctx3, 1)
	s.Require().NoError(err)
	s.NotNil(res3) // returns the notice
}

func (s *AdapterSuite) TestGetNoticesFilteredByAppContract() {
	ctx := context.Background()
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	s.createTestData(ctx)

	// without address
	res, err := s.adapter.GetNotices(ctx, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(3, res.TotalCount) // returns all

	// with inexistent address
	appContract2 := common.HexToAddress("0x000028bb862fb57e8a2bcd567a2e929a0be56a5e")
	ctx2 := context.WithValue(ctx, cModel.AppContractKey, appContract2.Hex())
	res2, err := s.adapter.GetNotices(ctx2, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(0, res2.TotalCount) // returns nothing

	// with correct address
	ctx3 := context.WithValue(ctx, cModel.AppContractKey, appContract.Hex())
	res3, err := s.adapter.GetNotices(ctx3, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(3, res3.TotalCount) // returns all
}

func (s *AdapterSuite) TestGetNoticeFilteredByAppContract() {
	ctx := context.Background()
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	s.createTestData(ctx)

	// without address
	res, err := s.adapter.GetNotice(ctx, 1)
	s.Require().NoError(err)
	s.NotNil(res) // returns the notice

	// with inexistent address
	appContract2 := common.HexToAddress("0x000028bb862fb57e8a2bcd567a2e929a0be56a5e")
	ctx2 := context.WithValue(ctx, cModel.AppContractKey, appContract2.Hex())
	res2, err := s.adapter.GetNotice(ctx2, 1)
	s.ErrorContains(err, "notice not found")
	s.Nil(res2) // returns nothing

	// with correct address
	ctx3 := context.WithValue(ctx, cModel.AppContractKey, appContract.Hex())
	res3, err := s.adapter.GetNotice(ctx3, 1)
	s.Require().NoError(err)
	s.NotNil(res3) // returns the notice
}

func (s *AdapterSuite) TestGetInputFilteredByAppContract() {
	ctx := context.Background()
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	s.createTestData(ctx)

	// without address
	res, err := s.adapter.GetInput(ctx, "1")
	s.Require().NoError(err)
	s.NotNil(res) // returns the input

	// with inexistent address
	appContract2 := common.HexToAddress("0x000028bb862fb57e8a2bcd567a2e929a0be56a5e")
	ctx2 := context.WithValue(ctx, cModel.AppContractKey, appContract2.Hex())
	res2, err := s.adapter.GetInput(ctx2, "1")
	s.ErrorContains(err, "input not found")
	s.Nil(res2) // returns nothing

	// with correct address
	ctx3 := context.WithValue(ctx, cModel.AppContractKey, appContract.Hex())
	res3, err := s.adapter.GetInput(ctx3, "1")
	s.Require().NoError(err)
	s.NotNil(res3) // returns the input
}

func (s *AdapterSuite) TestGetInputByIndexFilteredAppContract() {
	ctx := context.Background()
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	s.createTestData(ctx)

	// without address
	res, err := s.adapter.GetInputByIndex(ctx, 1)
	s.Require().NoError(err)
	s.NotNil(res) // returns the input

	// with inexistent address
	appContract2 := common.HexToAddress("0x000028bb862fb57e8a2bcd567a2e929a0be56a5e")
	ctx2 := context.WithValue(ctx, cModel.AppContractKey, appContract2.Hex())
	res2, err := s.adapter.GetInputByIndex(ctx2, 1)
	s.ErrorContains(err, "input not found")
	s.Nil(res2) // returns nothing

	// with correct address
	ctx3 := context.WithValue(ctx, cModel.AppContractKey, appContract.Hex())
	res3, err := s.adapter.GetInputByIndex(ctx3, 1)
	s.Require().NoError(err)
	s.NotNil(res3) // returns the input
}

func (s *AdapterSuite) TestGetReportsFilteredByAppContract() {
	ctx := context.Background()
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	s.createTestData(ctx)

	// without address
	res, err := s.adapter.GetReports(ctx, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(3, res.TotalCount) // returns all

	// with inexistent address
	appContract2 := common.HexToAddress("0x000028bb862fb57e8a2bcd567a2e929a0be56a5e")
	ctx2 := context.WithValue(ctx, cModel.AppContractKey, appContract2.Hex())
	res2, err := s.adapter.GetReports(ctx2, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(0, res2.TotalCount) // returns nothing

	// with correct address
	ctx3 := context.WithValue(ctx, cModel.AppContractKey, appContract.Hex())
	res3, err := s.adapter.GetReports(ctx3, nil, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Equal(3, res3.TotalCount) // returns all
}

func (s *AdapterSuite) TestGetReportFilteredByAppContract() {
	ctx := context.Background()
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	s.createTestData(ctx)

	// without address
	res, err := s.adapter.GetReport(ctx, 1)
	s.Require().NoError(err)
	s.NotNil(res) // returns the report

	// with inexistent address
	appContract2 := common.HexToAddress("0x000028bb862fb57e8a2bcd567a2e929a0be56a5e")
	ctx2 := context.WithValue(ctx, cModel.AppContractKey, appContract2.Hex())
	res2, err := s.adapter.GetReport(ctx2, 1)
	s.ErrorContains(err, "report not found")
	s.Nil(res2) // returns nothing

	// with correct address
	ctx3 := context.WithValue(ctx, cModel.AppContractKey, appContract.Hex())
	res3, err := s.adapter.GetReport(ctx3, 1)
	s.Require().NoError(err)
	s.NotNil(res3) // returns all
}

func (s *AdapterSuite) createTestData(ctx context.Context) {
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	for i := 0; i < 3; i++ {
		_, err := s.inputRepository.Create(ctx, cModel.AdvanceInput{
			ID:             strconv.Itoa(i),
			Index:          i,
			Status:         cModel.CompletionStatusUnprocessed,
			MsgSender:      common.HexToAddress(fmt.Sprintf("000000000000000000000000000000000000000%d", i)),
			Payload:        "0x1122",
			BlockNumber:    1,
			BlockTimestamp: time.Now(),
			AppContract:    appContract,
		})
		s.NoError(err)
		_, err = s.noticeRepository.Create(ctx, &cModel.ConvenienceNotice{
			AppContract: appContract.Hex(),
			OutputIndex: uint64(i),
			InputIndex:  uint64(i),
		})
		s.Require().NoError(err)
		_, err = s.voucherRepository.CreateVoucher(ctx, &cModel.ConvenienceVoucher{
			AppContract: appContract,
			OutputIndex: uint64(i),
			InputIndex:  uint64(i),
		})
		s.Require().NoError(err)
		_, err = s.reportRepository.CreateReport(ctx, cModel.Report{
			AppContract: appContract,
			InputIndex:  i,
			Index:       i, // now it's a global number for the dapp
			Payload:     "1122",
		})
		s.NoError(err)
	}
}
