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
	"github.com/stretchr/testify/require"
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

func TestCommit_handleWithDBTransaction(t *testing.T) {
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

	var count int
	err = synchronizer.SynchronizerRepository.GetDB().Get(&count, "SELECT COUNT(*) FROM synchronizer_fetch")
	if err != nil {
		t.Fatalf("Error checking the number of rows in the 'synchronizer_fetch' table: %v", err)
	}

	require.Equal(t, 0, count, "The table should be empty.")

	decoderMock.On("RetrieveDestination", mock.Anything).Return(common.Address{}, nil)
	decoderMock.On("HandleOutputV2", mock.Anything, mock.Anything).Return(nil)
	decoderMock.On("HandleInput", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	decoderMock.On("HandleReport", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	outputResponse := getTestOutputResponse()

	err = synchronizer.handleWithDBTransaction(outputResponse)
	if err != nil {
		fmt.Println("ERRO handleWithDBTransaction ")
	}

	var otherCount int
	err = synchronizer.SynchronizerRepository.GetDB().Get(&otherCount, "SELECT COUNT(*) FROM synchronizer_fetch")
	if err != nil {
		t.Fatalf("Error checking the number of rows in the 'synchronizer_fetch' table: %v", err)
	}

	require.Equal(t, 1, otherCount, "The table should have one row.")

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

	dataBase := synchronizer.SynchronizerRepository.GetDB()
	ctx, err := repository.StartTransaction(context.Background(), dataBase)
	require.NoError(t, err)

	if err != nil {
		fmt.Printf("Fail to initialize transaction on test %v \n", err)
	}

	tx, err := repository.GetTransaction(ctx)

	if err != nil {
		fmt.Printf("Recovery transaction fail on test %v \n", err)
	}
	defer tx.Rollback()

	syncFetch := &model.SynchronizerFetch{
		Id:                   1,
		TimestampAfter:       1234567890,
		IniCursorAfter:       "cursor_1",
		LogVouchersIds:       "voucher_123,voucher_456",
		EndCursorAfter:       "cursor_2",
		IniInputCursorAfter:  "input_cursor_1",
		EndInputCursorAfter:  "input_cursor_2",
		IniReportCursorAfter: "report_cursor_1",
		EndReportCursorAfter: "report_cursor_2",
	}

	_, err = synchronizer.SynchronizerRepository.Create(ctx, syncFetch)

	if err != nil {
		fmt.Printf("Erro ao criar SynchronizerFetch: %v\n", err)
	}

	var fetched model.SynchronizerFetch
	err = tx.GetContext(context.Background(), &fetched, "SELECT * FROM synchronizer_fetch WHERE id = ?", syncFetch.Id)
	require.NoError(t, err, "Erro ao recuperar dados")

	require.Equal(t, syncFetch.Id, fetched.Id, "ID não corresponde")
	require.Equal(t, syncFetch.TimestampAfter, fetched.TimestampAfter, "TimestampAfter não corresponde")
	require.Equal(t, syncFetch.IniCursorAfter, fetched.IniCursorAfter, "IniCursorAfter não corresponde")
	require.Equal(t, syncFetch.LogVouchersIds, fetched.LogVouchersIds, "LogVouchersIds não corresponde")
	require.Equal(t, syncFetch.EndCursorAfter, fetched.EndCursorAfter, "EndCursorAfter não corresponde")
	require.Equal(t, syncFetch.IniInputCursorAfter, fetched.IniInputCursorAfter, "IniInputCursorAfter não corresponde")
	require.Equal(t, syncFetch.EndInputCursorAfter, fetched.EndInputCursorAfter, "EndInputCursorAfter não corresponde")
	require.Equal(t, syncFetch.IniReportCursorAfter, fetched.IniReportCursorAfter, "IniReportCursorAfter não corresponde")
	require.Equal(t, syncFetch.EndReportCursorAfter, fetched.EndReportCursorAfter, "EndReportCursorAfter não corresponde")

	erro := errors.New("Handle Output Value")

	decoderMock.On("RetrieveDestination", mock.Anything).Return(common.Address{}, nil)
	decoderMock.On("HandleOutputV2", mock.Anything, mock.Anything).Return(nil)
	decoderMock.On("HandleInput", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	decoderMock.On("HandleReport", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(erro)
	outputResponse := getTestOutputResponse()

	err = synchronizer.handleWithDBTransaction(outputResponse)
	if err != nil {
		fmt.Println("ERRO handleWithDBTransaction ")
		// tx.Rollback()
	}

	var otherCount int
	err = tx.GetContext(context.Background(), &otherCount, "SELECT COUNT(*) FROM synchronizer_fetch")
	if err != nil {
		t.Fatalf("Erro ao verificar se a tabela foi criada: %v", err)
	}

	var otherFetched model.SynchronizerFetch
	err = tx.GetContext(context.Background(), &otherFetched, "SELECT * FROM synchronizer_fetch WHERE id = ?", syncFetch.Id)
	require.NoError(t, err, "Erro ao recuperar dados")

	// Exibir o resultado na tela
	// fmt.Printf("Dados recuperados:\n")
	// fmt.Printf("ID: %d\n", otherFetched.Id)
	// fmt.Printf("Timestamp After: %d\n", otherFetched.TimestampAfter)
	// fmt.Printf("Ini Cursor After: %s\n", otherFetched.IniCursorAfter)
	// fmt.Printf("Log Vouchers IDs: %s\n", otherFetched.LogVouchersIds)
	// fmt.Printf("End Cursor After: %s\n", otherFetched.EndCursorAfter)
	// fmt.Printf("Ini Input Cursor After: %s\n", otherFetched.IniInputCursorAfter)
	// fmt.Printf("End Input Cursor After: %s\n", otherFetched.EndInputCursorAfter)
	// fmt.Printf("Ini Report Cursor After: %s\n", otherFetched.IniReportCursorAfter)
	// fmt.Printf("End Report Cursor After: %s\n", otherFetched.EndReportCursorAfter)

	// Confirmar a transação
	// err = tx.Commit()
	// require.NoError(t, err, "Erro ao confirmar a transação")

	require.Equal(t, 0, otherCount, "A Tabela deveria estar vazia")

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
