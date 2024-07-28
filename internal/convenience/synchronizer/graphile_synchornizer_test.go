package synchronizer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type DecoderInterfaceMock struct {
	mock.Mock
}

func (m *DecoderInterfaceMock) RetrieveDestination(payload string) (common.Address, error) {
	args := m.Called(payload)
	return args.Get(0).(common.Address), args.Error(1)
}

func (m *DecoderInterfaceMock) GetConvertedInput(input model.InputEdge) (model.ConvertedInput, error) {
	args := m.Called(input)
	return args.Get(0).(model.ConvertedInput), args.Error(1)
}

func (m *DecoderInterfaceMock) HandleOutputV2(ctx context.Context, processOutputData model.ProcessOutputData) error {
	args := m.Called(ctx, processOutputData)
	return args.Error(0)
}

func (m *DecoderInterfaceMock) HandleInput(ctx context.Context, input model.InputEdge, status model.CompletionStatus) error {
	args := m.Called(ctx, input, status)
	return args.Error(0)
}

func (m *DecoderInterfaceMock) HandleReport(ctx context.Context, index int, outputIndex int, payload string) error {
	args := m.Called(ctx, index, outputIndex, payload)
	return args.Error(0)
}

type MockSynchronizerRepository struct {
	mock.Mock
}

func (m *MockSynchronizerRepository) GetDB() *sql.DB {
	args := m.Called()
	return args.Get(0).(*sql.DB)
}

func (m *MockSynchronizerRepository) BeginTxx(ctx context.Context) (*sqlx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(*sqlx.Tx), args.Error(1)
}

func (m *MockSynchronizerRepository) CreateTables() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSynchronizerRepository) Create(ctx context.Context, data *model.SynchronizerFetch) (*model.SynchronizerFetch, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(*model.SynchronizerFetch), args.Error(1)
}

func (m *MockSynchronizerRepository) Count(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockSynchronizerRepository) GetLastFetched(ctx context.Context) (*model.SynchronizerFetch, error) {
	args := m.Called(ctx)
	return args.Get(0).(*model.SynchronizerFetch), args.Error(1)
}

func getTestOutputResponse() OutputResponse {
	jsonData := `
    {
        "data": {
            "outputs": {
                "pageInfo": {
                    "startCursor": "output_start_1",
                    "endCursor": "output_end_1",
                    "hasNextPage": true,
                    "hasPreviousPage": false
                },
                "edges": [
                    {
                        "cursor": "output_cursor_1",
                        "node": {
                            "index": 1,
                            "blob": "0x1a2b3c",
                            "inputIndex": 1
                        }
                    },
                    {
                        "cursor": "output_cursor_2",
                        "node": {
                            "index": 2,
                            "blob": "0x4d5e6f",
                            "inputIndex": 2
                        }
                    }
                ]
            },
            "inputs": {
                "pageInfo": {
                    "startCursor": "input_start_1",
                    "endCursor": "input_end_1",
                    "hasNextPage": false,
                    "hasPreviousPage": false
                },
                "edges": [
                    {
                        "cursor": "input_cursor_1",
                        "node": {
                            "index": 1,
                            "blob": "0x7a8b9c"
                        }
                    },
                    {
                        "cursor": "input_cursor_2",
                        "node": {
                            "index": 2,
                            "blob": "0xabcdef"
                        }
                    }
                ]
            },
            "reports": {
                "pageInfo": {
                    "startCursor": "report_start_1",
                    "endCursor": "report_end_1",
                    "hasNextPage": false,
                    "hasPreviousPage": true
                },
                "edges": [
                    {
                        "node": {
                            "index": 1,
                            "inputIndex": 1,
                            "blob": "0x123456"
                        }
                    },
                    {
                        "node": {
                            "index": 2,
                            "inputIndex": 2,
                            "blob": "0x789abc"
                        }
                    }
                ]
            }
        }
    }
    `

	var response OutputResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		panic("Error while unmarshaling the test JSON: " + err.Error())
	}
	return response
}

func TestHandleOutput_Failure(t *testing.T) {
	response := getTestOutputResponse()

	ctx := context.Background()

	decoderMock := &DecoderInterfaceMock{}

	synchronizer := GraphileSynchronizer{
		Decoder:                decoderMock,
		SynchronizerRepository: &repository.SynchronizerRepository{},
		GraphileFetcher:        &GraphileFetcher{},
	}

	erro := errors.New("Handle Output Failure")

	decoderMock.On("HandleOutputV2", mock.Anything, mock.Anything).Return(erro)

	err := synchronizer.handleGraphileResponse(ctx, response)

	assert.Error(t, err)
	assert.EqualError(t, err, "error handling output: Handle Output Failure")
}

func TestDecoderHandleOutput_Failure(t *testing.T) {
	response := getTestOutputResponse()
	ctx := context.Background()

	decoderMock := &DecoderInterfaceMock{}

	synchronizer := GraphileSynchronizer{
		Decoder:                decoderMock,
		SynchronizerRepository: &repository.SynchronizerRepository{},
		GraphileFetcher:        &GraphileFetcher{},
	}
	erro := errors.New("Decoder Handler Output Failure")

	decoderMock.On("RetrieveDestination", mock.Anything).Return(common.Address{}, nil)
	decoderMock.On("HandleOutputV2", mock.Anything, mock.Anything).Return(erro)

	err := synchronizer.handleGraphileResponse(ctx, response)

	assert.Error(t, err)
	assert.EqualError(t, err, "error handling output: Decoder Handler Output Failure")

}

func TestHandleInput_Failure(t *testing.T) {
	response := getTestOutputResponse()
	ctx := context.Background()

	decoderMock := &DecoderInterfaceMock{}

	synchronizer := GraphileSynchronizer{
		Decoder:                decoderMock,
		SynchronizerRepository: &repository.SynchronizerRepository{},
		GraphileFetcher:        &GraphileFetcher{},
	}

	erro := errors.New("Handle Input Failure")

	decoderMock.On("RetrieveDestination", mock.Anything).Return(common.Address{}, nil)
	decoderMock.On("HandleOutputV2", mock.Anything, mock.Anything).Return(nil)
	decoderMock.On("HandleInput", mock.Anything, mock.Anything, mock.Anything).Return(erro)

	err := synchronizer.handleGraphileResponse(ctx, response)

	assert.Error(t, err)
	assert.EqualError(t, err, "error handling input: Handle Input Failure")

}

func TestContextWithTimeout_Failure(t *testing.T) {
	db := sqlx.MustConnect("sqlite3", ":memory:")
	defer db.Close()

	decoderMock := &DecoderInterfaceMock{}
	synchronizer := GraphileSynchronizer{
		Decoder: decoderMock,
		SynchronizerRepository: &repository.SynchronizerRepository{
			Db: *db,
		},
		GraphileFetcher: &GraphileFetcher{},
	}

	response := getTestOutputResponse()

	err := synchronizer.SynchronizerRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	// Verificar se a tabela foi criada
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='synchronizer_fetch';")
	if err != nil {
		t.Fatalf("Erro ao verificar se a tabela foi criada: %v", err)
	}

	if count == 0 {
		t.Fatalf("A tabela synchronizer_fetch não foi criada.")
	}

	fmt.Println("A tabela synchronizer_fetch foi criada com sucesso.")

	decoderMock.On("RetrieveDestination", mock.Anything).Return(common.Address{}, nil)
	decoderMock.On("HandleOutputV2", mock.Anything, mock.Anything).Return(nil)
	decoderMock.On("HandleInput", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	decoderMock.On("HandleReport", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err = synchronizer.handleWithDBTransaction(response)
	if err != nil {
		fmt.Println("ERRO handleWithDBTransaction ")
	}

}

// func TestContextWithTimeout_Failure(t *testing.T) {
// 	db, sqlmock, err := sqlmock.New()
// 	assert.NoError(t, err)
// 	defer db.Close()

// 	sqlmock.ExpectBegin()

// 	sqlxDB := sqlx.NewDb(db, "sqlmock")
// 	tx, err := sqlxDB.BeginTxx(context.Background(), nil)
// 	assert.NoError(t, err)

// 	syncRepoMock := &MockSynchronizerRepository{}

// 	response := getTestOutputResponse()

// 	decoderMock := &DecoderInterfaceMock{}

// 	synchronizer := GraphileSynchronizer{
// 		Decoder:                decoderMock,
// 		SynchronizerRepository: syncRepoMock,
// 		GraphileFetcher:        &GraphileFetcher{},
// 	}

// 	erro := errors.New("Handle Output Value")

// 	syncRepoMock.On("BeginTxx", mock.Anything).Return(tx, nil)
// 	decoderMock.On("RetrieveDestination", mock.Anything).Return(common.Address{}, nil)
// 	decoderMock.On("HandleOutputV2", mock.Anything, mock.Anything).Return(erro)

// 	synchronizer.handleWithDBTransaction(response)

// }
