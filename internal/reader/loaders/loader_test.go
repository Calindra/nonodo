package loaders

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	cModel "github.com/calindra/nonodo/internal/convenience/model"
	cRepos "github.com/calindra/nonodo/internal/convenience/repository"
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

type LoaderSuite struct {
	suite.Suite
	reportRepository  *cRepos.ReportRepository
	inputRepository   *cRepos.InputRepository
	voucherRepository *cRepos.VoucherRepository
	noticeRepository  *cRepos.NoticeRepository
	dbFactory         *commons.DbFactory
}

func (s *LoaderSuite) SetupTest() {
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

}

func (s *LoaderSuite) TearDownTest() {
	s.dbFactory.Cleanup()
}

func TestAdapterSuite(t *testing.T) {
	suite.Run(t, new(LoaderSuite))
}
func (s *LoaderSuite) TestGetReports() {
	ctx := context.Background()
	s.createTestData(ctx)
	loaders := NewLoaders(s.reportRepository.Db)
	rCtx := context.WithValue(ctx, loadersKey, loaders)

	var wg sync.WaitGroup
	wg.Add(2) // We will be loading 2 reports in parallel

	// Channel to capture the results
	results := make(chan *model.Report, 2)
	errs := make(chan error, 2)

	// First report loader
	go func() {
		defer wg.Done()
		report, err := loaders.ReportLoader.Load(rCtx, "1")
		if err != nil {
			errs <- err
			return
		}
		results <- report
	}()

	// Second report loader
	go func() {
		defer wg.Done()
		report, err := loaders.ReportLoader.Load(rCtx, "2")
		if err != nil {
			errs <- err
			return
		}
		results <- report
	}()

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)
	close(errs)

	// Collect and assert results
	for err := range errs {
		s.Require().NoError(err)
	}

	reports := make(map[string]*model.Report)
	for r := range results {
		reports[strconv.FormatInt(int64(r.Index), 10)] = r
	}
	s.Equal(1, int(reports["1"].InputIndex))
	s.Equal(2, int(reports["2"].InputIndex))
	s.Fail("This failure is intentional ;-)")
}

func (s *LoaderSuite) createTestData(ctx context.Context) {
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	for i := 0; i < 3; i++ {
		_, err := s.inputRepository.Create(ctx, cModel.AdvanceInput{
			ID:             strconv.Itoa(i),
			Index:          i,
			Status:         cModel.CompletionStatusUnprocessed,
			MsgSender:      common.HexToAddress(fmt.Sprintf("000000000000000000000000000000000000000%d", i)),
			Payload:        common.Hex2Bytes("0x1122"),
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
			Payload:     common.Hex2Bytes("1122"),
		})
		s.NoError(err)
	}
}