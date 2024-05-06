package reader

import (
	"context"
	"errors"
	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
	"log/slog"
	"testing"
)

type AdapterV2TestSuite struct {
	suite.Suite
	adapter           Adapter
	voucherRepository repository.VoucherRepository
	noticeRepository  repository.NoticeRepository
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

	voucherRepository := &repository.VoucherRepository{Db: *db}
	noticeRepository := &repository.NoticeRepository{Db: *db}

	convenienceService := services.NewConvenienceService(voucherRepository, noticeRepository)
	httpClient := &MockHttpClient{}

	s.voucherRepository = *voucherRepository
	s.noticeRepository = *noticeRepository
	s.httpClient = httpClient

	err := s.voucherRepository.CreateTables()
	s.NoError(err)

	err = s.noticeRepository.CreateTables()
	s.NoError(err)

	inputBlobAdapter, err := NewInputBlobAdapter()

	s.NoError(err)

	s.adapter = AdapterV2{convenienceService, httpClient, *inputBlobAdapter}

}

func TestAdapterV2Suite(t *testing.T) {
	suite.Run(t, new(AdapterV2TestSuite))
}

func (s *AdapterV2TestSuite) TestGetVoucherNotFound() {
	voucherIndex := 2
	inputIndex := 1
	_, err := s.adapter.GetVoucher(voucherIndex, inputIndex)
	s.Error(err, "voucher not found")
}

func (s *AdapterV2TestSuite) TestGetVoucherFound() {
	_, err := s.voucherRepository.CreateVoucher(context.TODO(), &model.ConvenienceVoucher{
		Destination: common.Address{},
		Payload:     "0x1rtyuio",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    false,
	})

	s.NoError(err)

	voucherIndex := 2
	inputIndex := 1
	voucher, err := s.adapter.GetVoucher(voucherIndex, inputIndex)
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
	inputIndex := 1
	_, err := s.adapter.GetNotice(noticeIndex, inputIndex)
	s.Error(err, "notice not found")
}

func (s *AdapterV2TestSuite) TestGetNoticeFound() {
	_, err := s.noticeRepository.Create(context.TODO(), &model.ConvenienceNotice{
		Payload:     "0x1rtyuio",
		InputIndex:  1,
		OutputIndex: 2,
	})

	s.NoError(err)

	noticeIndex := 2
	inputIndex := 1
	notice, err := s.adapter.GetNotice(noticeIndex, inputIndex)
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

//func (s *AdapterV2TestSuite) TestGetInputFound() {
//	blob := generateBlob()
//	s.httpClient.PostFunc = func(body []byte) ([]byte, error) {
//		return []byte(fmt.Sprintf(`{
//  "data": {
//    "inputs": {
//      "edges": [
//        {
//          "cursor": "WyJwcmltYXJ5X2tleV9hc2MiLFsxXV0=",
//          "node": {
//            "index": 1,
//            "blob": "%s",
//            "status": "ACCEPTED"
//          }
//        }
//      ]
//    }
//  }
//}`, blob)), nil
//	}
//
//	input := 2
//
//	inputResponse, err := s.adapter.GetInput(input)
//
//	s.NoError(err)
//	s.NotNil(inputResponse)
//}

func (s *AdapterV2TestSuite) TestGetInputNotFound() {
	s.httpClient.PostFunc = func(body []byte) ([]byte, error) {
		return []byte(`{
  "data": {
    "inputs": {
      "edges": []
    }
  }
}`), nil
	}

	input := 2

	inputResponse, err := s.adapter.GetInput(input)

	s.NoError(err)
	s.Nil(inputResponse)
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
	inputIndex := 2

	report, err := s.adapter.GetReport(reportIndex, inputIndex)

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
	inputIndex := 2

	report, err := s.adapter.GetReport(reportIndex, inputIndex)

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

	reports, err := s.adapter.GetReports(nil, nil, nil, nil, nil)

	s.NoError(err)
	s.NotNil(reports)
	s.Equal(reports.TotalCount, 0)
}

func (s *AdapterV2TestSuite) TestGetReportsFound() {
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

	reports, err := s.adapter.GetReports(nil, nil, nil, nil, nil)

	s.NoError(err)
	s.NotNil(reports)
	s.Equal(reports.TotalCount, 1)
}

func (s *AdapterV2TestSuite) TestGetInputsNotFound() {
	s.httpClient.PostFunc = func(body []byte) ([]byte, error) {
		return []byte(`{
  "data": {
    "inputs": {
      "edges": []
    }
  }
}`), nil
	}

	inputs, err := s.adapter.GetInputs(nil, nil, nil, nil, nil)

	s.NoError(err)
	s.NotNil(inputs)
	s.Equal(inputs.TotalCount, 0)
}

//func (s *AdapterV2TestSuite) TestGetInputsFound() {
//	blob := generateBlob()
//	s.httpClient.PostFunc = func(body []byte) ([]byte, error) {
//		return []byte(fmt.Sprintf(`{
//  "data": {
//    "inputs": {
//      "edges": [
//        {
//          "cursor": "WyJwcmltYXJ5X2tleV9hc2MiLFsxXV0=",
//          "node": {
//            "index": 1,
//            "blob": "%s",
//            "status": "ACCEPTED"
//          }
//        }
//      ]
//    }
//  }
//}`, blob)), nil
//	}
//
//	inputs, err := s.adapter.GetInputs(nil, nil, nil, nil, nil)
//
//	s.NoError(err)
//	s.NotNil(inputs)
//	s.Equal(inputs.TotalCount, 1)
//}

//func generateBlob() string {
//	chainId := uint256ToBytes(123)
//	appContract := addressToBytes("0x1234567890123456789012345678901234567890")
//	msgSender := addressToBytes("0xabcdef1234567890abcdef1234567890abcdef12")
//	blockNumber := uint256ToBytes(456)
//	blockTimestamp := uint256ToBytes(789)
//	prevRandao := uint256ToBytes(987)
//	index := uint256ToBytes(321)
//	payload := "Hello, World!"
//
//	payloadHex := fmt.Sprintf(
//		"%x%x%x%x%x%x%x%x",
//		chainId,
//		appContract,
//		msgSender,
//		blockNumber,
//		blockTimestamp,
//		prevRandao,
//		index,
//		payload)
//
//	payloadHex = "0x" + payloadHex
//
//	return payloadHex
//}
//
//func uint256ToBytes(value uint64) []byte {
//	bytes := make([]byte, 32)
//	for i := 0; i < 32; i++ {
//		bytes[31-i] = byte(value >> (8 * i))
//	}
//	return bytes
//}
//
//func addressToBytes(address string) []byte {
//	bytes, _ := hex.DecodeString(address[2:])
//	return bytes
//}
