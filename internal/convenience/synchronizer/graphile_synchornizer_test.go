package synchronizer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

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

	err := synchronizer.SynchronizerRepository.CreateTables()
	if err != nil {
		panic(err)
	}

	//creating SynchronizerFetch model
	synchronizerFetch := &model.SynchronizerFetch{
		TimestampAfter:       uint64(time.Now().UnixMilli()),
		IniCursorAfter:       "1f2d3e4b5c6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
		EndCursorAfter:       "baadf00db16b00b5facefeeda1b2c3d4e5f67890a1b2c3d4e5f67890abcdef1234",
		LogVouchersIds:       "",
		IniInputCursorAfter:  "a1b2c3d4e5f67890abcdeffedcba09876543210fedcba9876543210fedcba1234",
		EndInputCursorAfter:  "fedcba9876543210fedcba9876543210abcdef1234567890abcdef1234567890",
		IniReportCursorAfter: "9b8a7c6d5e4f3a2b1c0d9e8f7a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b",
		EndReportCursorAfter: "0a1b2c3d4e5f67890a1b2c3d4e5f67890abcdefabcdefabcdefabcdefabcdefabc",
	}

	_, err = synchronizer.SynchronizerRepository.Create(context.Background(), synchronizerFetch)
	if err != nil {
		panic(err)
	}

	var synchronizerFetches []model.SynchronizerFetch
	err = db.Select(&synchronizerFetches, "SELECT * FROM synchronizer_fetch")
	if err != nil {
		panic(err)
	}

	for _, fetch := range synchronizerFetches {
		fmt.Printf("LIDO DO BANCO DE DADOS: %+v\n", fetch)
	}

	// fmt.Printf("DADO PERSISTIDO %v", data)

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
