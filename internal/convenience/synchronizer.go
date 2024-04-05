package convenience

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Synchronizer struct {
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

	for {
		// Make GraphQL query
		query := `query {
			vouchers(last: 10) {
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
			}
		}`

		payload, err := json.Marshal(map[string]interface{}{
			"operationName": nil,
			"query":         query,
			"variables":     map[string]interface{}{},
		})
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

		// Parse GraphQL response
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Println("Error parsing JSON:", err)
			continue
		}

		// Handle response data
		// ...
		fmt.Println(string(body))
		// Wait for a certain duration before making the next request (e.g., 5 seconds)
		time.Sleep(5 * time.Second)
	}
}
