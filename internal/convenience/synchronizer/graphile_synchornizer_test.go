package synchronizer

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type AdapterInterfaceMock struct {
	mock.Mock
}

type DecoderInterfaceMock struct {
	mock.Mock
}

func (m *AdapterInterfaceMock) RetrieveDestination(output model.OutputEdge) (common.Address, error) {
	args := m.Called(output)
	return args.Get(0).(common.Address), args.Error(1)
}

func (m *AdapterInterfaceMock) GetConvertedInput(input model.InputEdge) (model.ConvertedInput, error) {
	args := m.Called(input)
	return args.Get(0).(model.ConvertedInput), args.Error(1)
}

func (m *DecoderInterfaceMock) HandleOutput(ctx context.Context, destination common.Address, payload string, inputIndex uint64, outputIndex uint64) error {
	args := m.Called(ctx, destination, payload, inputIndex, outputIndex)
	return args.Error(0)
}

func (m *DecoderInterfaceMock) HandleInput(ctx context.Context, index int, status model.CompletionStatus, msgSender common.Address, payload string, blockNumber uint64, blockTimestamp time.Time, prevRandao string) error {
	args := m.Called(ctx, index, status, msgSender, payload, blockNumber, blockTimestamp, prevRandao)
	return args.Error(0)
}

func (m *DecoderInterfaceMock) HandleReport(ctx context.Context, index int, outputIndex int, payload string) error {
	args := m.Called(ctx, index, outputIndex, payload)
	return args.Error(0)
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

func TestGetDestination_Failure(t *testing.T) {
	response := getTestOutputResponse()

	ctx := context.Background()

	adapterMock := &AdapterInterfaceMock{}
	decoderMock := &DecoderInterfaceMock{}

	synchronizer := GraphileSynchronizer{
		Decoder:                decoderMock,
		SynchronizerRepository: &repository.SynchronizerRepository{},
		GraphileFetcher:        &GraphileFetcher{},
		Adapter:                adapterMock,
	}

	erro := errors.New("error")
	// adapterMock.On("ConvertVoucher", mock.Anything).Return("1a2b3c")
	adapterMock.On("RetrieveDestination", mock.Anything).Return(common.Address{}, erro)

	err := synchronizer.handleGraphileResponse(response, ctx)

	assert.Error(t, err)
	assert.EqualError(t, err, "error retrieving destination for node blob '0x1a2b3c': error")
}

func TestDecoderHandleOutput_Failure(t *testing.T) {
	response := getTestOutputResponse()
	ctx := context.Background()

	adapterMock := &AdapterInterfaceMock{}
	decoderMock := &DecoderInterfaceMock{}

	synchronizer := GraphileSynchronizer{
		Decoder:                decoderMock,
		SynchronizerRepository: &repository.SynchronizerRepository{},
		GraphileFetcher:        &GraphileFetcher{},
		Adapter:                adapterMock,
	}
	erro := errors.New("Decoder Handler Output Failure")
	// adapterMock.On("ConvertVoucher", mock.Anything).Return("1a2b3c")
	adapterMock.On("RetrieveDestination", mock.Anything).Return(common.Address{}, nil)
	decoderMock.On("HandleOutput", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(erro)

	err := synchronizer.handleGraphileResponse(response, ctx)

	assert.Error(t, err)
	assert.EqualError(t, err, "error handling output: Decoder Handler Output Failure")

}

func TestGetConvertedInput_Failure(t *testing.T) {
	response := getTestOutputResponse()
	ctx := context.Background()

	adapterMock := &AdapterInterfaceMock{}
	decoderMock := &DecoderInterfaceMock{}

	synchronizer := GraphileSynchronizer{
		Decoder:                decoderMock,
		SynchronizerRepository: &repository.SynchronizerRepository{},
		GraphileFetcher:        &GraphileFetcher{},
		Adapter:                adapterMock,
	}

	convertedInput := model.ConvertedInput{
		MsgSender:      common.Address{},
		BlockNumber:    big.NewInt(0),
		BlockTimestamp: 0,
		PrevRandao:     "",
		Payload:        "",
	}

	erro := errors.New("Get Converted Input Failure")
	// adapterMock.On("ConvertVoucher", mock.Anything).Return("1a2b3c")
	adapterMock.On("RetrieveDestination", mock.Anything).Return(common.Address{}, nil)
	decoderMock.On("HandleOutput", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	adapterMock.On("GetConvertedInput", mock.Anything).Return(convertedInput, erro)

	err := synchronizer.handleGraphileResponse(response, ctx)

	assert.Error(t, err)
	assert.EqualError(t, err, "error getting converted input: Get Converted Input Failure")

}
