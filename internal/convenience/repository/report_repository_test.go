package repository

import (
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	cModel "github.com/calindra/nonodo/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/suite"
)

//
// Test suite
//

type ReportRepositorySuite struct {
	suite.Suite
	reportRepository *ReportRepository
}

func (s *ReportRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.reportRepository = &ReportRepository{
		Db: db,
	}
	err := s.reportRepository.CreateTables()
	s.NoError(err)
}

func TestReportRepositorySuite(t *testing.T) {
	suite.Run(t, new(ReportRepositorySuite))
}

func (s *ReportRepositorySuite) TestCreateTables() {
	err := s.reportRepository.CreateTables()
	s.NoError(err)
}

func (s *ReportRepositorySuite) TestCreateReport() {
	_, err := s.reportRepository.Create(cModel.Report{
		Index:      1,
		InputIndex: 2,
		Payload:    common.Hex2Bytes("1122"),
	})
	s.NoError(err)
}

func (s *ReportRepositorySuite) TestCreateReportAndFind() {
	_, err := s.reportRepository.Create(cModel.Report{
		InputIndex: 1,
		Index:      2,
		Payload:    common.Hex2Bytes("1122"),
	})
	s.NoError(err)
	report, err := s.reportRepository.FindByInputAndOutputIndex(
		uint64(1),
		uint64(2),
	)
	s.NoError(err)
	s.Equal("1122", common.Bytes2Hex(report.Payload))
}

func (s *ReportRepositorySuite) TestReportNotFound() {
	report, err := s.reportRepository.FindByInputAndOutputIndex(
		uint64(404),
		uint64(404),
	)
	s.NoError(err)
	s.Nil(report)
}

func (s *ReportRepositorySuite) TestCreateReportAndFindAll() {
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			_, err := s.reportRepository.Create(cModel.Report{
				InputIndex: i,
				Index:      j,
				Payload:    common.Hex2Bytes("1122"),
			})
			s.NoError(err)
		}
	}
	reports, err := s.reportRepository.FindAll(nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(12, int(reports.Total))
	s.Equal(0, reports.Rows[0].InputIndex)
	s.Equal(2, reports.Rows[len(reports.Rows)-1].InputIndex)

	filter := []*cModel.ConvenienceFilter{}
	{
		field := "InputIndex"
		value := "1"
		filter = append(filter, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	reports, err = s.reportRepository.FindAll(nil, nil, nil, nil, filter)
	s.NoError(err)
	s.Equal(4, int(reports.Total))
	s.Equal(1, reports.Rows[0].InputIndex)
	s.Equal(0, reports.Rows[0].Index)
	s.Equal(1, reports.Rows[len(reports.Rows)-1].InputIndex)
	s.Equal(3, reports.Rows[len(reports.Rows)-1].Index)
	s.Equal("1122", common.Bytes2Hex(reports.Rows[0].Payload))
}
