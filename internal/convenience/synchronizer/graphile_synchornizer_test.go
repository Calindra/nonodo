package synchronizer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type DecoderInterfaceMock struct {
	mock.Mock
}

func (m *DecoderInterfaceMock) RetrieveDestination(output model.OutputEdge) (common.Address, error) {
	args := m.Called(output)
	return args.Get(0).(common.Address), args.Error(1)
}

func (m *DecoderInterfaceMock) GetConvertedInput(input model.InputEdge) (model.ConvertedInput, error) {
	args := m.Called(input)
	return args.Get(0).(model.ConvertedInput), args.Error(1)
}

func (m *DecoderInterfaceMock) HandleOutput(ctx context.Context, destination common.Address, payload string, inputIndex uint64, outputIndex uint64) error {
	args := m.Called(ctx, destination, payload, inputIndex, outputIndex)
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

	decoderMock.On("RetrieveDestination", mock.Anything).Return(common.Address{}, erro)

	err := synchronizer.handleGraphileResponse(ctx, response)

	assert.Error(t, err)
	assert.EqualError(t, err, "error processing output: error retrieving destination for node blob '0x1a2b3c': Handle Output Failure")
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
	decoderMock.On("HandleOutput", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(erro)

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
	decoderMock.On("HandleOutput", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	decoderMock.On("HandleInput", mock.Anything, mock.Anything, mock.Anything).Return(erro)

	err := synchronizer.handleGraphileResponse(ctx, response)

	assert.Error(t, err)
	assert.EqualError(t, err, "error handling input: Handle Input Failure")

}
