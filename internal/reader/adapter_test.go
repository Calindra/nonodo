package reader

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	convenience "github.com/calindra/nonodo/internal/convenience/model"
	cRepos "github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/reader/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/suite"
)

//
// Test suite
//

type AdapterSuite struct {
	suite.Suite
	reportRepository *cRepos.ReportRepository
	inputRepository  *cRepos.InputRepository
	adapter          Adapter
}

func (s *AdapterSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
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
	s.adapter = &AdapterV1{
		reportRepository: s.reportRepository,
		inputRepository:  s.inputRepository,
	}
}

func TestReportRepositorySuite(t *testing.T) {
	suite.Run(t, new(AdapterSuite))
}

func (s *AdapterSuite) TestCreateTables() {
	err := s.reportRepository.CreateTables()
	s.NoError(err)
}

func (s *AdapterSuite) TestGetReport() {
	ctx := context.Background()
	_, err := s.reportRepository.Create(ctx, convenience.Report{
		InputIndex: 1,
		Index:      2,
		Payload:    common.Hex2Bytes("1122"),
	})
	s.NoError(err)
	report, err := s.adapter.GetReport(2, 1)
	s.NoError(err)
	s.Equal("0x1122", report.Payload)
}

func (s *AdapterSuite) TestGetReports() {
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := s.reportRepository.Create(ctx, convenience.Report{
			InputIndex: i,
			Index:      0,
			Payload:    common.Hex2Bytes("1122"),
		})
		s.NoError(err)
	}
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
	for i := 0; i < 3; i++ {
		_, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
			ID:             strconv.Itoa(i),
			Index:          i,
			Status:         convenience.CompletionStatusUnprocessed,
			MsgSender:      common.HexToAddress(fmt.Sprintf("000000000000000000000000000000000000000%d", i)),
			Payload:        common.Hex2Bytes("0x1122"),
			BlockNumber:    1,
			BlockTimestamp: time.Now(),
		})
		s.NoError(err)
	}
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
