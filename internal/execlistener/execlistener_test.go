package execlistener

import (
	"log/slog"
	"math/big"
	"os"
	"testing"

	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/tint"
	"github.com/stretchr/testify/suite"
)

func TestExecListenerSuite(t *testing.T) {
	suite.Run(t, new(ExecListenerSuite))
}

type ExecListenerSuite struct {
	suite.Suite
	m *model.NonodoModel
}

var Bob = common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
var Bruno = common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
var Alice = common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
var Token = common.HexToAddress("0xc6e7DF5E7b4f2A278906862b61205850344D4e7d")

func (s *ExecListenerSuite) SetupTest() {
	logOpts := new(tint.Options)
	logOpts.Level = slog.LevelDebug
	logOpts.AddSource = true
	logOpts.NoColor = false
	logOpts.TimeFormat = "[15:04:05.000]"
	handler := tint.NewHandler(os.Stdout, logOpts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func (s *ExecListenerSuite) TestItUpdateExecutedAtAndBlocknumber() {
	s.m = model.NewNonodoModel(nil)
	{
		createVoucherMetadataOrFail(s, model.VoucherMetadata{
			Beneficiary: Bruno,
			Contract:    Token,
			InputIndex:  1,
			OutputIndex: 0,
			ExecutedAt:  0,
		})
		createVoucherMetadataOrFail(s, model.VoucherMetadata{
			Beneficiary: Bob,
			Contract:    Token,
			InputIndex:  2,
			OutputIndex: 0,
			ExecutedAt:  0,
		})
		createVoucherMetadataOrFail(s, model.VoucherMetadata{
			Beneficiary: Alice,
			Contract:    Token,
			InputIndex:  3,
			OutputIndex: 0,
			ExecutedAt:  0,
		})
	}
	listener := NewExecListener(s.m, "not a problem", Token)
	eventValues := make([]interface{}, 1)
	eventValues[0] = big.NewInt(2)
	timestamp := uint64(9999)
	blocknumber := uint64(2008)
	err := listener.OnEvent(eventValues, timestamp, blocknumber)
	if err != nil {
		panic(err)
	}
	filters := model.CreateFilterList(`[{"field": "executedAt", "gt": "0"}]`)
	results, err := s.m.GetVouchersMetadata(filters)
	if err != nil {
		panic(err)
	}
	s.Equal(1, len(results))
	s.Equal(Bob.String(), results[0].Beneficiary.String())
	s.Equal(timestamp, results[0].ExecutedAt)
	s.Equal(blocknumber, results[0].ExecutedBlock)
}

func createVoucherMetadataOrFail(s *ExecListenerSuite, voucherMetadata model.VoucherMetadata) {
	if err := s.m.AddVoucherMetadata(&voucherMetadata); err != nil {
		panic(err)
	}
}
