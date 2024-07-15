package synchronizer

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/calindra/nonodo/internal/graphile"
)

const outputQuery = `
	query { 
		outputs(first: %d) {
			edges { 
				cursor
				node { 
					blob
					index
					inputIndex
					nodeId
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
		%s	

		#report
		%s
	}`

const inputQuery = `
	inputs(first: %d) {
		edges {
			cursor
			node {
				blob
				index
				nodeId
			}
		}
		pageInfo {
			endCursor
			hasNextPage
			hasPreviousPage
			startCursor
		}
	}
`

const inputQueryWithCursor = `
	inputs(first: %d, after: %s) {
		edges {
			cursor
			node {
				blob
				index
				nodeId
			}
		}
		pageInfo {
			endCursor
			hasNextPage
			hasPreviousPage
			startCursor
		}
	}
`

const outputQueryWithCursor = `
	query { 
		outputs(first: %d, after: %s ) { 
			edges {
				cursor
				node {
					blob
					index
					inputIndex
					nodeId
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
		%s	

		#report
		%s
	}`

const reportQuery = `
	reports(first: %d) {
		edges {
			node {
				blob
				index
				inputIndex
			}
		}
	}
`
const reportQueryWithCursor = `
	reports(first: %d, after: %s) {
		edges {
			node {
				blob
				index
				inputIndex
			}
		}
	}
`

const DefaultQueryBatchSize = 10

const ErrorSendingGraphileRequest = `
+-----------------------------------------------------------+
| Please ensure that the rollups-node is up and running at: |
GRAPH_QL_URL
+-----------------------------------------------------------+
`

type GraphileFetcher struct {
	Url               string
	CursorAfter       string
	CursorInputAfter  string
	CursorReportAfter string
	BatchSize         int
	Query             string
	QueryWithCursor   string
	GraphileClient    graphile.GraphileClient
}

func NewGraphileFetcher(graphileClient graphile.GraphileClient) *GraphileFetcher {
	return &GraphileFetcher{
		Url:             "http://localhost:5000/graphql",
		CursorAfter:     "",
		BatchSize:       DefaultQueryBatchSize,
		Query:           outputQuery,
		QueryWithCursor: outputQueryWithCursor,
		GraphileClient:  graphileClient,
	}
}

func (v *GraphileFetcher) GetReportQuery() string {
	if len(v.CursorReportAfter) > 0 {
		return fmt.Sprintf(reportQueryWithCursor, v.BatchSize, v.CursorReportAfter)
	} else {
		return fmt.Sprintf(reportQuery, v.BatchSize)
	}
}

func (v *GraphileFetcher) Fetch() (*OutputResponse, error) {
	slog.Debug("Graphile querying",
		"afterOutput", v.CursorAfter,
		"afterInput", v.CursorInputAfter,
		"afterReport", v.CursorReportAfter,
	)

	var query string

	reportQueryWithVars := v.GetReportQuery()
	if len(v.CursorAfter) > 0 {
		if len(v.CursorInputAfter) > 0 {
			query = fmt.Sprintf(outputQueryWithCursor, v.BatchSize, v.CursorAfter, fmt.Sprintf(inputQueryWithCursor, v.BatchSize, v.CursorInputAfter), reportQueryWithVars)
		} else {
			query = fmt.Sprintf(outputQueryWithCursor, v.BatchSize, v.CursorAfter, fmt.Sprintf(inputQuery, v.BatchSize), reportQueryWithVars)
		}
	} else {
		if len(v.CursorInputAfter) > 0 {
			query = fmt.Sprintf(outputQuery, v.BatchSize, fmt.Sprintf(inputQueryWithCursor, v.BatchSize, v.CursorInputAfter), reportQueryWithVars)
		} else {
			query = fmt.Sprintf(outputQuery, v.BatchSize, fmt.Sprintf(inputQuery, v.BatchSize), reportQueryWithVars)
		}
	}

	payload, err := json.Marshal(map[string]interface{}{
		"query": query,
	})
	if err != nil {
		slog.Error("Error marshalling JSON:", "error", err)
		return nil, err
	}

	// Make a POST request to the GraphQL endpoint

	body, err := v.GraphileClient.Post(payload)

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
		Inputs struct {
			PageInfo struct {
				StartCursor     string `json:"startCursor"`
				EndCursor       string `json:"endCursor"`
				HasNextPage     bool   `json:"hasNextPage"`
				HasPreviousPage bool   `json:"hasPreviousPage"`
			}
			Edges []struct {
				Cursor string `json:"cursor"`
				Node   struct {
					Index int    `json:"index"`
					Blob  string `json:"blob"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"inputs"`
	} `json:"data"`
}
