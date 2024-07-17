package synchronizer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/calindra/nonodo/internal/convenience/decoder"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type AdapterMock struct {
	mock.Mock
}

func (m *AdapterMock) GetDestination(payload string) (common.Address, error) {
	args := m.Called(payload)
	return args.Get(0).(common.Address), args.Error(1)
}

func (m *AdapterMock) ConvertVoucherPayloadToV2(payloadV1 string) string {
	args := m.Called(payloadV1)
	return args.String(0)
}

func TestGetDestination_Failure(t *testing.T) {
	jsonData := `{
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
	}`

	var response OutputResponse

	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		fmt.Println("Erro ao fazer Unmarshal:", err)
		return
	}

	ctx := context.Background()
	adapterMock := &AdapterMock{}
	synchronizer := GraphileSynchronizer{
		Decoder:                &decoder.OutputDecoder{},
		SynchronizerRepository: &repository.SynchronizerRepository{},
		GraphileFetcher:        &GraphileFetcher{},
	}

	erro := errors.New("error")
	adapterMock.On("ConvertVoucherPayloadToV2", mock.Anything).Return("1a2b3c")
	adapterMock.On("GetDestination", mock.Anything).Return(common.Address{}, erro)

	err = synchronizer.handleGraphileResponseTwo(ctx, response, adapterMock)

	assert.Error(t, err)
	assert.EqualError(t, err, "error retrieving destination for node blob '0x1a2b3c': error")
}
