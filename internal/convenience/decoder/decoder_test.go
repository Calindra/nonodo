package decoder

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/calindra/nonodo/internal/devnet"
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
	reportRepository  *repository.ReportRepository
}

func (s *OutputDecoderSuite) SetupTest() {
	db := sqlx.MustConnect("sqlite3", ":memory:")
	outputRepository := repository.OutputRepository{
		Db: *db,
	}
	s.voucherRepository = &repository.VoucherRepository{
		Db: *db, OutputRepository: outputRepository,
	}
	err := s.voucherRepository.CreateTables()
	if err != nil {
		panic(err)
	}

	s.noticeRepository = &repository.NoticeRepository{
		Db: *db, OutputRepository: outputRepository,
	}
	err = s.noticeRepository.CreateTables()
	if err != nil {
		panic(err)
	}

	s.inputRepository = &repository.InputRepository{
		Db: *db,
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
			s.reportRepository,
		),
	}
}

func TestDecoderSuite(t *testing.T) {
	suite.Run(t, new(OutputDecoderSuite))
}

func (s *OutputDecoderSuite) TestHandleOutput() {
	ctx := context.Background()
	err := s.decoder.HandleOutput(ctx, Token, "0x237a816f11", 1, 3)
	if err != nil {
		panic(err)
	}
	voucher, err := s.voucherRepository.FindVoucherByInputAndOutputIndex(ctx, 1, 3)
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

func (s *OutputDecoderSuite) XTestCreateVoucherIdempotency() {
	// we need a better way to check the Idempotency
	ctx := context.Background()
	err := s.decoder.HandleOutput(ctx, Token, "0x237a816f1122", 3, 4)
	if err != nil {
		panic(err)
	}
	voucherCount, err := s.voucherRepository.VoucherCount(ctx)

	if err != nil {
		panic(err)
	}

	s.Equal(1, int(voucherCount))

	err = s.decoder.HandleOutput(ctx, Token, "0x237a816f1122", 3, 4)

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

func (s *OutputDecoderSuite) TestGetConvertedInput() {
	blob := GenerateInputBlob()
	edge := model.InputEdge{
		Node: struct {
			Index int    `json:"index"`
			Blob  string `json:"blob"`
		}{
			Index: 0,
			Blob:  blob,
		},
	}
	cInput, err := s.decoder.GetConvertedInput(edge)
	s.NoError(err)
	s.NotNil(cInput)
}

func (s *OutputDecoderSuite) TestGetConvertedInputFromBytes() {
	blob := GenerateInputBlob()
	edge := model.InputEdge{
		Node: struct {
			Index int    `json:"index"`
			Blob  string `json:"blob"`
		}{
			Index: 0,
			Blob:  blob,
		},
	}
	cInput, err := s.decoder.GetConvertedInput(edge)
	s.NoError(err)
	s.NotNil(cInput)
}

func (s *OutputDecoderSuite) TestParseBytesToInput() {
	blob := GenerateInputBlob()
	decodedInput, err := s.decoder.ParseBytesToInput(common.Hex2Bytes(blob[2:]))
	s.Require().NoError(err)
	s.Equal(common.HexToAddress(devnet.ApplicationAddress), decodedInput.AppContract)
}

func GenerateInputBlob() string {
	// Parse the ABI JSON
	abiParsed, err := contracts.InputsMetaData.GetAbi()

	if err != nil {
		log.Fatal(err)
	}

	chainId := big.NewInt(1000000000000000000)
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	blockNumber := big.NewInt(1000000000000000000)
	blockTimestamp := big.NewInt(1720701841)
	payload := common.Hex2Bytes("11223344556677889900")
	prevRandao := big.NewInt(1000000000000000000)
	index := big.NewInt(1000000000000000000)
	msgSender := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	inputData, _ := abiParsed.Pack("EvmAdvance",
		chainId,
		appContract,
		msgSender,
		blockNumber,
		blockTimestamp,
		prevRandao,
		index,
		payload,
	)
	return fmt.Sprintf("0x%s", common.Bytes2Hex(inputData))
}
