package dataavailability

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
)

var (
	BLOBSCAN_API_URL = "https://api.sepolia.blobscan.com"
)

type BlobFetcher struct {
}

type BlobsResponseOk struct {
	Data string `json:"data"`
}

type BlobsResponseError struct {
	Message string `json:"message"`
}

func NewBlobFetcher() Fetch {
	return &BlobFetcher{}
}

func (e *BlobFetcher) Fetch(ctx echo.Context, id string) (*string, *HttpCustomError) {

	idRegexp, err := regexp.Compile("^0x[0-9a-fA-F]{64}$")

	if err != nil {
		message := "Failed to compile regexp for GIO request ID"
		return nil, NewHttpCustomError(http.StatusInternalServerError, &message)
	}

	if !idRegexp.MatchString(id) {
		message := fmt.Sprintf("Expected 32-byte hex string for GIO request ID, got %s", id)
		return nil, NewHttpCustomError(http.StatusBadRequest, &message)
	}

	method := "GET"

	url := BLOBSCAN_API_URL + "/blobs/" + id

	request, err := http.NewRequest(method, url, nil)

	if err != nil {
		message := fmt.Sprintf("Failed to create HTTP request object: %s", err)
		return nil, NewHttpCustomError(http.StatusInternalServerError, &message)
	}

	request.Header.Set("accept", "application/json")

	client := &http.Client{}

	slog.Info("Sent HTTP request", "method", method, "url", url)

	response, err := client.Do(request)

	if err != nil {
		message := fmt.Sprintf("Failed to perform HTTP request: %s", err)
		return nil, NewHttpCustomError(http.StatusInternalServerError, &message)
	}

	slog.Info("Got HTTP response", "status", response.Status)

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {
		message := fmt.Sprintf("Failed to read HTTP body: %s", err)
		return nil, NewHttpCustomError(http.StatusInternalServerError, &message)
	}

	if response.StatusCode != http.StatusOK {
		var responseError BlobsResponseError

		err = json.Unmarshal(body, &responseError)

		if err != nil {
			message := fmt.Sprintf("Failed to unmarshal HTTP response body as JSON: %s", err)
			return nil, NewHttpCustomError(http.StatusInternalServerError, &message)
		}

		message := fmt.Sprintf("HTTP request returned with error: %s", responseError.Message)
		return nil, NewHttpCustomError(response.StatusCode, &message)
	}

	var responseOk BlobsResponseOk

	err = json.Unmarshal(body, &responseOk)

	if err != nil {
		message := fmt.Sprintf("Failed to unmarshal HTTP response body as JSON: %s", err)
		return nil, NewHttpCustomError(http.StatusInternalServerError, &message)
	}

	return &responseOk.Data, nil
}
