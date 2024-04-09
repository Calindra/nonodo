package convenience

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type OutputDecoderSuite struct {
	suite.Suite
	decoder *OutputDecoder
}

func (s *OutputDecoderSuite) SetupTest() {
	db := sqlx.MustConnect("sqlite3", ":memory:")
	repository := repository.VoucherRepository{
		Db: *db,
	}
	err := repository.CreateTables()
	if err != nil {
		panic(err)
	}
	s.decoder = &OutputDecoder{
		convenienceService: ConvenienceService{
			repository: &repository,
		},
	}
}

func TestDecoderSuite(t *testing.T) {
	suite.Run(t, new(OutputDecoderSuite))
}

func (s *OutputDecoderSuite) TestHandleOutput() {
	ctx := context.Background()
	err := s.decoder.HandleOutput(ctx, Token, "0x111122", 1, 2)
	if err != nil {
		panic(err)
	}
	voucher, err := s.decoder.convenienceService.
		repository.FindVoucherByInputAndOutputIndex(ctx, 1, 2)
	if err != nil {
		panic(err)
	}
	s.Equal(Token.String(), voucher.Destination.String())
	s.Equal("0x111122", voucher.Payload)
}

func (s *OutputDecoderSuite) TestGetAbiFromEtherscan() {
	s.T().Skip()
	address := common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29")
	abi, err := s.decoder.GetAbi(address)
	checkError(s, err)
	selectorBytes, err2 := hex.DecodeString("a9059cbb")
	checkError(s, err2)
	abiMethod, err3 := abi.MethodById(selectorBytes)
	checkError(s, err3)
	s.Equal("transfer", abiMethod.RawName)
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
	checkError(s, err)
	selectorBytes, err2 := hex.DecodeString("a9059cbb")
	checkError(s, err2)
	abiMethod, err3 := abi.MethodById(selectorBytes)
	checkError(s, err3)
	s.Equal("transfer", abiMethod.RawName)
}

func checkError(s *OutputDecoderSuite, err error) {
	if err != nil {
		s.T().Fatal(err.Error())
	}
}
