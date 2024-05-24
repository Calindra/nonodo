package reader

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
)

type HTTPClientImpl struct{}

const GRAPHILE_ENDPOINT = "http://localhost:5001/graphql"

func (c *HTTPClientImpl) Post(requestBody []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", GRAPHILE_ENDPOINT, bytes.NewBuffer(requestBody))
	if err != nil {
		slog.Error("Error creating request", "error", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		slog.Error("Error executing request:", "error", err)
		return nil, err
	}

	defer resp.Body.Close()

	// Lê o corpo da resposta
	responseBody, err := io.ReadAll(resp.Body)

	if err != nil {
		slog.Error("Error reading body:", "error", err)
		return nil, err
	}

	return responseBody, err
}
