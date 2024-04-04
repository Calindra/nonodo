package convenience

import (
	"context"
	"log/slog"
	"math/big"
	"os"
	"testing"

	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/lmittmann/tint"
	"github.com/stretchr/testify/suite"
)

func TestExecListenerSuite(t *testing.T) {
	suite.Run(t, new(ExecListenerSuite))
}

type ExecListenerSuite struct {
	suite.Suite
	m          *model.NonodoModel
	repository *ConvenienceRepositoryImpl
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
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &ConvenienceRepositoryImpl{
		db: *db,
	}
	err := s.repository.CreateTables()
	if err != nil {
		panic(err)
	}
}

func (s *ExecListenerSuite) TestItUpdateExecutedAtAndBlocknumber() {
	s.m = model.NewNonodoModel()
	{
		createVoucherMetadataOrFail(s, &ConvenienceVoucher{
			Destination: Bruno,
			Payload:     "0x1122",
			InputIndex:  1,
			OutputIndex: 0,
			Executed:    false,
		})
		createVoucherMetadataOrFail(s, &ConvenienceVoucher{
			Destination: Bob,
			Payload:     "0x1122",
			InputIndex:  2,
			OutputIndex: 0,
			Executed:    false,
		})
		createVoucherMetadataOrFail(s, &ConvenienceVoucher{
			Destination: Alice,
			Payload:     "0x1122",
			InputIndex:  3,
			OutputIndex: 0,
			Executed:    false,
		})
	}
	listener := NewExecListener(s.m, "not a problem", Token)
	listener.repository = s.repository
	eventValues := make([]interface{}, 1)
	eventValues[0] = big.NewInt(2)
	timestamp := uint64(9999)
	blocknumber := uint64(2008)
	err := listener.OnEvent(eventValues, timestamp, blocknumber)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	voucher, err2 := s.repository.FindVoucherByInputAndOutputIndex(ctx, 2, 0)
	if err2 != nil {
		panic(err2)
	}
	s.Equal(Bob.String(), voucher.Destination.String())
	s.Equal(true, voucher.Executed)
}

func createVoucherMetadataOrFail(s *ExecListenerSuite, voucher *ConvenienceVoucher) {
	ctx := context.Background()
	if _, err := s.repository.CreateVoucher(ctx, voucher); err != nil {
		panic(err)
	}
}
