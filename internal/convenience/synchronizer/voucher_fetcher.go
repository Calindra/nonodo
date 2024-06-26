package synchronizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const query = `query GetVouchers($after: String, $batchSize: Int) {
	vouchers(first: $batchSize, after: $after) {
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

const ErrorSendingRequest = `
+-----------------------------------------------------------+
| Please ensure that the rollups-node is up and running at: |
GRAPH_QL_URL
+-----------------------------------------------------------+
`

const DefaultBatchSize = 10

type VoucherFetcher struct {
	Url         string
	CursorAfter string
	BatchSize   int
	Query       string
}

func NewVoucherFetcher() *VoucherFetcher {
	return &VoucherFetcher{
		Url:         "http://localhost:8080/graphql",
		CursorAfter: "",
		BatchSize:   DefaultBatchSize,
		Query:       query,
	}
}

func (v *VoucherFetcher) Fetch() (*VoucherResponse, error) {
	slog.Debug("GraphQL querying", "after", v.CursorAfter)

	variables := map[string]interface{}{
		"batchSize": v.BatchSize,
	}
	if len(v.CursorAfter) > 0 {
		variables["after"] = v.CursorAfter
	}

	payload, err := json.Marshal(map[string]interface{}{
		"operationName": nil,
		"query":         query,
		"variables":     variables,
	})
	if err != nil {
		slog.Error("Error marshalling JSON:", "error", err)
		return nil, err
	}

	// Make a POST request to the GraphQL endpoint
	req, err := http.NewRequest("POST", v.Url, bytes.NewBuffer(payload))
	if err != nil {
		slog.Error("Error creating request:", "error", err)
		return nil, err
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending request:", "error", err)
		fmt.Println(
			strings.Replace(
				ErrorSendingRequest,
				"GRAPH_QL_URL",
				fmt.Sprintf("|    %-55s|", v.Url),
				-1,
			))
		return nil, err
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		slog.Error("Error reading response:", "error", err)
		return nil, err
	}

	var response VoucherResponse
	if err := json.Unmarshal(body, &response); err != nil {
		slog.Error("Error parsing JSON:", "error", err)
		return nil, err
	}
	return &response, nil
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
