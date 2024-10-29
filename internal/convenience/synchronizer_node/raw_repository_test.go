package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
)

type RawNodeSuite struct {
	suite.Suite
	rawRepository              RawRepository
	ctx                        context.Context
	dockerComposeStartedByTest bool
	DefaultTimeout             time.Duration
}

func (s *RawNodeSuite) SetupSuite() {
	s.DefaultTimeout = 1 * time.Minute
	s.ctx = context.Background()
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}

	envMap, err := raw.LoadMapEnvFile()
	s.NoError(err)
	dbName := "rollupsdb"
	dbPass := "password"
	if _, ok := envMap["POSTGRES_PASSWORD"]; ok {
		dbPass = envMap["POSTGRES_PASSWORD"]
	}
	if _, ok := envMap["POSTGRES_DB"]; ok {
		dbName = envMap["POSTGRES_DB"]
	}
	uri := fmt.Sprintf("postgres://postgres:%s@localhost:5432/%s?sslmode=disable", dbPass, dbName)
	slog.Info("Raw Input URI", "uri", uri)
	dbNodeV2 := sqlx.MustConnect("postgres", uri)
	s.rawRepository = RawRepository{
		connectionURL: uri,
		Db:            dbNodeV2,
	}
}

func (s *RawNodeSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
}
func (s *RawNodeSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
}

func (s *RawNodeSuite) TearDownTest() {}

func TestRawNodeSuite(t *testing.T) {
	suite.Run(t, new(RawNodeSuite))
}

func (s *RawNodeSuite) TestSynchronizerNodeListInputs() {
	ctx, cancel := context.WithTimeout(s.ctx, s.DefaultTimeout)
	defer cancel()

	result, err := s.rawRepository.Db.QueryxContext(ctx, "SELECT * FROM input")
	s.NoError(err)

	inputs := []RawInput{}

	for result.Next() {
		var input RawInput
		err := result.StructScan(&input)
		s.NoError(err)
		inputs = append(inputs, input)
	}

	firstInput := inputs[0]
	s.Equal(firstInput.ID, uint64(1))

	b := inputs[0].BlockNumber

	firstBlockNumber := big.NewInt(0).SetUint64(b)
	slog.Info("First block number", "blockNumber", firstBlockNumber)

	firstBlockNumberDB := big.NewInt(122)

	s.Equal(firstBlockNumberDB, firstBlockNumber)

	s.Equal(DEFAULT_TEST_APP_CONTRACT, common.BytesToAddress(inputs[0].ApplicationAddress).Hex())

}

func (s *RawNodeSuite) TestSynchronizerNodeInputByID() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	inputs, err := s.rawRepository.FindAllInputsByFilter(ctx, FilterInput{IDgt: 2, IsStatusNone: false}, nil)
	s.NoError(err)
	firstInput := inputs[0]
	s.Equal(firstInput.ID, uint64(2))

	b := inputs[0].BlockNumber

	firstBlockNumber := big.NewInt(0).SetUint64(b)
	slog.Info("First block number", "blockNumber", firstBlockNumber)

	firstBlockNumberDB := big.NewInt(124)

	s.Equal(firstBlockNumberDB, firstBlockNumber)

	s.Equal(DEFAULT_TEST_APP_CONTRACT, common.BytesToAddress(inputs[0].ApplicationAddress).Hex())
}

func (s *RawNodeSuite) TestSynchronizerNodeReportByID() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	reports, err := s.rawRepository.FindAllReportsByFilter(ctx, FilterID{IDgt: 1})
	s.NoError(err)
	firstInput := reports[0]
	s.Equal(firstInput.ID, int64(1))

	b := reports[0].InputID

	firstInputID := big.NewInt(b)
	slog.Info("First Input ID", "firstInputID", firstInputID)

	firstInputIDDB := big.NewInt(1)

	s.Equal(firstInputIDDB, firstInputID)
}

func (s *RawNodeSuite) TestSynchronizerNodeOutputByID() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	outputs, err := s.rawRepository.FindAllOutputsByFilter(ctx, FilterID{IDgt: 1})
	s.NoError(err)
	firstInput := outputs[0]
	s.Equal(2, int(firstInput.ID))

	b := outputs[0].InputID

	firstInputID := big.NewInt(0).SetUint64(b)
	slog.Info("First Input ID", "firstInputID", firstInputID)

	firstInputIdDB := big.NewInt(1)

	s.Equal(firstInputIdDB, firstInputID)
	s.Equal(DEFAULT_TEST_APP_CONTRACT, common.BytesToAddress(outputs[0].AppContract).Hex())
	s.Equal("1", outputs[1].InputIndex)
}

func (s *RawNodeSuite) TestDecodeChainIDFromInputbox() {
	abi, err := contracts.InputsMetaData.GetAbi()
	s.NoError(err)

	rawData := "0x415bf3630000000000000000000000000000000000000000000000000000000000007a690000000000000000000000005112cf49f2511ac7b13a032c4c62a48410fc28fb000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000000000000000000000000000000000000000046900000000000000000000000000000000000000000000000000000000670931c70a06511d13afecb37c88e47c1a7357e42205ac4b8e49fcd4632373e036261e26000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000005deadbeef11000000000000000000000000000000000000000000000000000000" // nolint

	// EvmAdvance
	data := common.Hex2Bytes(strings.TrimPrefix(rawData, "0x"))
	methodId := data[:4]
	slog.Debug("MethodId", "methodId", methodId, "hex", rawData[2:10])
	input, err := abi.MethodById(methodId)
	s.NoError(err)

	dataDecoded := make(map[string]interface{})
	dataEncoded := data[4:]
	err = input.Inputs.UnpackIntoMap(dataDecoded, dataEncoded)
	s.NoError(err)
	s.NotEmpty(dataDecoded)
	s.Equal(big.NewInt(31337), dataDecoded["chainId"])
	slog.Info("DataDecoded", "dataDecoded", dataDecoded)
	// s.NotNil(nil)
}

func (s *RawNodeSuite) TestSynchronizerNodeFindInputByOutput() {
	ctx := context.Background()

	input, err := s.rawRepository.FindInputByOutput(ctx, FilterID{IDgt: 1})
	s.NoError(err)
	s.Equal(uint64(1), input.ID)
}
