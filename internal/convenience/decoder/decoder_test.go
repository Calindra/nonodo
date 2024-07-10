package decoder

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

var Token = common.HexToAddress("0xc6e7DF5E7b4f2A278906862b61205850344D4e7d")

type OutputDecoderSuite struct {
	suite.Suite
	decoder           *OutputDecoder
	voucherRepository *repository.VoucherRepository
	noticeRepository  *repository.NoticeRepository
	inputRepository   *repository.InputRepository
}

func (s *OutputDecoderSuite) SetupTest() {
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.voucherRepository = &repository.VoucherRepository{
		Db: *db,
	}
	err := s.voucherRepository.CreateTables()
	if err != nil {
		panic(err)
	}

	s.noticeRepository = &repository.NoticeRepository{
		Db: *db,
	}
	err = s.noticeRepository.CreateTables()
	if err != nil {
		panic(err)
	}

	s.inputRepository = &repository.InputRepository{
		Db: db,
	}
	err = s.inputRepository.CreateTables()

	if err != nil {
		panic(err)
	}

	s.decoder = &OutputDecoder{
		convenienceService: *services.NewConvenienceService(
			s.voucherRepository,
			s.noticeRepository,
			s.inputRepository,
		),
	}
}

func TestDecoderSuite(t *testing.T) {
	suite.Run(t, new(OutputDecoderSuite))
}

func (s *OutputDecoderSuite) TestHandleOutput() {
	ctx := context.Background()
	err := s.decoder.HandleOutput(ctx, Token, "0xef615e2f11", 1, 2)
	if err != nil {
		panic(err)
	}
	voucher, err := s.voucherRepository.FindVoucherByInputAndOutputIndex(ctx, 1, 2)
	if err != nil {
		panic(err)
	}
	s.Equal(Token.String(), voucher.Destination.String())
	s.Equal("0x11", voucher.Payload)
}

func (s *OutputDecoderSuite) TestGetAbiFromEtherscan() {
	s.T().Skip()
	address := common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29")
	abi, err := s.decoder.GetAbi(address)
	s.NoError(err)
	selectorBytes, err := hex.DecodeString("a9059cbb")
	s.NoError(err)
	abiMethod, err := abi.MethodById(selectorBytes)
	s.NoError(err)
	s.Equal("transfer", abiMethod.RawName)
}

func (s *OutputDecoderSuite) TestCreateVoucherIdempotency() {
	ctx := context.Background()
	err := s.decoder.HandleOutput(ctx, Token, "0xef615e2f1122", 3, 4)
	if err != nil {
		panic(err)
	}
	voucherCount, err := s.voucherRepository.VoucherCount(ctx)

	if err != nil {
		panic(err)
	}

	s.Equal(1, int(voucherCount))

	err = s.decoder.HandleOutput(ctx, Token, "0xef615e2f1122", 3, 4)

	if err != nil {
		panic(err)
	}

	voucherCount, err = s.voucherRepository.VoucherCount(ctx)

	if err != nil {
		panic(err)
	}

	s.Equal(1, int(voucherCount))
}

func (s *OutputDecoderSuite) TestDecode() {
	json := `[{
		"constant": false,
		"inputs": [
			{
				"name": "_to",
				"type": "address"
			},
			{
				"name": "_value",
				"type": "uint256"
			}
		],
		"name": "transfer",
		"outputs": [
			{
				"name": "",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	}]`
	abi, err := jsonToAbi(json)
	s.NoError(err)
	selectorBytes, err := hex.DecodeString("a9059cbb")
	s.NoError(err)
	abiMethod, err := abi.MethodById(selectorBytes)
	s.NoError(err)
	s.Equal("transfer", abiMethod.RawName)
}
