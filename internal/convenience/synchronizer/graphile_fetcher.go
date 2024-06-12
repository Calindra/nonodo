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

const graphileQuery = `query {
    outputs(first: %d, after: %s) {
      edges {
        cursor
        node {
          blob
          index
          inputIndex
          nodeId
          inputByInputIndex {
            index
          }
          proofByInputIndexAndOutputIndex {
            validityOutputIndexWithinInput
            validityOutputHashesRootHash
            validityOutputHashesInEpochSiblings
            validityOutputHashInOutputHashesSiblings
            validityOutputEpochRootHash
            validityMachineStateHash
            validityInputIndexWithinEpoch
          }
        }
      }
      pageInfo {
        endCursor
        hasNextPage
        hasPreviousPage
        startCursor
      }
    }
  }`

const ErrorSendingGraphileRequest = `
+-----------------------------------------------------------+
| Please ensure that the rollups-node is up and running at: |
GRAPH_QL_URL
+-----------------------------------------------------------+
`

type GraphileFetcher struct {
	Url         string
	CursorAfter string
	BatchSize   int
	Query       string
}

func NewGraphileFetcher() *GraphileFetcher {
	return &GraphileFetcher{
		Url:         "http://localhost:5000/graphql",
		CursorAfter: "",
		BatchSize:   10,
		Query:       graphileQuery,
	}
}

func (v *GraphileFetcher) Fetch() (*OutputResponse, error) {
	slog.Debug("Graphile querying", "after", v.CursorAfter)

	variables := map[string]interface{}{
		"batchSize": v.BatchSize,
	}
	if len(v.CursorAfter) > 0 {
		variables["after"] = v.CursorAfter
	}

	payload, err := json.Marshal(map[string]interface{}{
		"operationName": nil,
		"query":         fmt.Sprintf(graphileQuery, v.BatchSize, v.CursorAfter),
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

	var response OutputResponse
	if err := json.Unmarshal(body, &response); err != nil {
		slog.Error("Error parsing JSON:", "error", err)
		return nil, err
	}
	return &response, nil
}

type OutputResponse struct {
	Data OutputData `json:"data"`
}

type OutputData struct {
	Query OutputQuery `json:"query"`
}

type OutputQuery struct {
	Outputs OutputConnection `json:"outputs"`
}

type OutputConnection struct {
	Edges    []OutputEdge   `json:"edges"`
	PageInfo OutputPageInfo `json:"pageInfo"`
}

type OutputEdge struct {
	Node   Output `json:"node"`
	Cursor string `json:"cursor"`
}

type Output struct {
	Index      int    `json:"index"`
	InputIndex int    `json:"inputIndex"`
	Blob       string `json:"blob"`
}

type OutputPageInfo struct {
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
}
