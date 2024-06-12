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

const graphileQuery = `query { outputs(first: %d) { edges { cursor node { blob index inputIndex nodeId  proofByInputIndexAndOutputIndex { validityOutputIndexWithinInput validityOutputHashesRootHash validityOutputHashesInEpochSiblings validityOutputHashInOutputHashesSiblings validityOutputEpochRootHash validityMachineStateHash validityInputIndexWithinEpoch } } }  pageInfo { endCursor hasNextPage hasPreviousPage startCursor }}}`

const graphileQueryWithCursor = `query { outputs(first: %d, after: %s ) { edges { cursor node { blob index inputIndex nodeId  proofByInputIndexAndOutputIndex { validityOutputIndexWithinInput validityOutputHashesRootHash validityOutputHashesInEpochSiblings validityOutputHashInOutputHashesSiblings validityOutputEpochRootHash validityMachineStateHash validityInputIndexWithinEpoch } } }  pageInfo { endCursor hasNextPage hasPreviousPage startCursor }}}`

const ErrorSendingGraphileRequest = `
+-----------------------------------------------------------+
| Please ensure that the rollups-node is up and running at: |
GRAPH_QL_URL
+-----------------------------------------------------------+
`

type GraphileFetcher struct {
	Url             string
	CursorAfter     string
	BatchSize       int
	Query           string
	QueryWithCursor string
}

func NewGraphileFetcher() *GraphileFetcher {
	return &GraphileFetcher{
		Url:             "http://localhost:5000/graphql",
		CursorAfter:     "",
		BatchSize:       10,
		Query:           graphileQuery,
		QueryWithCursor: graphileQueryWithCursor,
	}
}

func (v *GraphileFetcher) Fetch() (*OutputResponse, error) {
	slog.Debug("Graphile querying", "after", v.CursorAfter)

	var query string

	if len(v.CursorAfter) > 0 {
		query = fmt.Sprintf(graphileQueryWithCursor, v.BatchSize, v.CursorAfter)
	} else {
		query = fmt.Sprintf(graphileQuery, v.BatchSize)
	}

	payload, err := json.Marshal(map[string]interface{}{
		"query": query,
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

	var response OutputResponse
	if err := json.Unmarshal(body, &response); err != nil {
		slog.Error("Error parsing JSON:", "error", err)
		return nil, err
	}

	slog.Info("response is ", "responseparsed", response)

	return &response, nil
}

type OutputResponse struct {
	Data struct {
		Outputs struct {
			PageInfo struct {
				StartCursor     string `json:"startCursor"`
				EndCursor       string `json:"endCursor"`
				HasNextPage     bool   `json:"hasNextPage"`
				HasPreviousPage bool   `json:"hasPreviousPage"`
			}
			Edges []struct {
				Cursor string `json:"cursor"`
				Node   struct {
					Index      int    `json:"index"`
					Blob       string `json:"blob"`
					InputIndex int    `json:"inputIndex"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"outputs"`
	} `json:"data"`
}
