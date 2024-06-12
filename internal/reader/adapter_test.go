package reader

import (
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

//
// Test suite
//

type AdapterSuite struct {
	suite.Suite
	reportRepository *model.ReportRepository
	adapter          Adapter
}

func (s *AdapterSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.reportRepository = &model.ReportRepository{
		Db: db,
	}
	err := s.reportRepository.CreateTables()
	s.NoError(err)
	s.adapter = &AdapterV1{
		reportRepository: s.reportRepository,
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
	_, err := s.reportRepository.Create(model.Report{
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
	for i := 0; i < 3; i++ {
		_, err := s.reportRepository.Create(model.Report{
			InputIndex: i,
			Index:      0,
			Payload:    common.Hex2Bytes("1122"),
		})
		s.NoError(err)
	}
	res, err := s.adapter.GetReports(nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(3, res.TotalCount)

	inputIndex := 1
	res, err = s.adapter.GetReports(nil, nil, nil, nil, &inputIndex)
	s.NoError(err)
	s.Equal(1, res.TotalCount)
}

func (s *AdapterSuite) TestGetOutputFormat() {
	s.Equal("1", GenerateBlobOutput())
}

func GenerateBlobOutput() string {
	// Parse the ABI JSON
	abiParsed, err := contracts.OutputsMetaData.GetAbi()

	if err != nil {
		log.Fatal(err)
	}

	value := big.NewInt(1000000000000000000)
	payload := common.Hex2Bytes("11223344556677889900")
	destination := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	inputData, _ := abiParsed.Pack("Voucher",
		destination,
		value,
		payload,
	)

	return fmt.Sprintf("0x%s", common.Bytes2Hex(inputData))
}
