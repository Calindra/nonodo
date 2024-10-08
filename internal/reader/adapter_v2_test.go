package reader

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/calindra/nonodo/internal/graphile"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/suite"
)

type AdapterV2TestSuite struct {
	suite.Suite
	adapter           Adapter
	voucherRepository repository.VoucherRepository
	noticeRepository  repository.NoticeRepository
	inputRepository   repository.InputRepository
	reportRepository  repository.ReportRepository
	httpClient        *MockHttpClient
}

type MockHttpClient struct {
	PostFunc func(body []byte) ([]byte, error)
}

func (m *MockHttpClient) Post(body []byte) ([]byte, error) {
	// If PostFUnc is defined, call it
	if m.PostFunc != nil {
		return m.PostFunc(body)
	}
	// Otherwise return error
	return nil, errors.New("PostFunc not set in the mock")
}

func (s *AdapterV2TestSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")

	outputRepository := repository.OutputRepository{Db: *db}
	voucherRepository := &repository.VoucherRepository{Db: *db, OutputRepository: outputRepository}
	noticeRepository := &repository.NoticeRepository{Db: *db, OutputRepository: outputRepository}
	inputRepository := &repository.InputRepository{Db: *db}
	reportRepository := &repository.ReportRepository{Db: db}

	convenienceService := services.NewConvenienceService(
		voucherRepository,
		noticeRepository,
		inputRepository,
		reportRepository,
	)
	httpClient := &MockHttpClient{}

	s.voucherRepository = *voucherRepository
	s.noticeRepository = *noticeRepository
	s.inputRepository = *inputRepository
	s.reportRepository = *reportRepository
	s.httpClient = httpClient

	err := s.voucherRepository.CreateTables()
	s.NoError(err)

	err = s.noticeRepository.CreateTables()
	s.NoError(err)

	err = s.inputRepository.CreateTables()
	s.NoError(err)

	err = reportRepository.CreateTables()
	s.NoError(err)

	inputBlobAdapter := InputBlobAdapter{}

	s.NoError(err)

	s.adapter = AdapterV2{convenienceService, httpClient, inputBlobAdapter}

}

func TestAdapterV2Suite(t *testing.T) {
	suite.Run(t, new(AdapterV2TestSuite))
}

func (s *AdapterV2TestSuite) TestGetVoucherNotFound() {
	voucherIndex := 2
	_, err := s.adapter.GetVoucher(voucherIndex)
	s.Error(err, "voucher not found")
}

func (s *AdapterV2TestSuite) TestGetVoucherFound() {
	savedVoucher, err := s.voucherRepository.CreateVoucher(context.TODO(), &model.ConvenienceVoucher{
		Destination: common.Address{},
		Payload:     "0x1rtyuio",
		InputIndex:  1,
		OutputIndex: 0,
		Executed:    false,
	})

	s.NoError(err)

	voucherIndex := int(savedVoucher.OutputIndex)
	inputIndex := 1
	voucher, err := s.adapter.GetVoucher(voucherIndex)
	s.NoError(err)
	s.Equal(inputIndex, voucher.InputIndex)
	s.Equal(voucherIndex, voucher.Index)

}

func (s *AdapterV2TestSuite) TestGetAllVouchers() {
	_, err := s.voucherRepository.CreateVoucher(context.TODO(), &model.ConvenienceVoucher{
		Destination: common.Address{},
		Payload:     "0x1rtyuio",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    false,
	})

	s.NoError(err)

	_, err = s.voucherRepository.CreateVoucher(context.TODO(), &model.ConvenienceVoucher{
		Destination: common.Address{},
		Payload:     "0x1rtyujhgfd",
		InputIndex:  2,
		OutputIndex: 3,
		Executed:    false,
	})

	s.NoError(err)

	vouchers, err := s.adapter.GetVouchers(nil, nil, nil, nil, nil)

	s.NoError(err)
	s.Equal(vouchers.TotalCount, 2)

}

func (s *AdapterV2TestSuite) TestGetNoticeNotFound() {
	noticeIndex := 2
	_, err := s.adapter.GetNotice(noticeIndex)
	s.Error(err, "notice not found")
}

func (s *AdapterV2TestSuite) TestGetNoticeFound() {
	savedNotice, err := s.noticeRepository.Create(context.TODO(), &model.ConvenienceNotice{
		Payload:     "0x1rtyuio",
		InputIndex:  1,
		OutputIndex: 2,
	})

	s.NoError(err)

	noticeIndex := int(savedNotice.OutputIndex)
	inputIndex := 1
	notice, err := s.adapter.GetNotice(noticeIndex)
	s.NoError(err)
	s.Equal(inputIndex, notice.InputIndex)
	s.Equal(noticeIndex, notice.Index)

}

func (s *AdapterV2TestSuite) TestGetAllNotices() {
	_, err := s.noticeRepository.Create(context.TODO(), &model.ConvenienceNotice{
		Payload:     "0x1rtyuio",
		InputIndex:  1,
		OutputIndex: 2,
	})

	s.NoError(err)

	_, err = s.noticeRepository.Create(context.TODO(), &model.ConvenienceNotice{
		Payload:     "0x1rtyuio",
		InputIndex:  2,
		OutputIndex: 3,
	})

	s.NoError(err)

	notices, err := s.adapter.GetNotices(nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(notices.TotalCount, 2)
}

func (s *AdapterV2TestSuite) TestGetInputFound() {
	ctx := context.Background()

	_, err := s.inputRepository.Create(ctx, model.AdvanceInput{
		ID: "1",
	})

	s.NoError(err)

	id := "1"
	input, err := s.adapter.GetInput(id)
	s.NoError(err)
	s.Equal(id, input.ID)
}

func (s *AdapterV2TestSuite) TestGetInputNotFound() {
	id := "100"
	_, err := s.adapter.GetInput(id)
	s.Error(err, "input not found")
}

func (s *AdapterV2TestSuite) TestGetReportFound() {
	s.httpClient.PostFunc = func(body []byte) ([]byte, error) {
		return []byte(`{
  "data": {
    "reports": {
      "edges": [
        {
          "node": {
            "blob": "\\x4772656574696e67",
            "index": 2,
            "inputIndex": 1
          }
        }
      ]
    }
  }
}`), nil
	}

	reportIndex := 1

	report, err := s.adapter.GetReport(reportIndex)

	s.NoError(err)
	s.NotNil(report)

}

func (s *AdapterV2TestSuite) TestGetReportNotFound() {
	s.httpClient.PostFunc = func(body []byte) ([]byte, error) {
		return []byte(`{
  "data": {
    "reports": {
      "edges": []
    }
  }
}`), nil
	}

	reportIndex := 1

	report, err := s.adapter.GetReport(reportIndex)

	s.NoError(err)
	s.Nil(report)
}

func (s *AdapterV2TestSuite) TestGetReportsNotFound() {
	s.httpClient.PostFunc = func(body []byte) ([]byte, error) {
		return []byte(`{
  "data": {
    "reports": {
      "edges": []
    }
  }
}`), nil
	}
	ctx := context.Background()
	reports, err := s.adapter.GetReports(ctx, nil, nil, nil, nil, nil)

	s.NoError(err)
	s.NotNil(reports)
	s.Equal(reports.TotalCount, 0)
}

func (s *AdapterV2TestSuite) TestGetReportsFound() {
	ctx := context.Background()
	_, err := s.reportRepository.CreateReport(ctx, model.Report{
		Index:      0,
		InputIndex: 0,
		Payload:    common.Hex2Bytes("deadbeef"),
	})
	s.NoError(err)

	reports, err := s.adapter.GetReports(ctx, nil, nil, nil, nil, nil)

	s.NoError(err)
	s.NotNil(reports)
	s.Equal(1, reports.TotalCount)
}

func (s *AdapterV2TestSuite) TestGetInputsNotFound() {
	ctx := context.Background()
	batch := 10
	inputs, err := s.adapter.GetInputs(ctx, &batch, nil, nil, nil, nil)

	s.NoError(err)
	s.NotNil(inputs)
	s.Equal(inputs.TotalCount, 0)
}

func (s *AdapterV2TestSuite) TestGetInputsFound() {
	ctx := context.Background()

	_, err := s.inputRepository.Create(ctx, model.AdvanceInput{
		ID:    "1",
		Index: 1,
	})

	s.NoError(err)

	_, error := s.inputRepository.Create(ctx, model.AdvanceInput{
		ID:    "2",
		Index: 2,
	})

	s.NoError(error)

	batch := 10
	inputs, err := s.adapter.GetInputs(ctx, &batch, nil, nil, nil, nil)

	s.NoError(err)
	s.Equal(inputs.TotalCount, 2)
}

func (s *AdapterV2TestSuite) TestGetProof() {
	ctx := context.Background()
	s.httpClient.PostFunc = func(body []byte) ([]byte, error) {
		return []byte(`{
			"data": {
				"proof": {
					"nodeId":"WyJwcm9vZnMiLDAsMF0=",
					"inputIndex":0,
					"outputIndex":0,
					"firstInput":0,
					"lastInput":0,
					"validityInputIndexWithinEpoch":0,
					"validityOutputIndexWithinInput":0,
					"validityOutputHashesRootHash":"\\xdeadbeef",
					"validityOutputEpochRootHash":"\\xdeadbeef",
					"validityMachineStateHash":"\\xdeadbeef",
					"validityOutputHashInOutputHashesSiblings":["\\xdeadbeef"],
					"validityOutputHashesInEpochSiblings":["\\xdeadbeef"]
				}
			}
		}
	`), nil
	}
	hitTheRealServer := false
	if hitTheRealServer {
		httpClient := graphile.GraphileClientImpl{}
		inputBlobAdapter := InputBlobAdapter{}
		s.adapter = AdapterV2{nil, &httpClient, inputBlobAdapter}
	}
	proof, err := s.adapter.GetProof(ctx, 0, 0)
	s.NoError(err)
	s.NotNil(proof)
	s.Equal("WyJwcm9vZnMiLDAsMF0=", proof.NodeID)
	s.Equal(0, proof.InputIndex)
	s.Equal(0, proof.OutputIndex)
	s.Equal(0, proof.FirstIndex)
	s.Equal(0, proof.LastInput)
	s.Equal(0, proof.ValidityInputIndexWithinEpoch)
	s.Equal(0, proof.ValidityOutputIndexWithinInput)
	s.Equal("\\xdeadbeef", proof.ValidityOutputHashesRootHash)
	s.Equal("\\xdeadbeef", proof.ValidityOutputEpochRootHash)
	s.Equal("\\xdeadbeef", proof.ValidityMachineStateHash)
	s.Equal("\\xdeadbeef", *proof.ValidityOutputHashInOutputHashesSiblings[0])
	s.Equal("\\xdeadbeef", *proof.ValidityOutputHashesInEpochSiblings[0])
}
