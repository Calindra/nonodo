package convenience

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Synchronizer struct {
	decoder *OutputDecoder
}

func NewSynchronizer(decoder *OutputDecoder) *Synchronizer {
	return &Synchronizer{
		decoder: decoder,
	}
}

// String implements supervisor.Worker.
func (x Synchronizer) String() string {
	return "Synchronizer"
}

func (x Synchronizer) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	return x.VoucherPolling(ctx)
}

func (x *Synchronizer) VoucherPolling(ctx context.Context) error {
	// GraphQL endpoint URL
	url := "http://localhost:8080/graphql"
	after := ""
	query := `query GetVouchers($after: String) {
		vouchers(first: 1, after: $after) {
			totalCount
			edges{
				node{
					destination
					payload
					index
					input {
						index
					}
					proof {
						validity {
							inputIndexWithinEpoch
							outputIndexWithinInput
							outputHashesRootHash
							vouchersEpochRootHash
							noticesEpochRootHash
							machineStateHash
							outputHashInOutputHashesSiblings
							outputHashesInEpochSiblings
						}
					context
					}
				}
			}
			pageInfo {
				startCursor
				endCursor
				hasNextPage
				hasPreviousPage
			}
		}
		
	}`
	for {
		fmt.Println("Querying...")
		variables := map[string]interface{}{}
		if len(after) > 0 {
			variables = map[string]interface{}{
				"after": after,
			}
		}

		payload, err := json.Marshal(map[string]interface{}{
			"operationName": nil,
			"query":         query,
			"variables":     variables,
		})
		fmt.Printf("after %s\n", after)
		if err != nil {
			fmt.Println("Error marshalling JSON:", err)
			return err
		}

		// Make a POST request to the GraphQL endpoint
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			fmt.Println("Error creating request:", err)
			continue
		}

		// Set request headers
		req.Header.Set("Content-Type", "application/json")

		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			continue
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Println("Error reading response:", err)
			continue
		}

		var response VoucherResponse
		if err := json.Unmarshal(body, &response); err != nil {
			fmt.Println("Error parsing JSON:", err)
			continue
		}

		// Handle response data
		for _, edge := range response.Data.Vouchers.Edges {
			outputIndex := edge.Node.Index
			inputIndex := edge.Node.Input.Index
			fmt.Printf("InputIndex %d; OutputIndex %d;\n", inputIndex, outputIndex)
			err := x.decoder.HandleOutput(ctx,
				common.HexToAddress(edge.Node.Destination),
				edge.Node.Payload,
				uint64(inputIndex),
				uint64(outputIndex),
			)
			if err != nil {
				panic(err)
			}
		}
		if len(response.Data.Vouchers.PageInfo.StartCursor) > 0 {
			after = response.Data.Vouchers.PageInfo.StartCursor
		}
		// Wait for a certain duration before making the next request (e.g., 5 seconds)
		sleepInSeconds := 3
		time.Sleep(time.Duration(sleepInSeconds) * time.Second)
	}
}

type VoucherResponse struct {
	Data VoucherData `json:"data"`
}

type VoucherData struct {
	Vouchers VoucherConnection `json:"vouchers"`
}

type VoucherConnection struct {
	TotalCount int           `json:"totalCount"`
	Edges      []VoucherEdge `json:"edges"`
	PageInfo   PageInfo      `json:"pageInfo"`
}

type VoucherEdge struct {
	Node   Voucher `json:"node"`
	Cursor string  `json:"cursor"`
}

type Voucher struct {
	Index       int      `json:"index"`
	Destination string   `json:"destination"`
	Payload     string   `json:"payload"`
	Proof       Proof    `json:"proof"`
	Input       InputRef `json:"input"`
}

type InputRef struct {
	Index int `json:"index"`
}
type Proof struct {
	Validity OutputValidityProof `json:"validity"`
	Context  string              `json:"context"`
}

type OutputValidityProof struct {
	InputIndexWithinEpoch            int      `json:"inputIndexWithinEpoch"`
	OutputIndexWithinInput           int      `json:"outputIndexWithinInput"`
	OutputHashesRootHash             string   `json:"outputHashesRootHash"`
	VouchersEpochRootHash            string   `json:"vouchersEpochRootHash"`
	NoticesEpochRootHash             string   `json:"noticesEpochRootHash"`
	MachineStateHash                 string   `json:"machineStateHash"`
	OutputHashInOutputHashesSiblings []string `json:"outputHashInOutputHashesSiblings"`
	OutputHashesInEpochSiblings      []string `json:"outputHashesInEpochSiblings"`
}

type PageInfo struct {
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
}
