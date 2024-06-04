package dataavailability

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

type SyscoinClient struct {
	client   *http.Client
	endpoint string
}

func NewSyscoinClient() Fetch {
	url := "https://poda.syscoin.org/vh"

	return &SyscoinClient{
		client:   http.DefaultClient,
		endpoint: url,
	}
}

func NewSyscoinClientMock(endpoint string, client *http.Client) Fetch {
	return &SyscoinClient{
		client,
		endpoint,
	}
}

// example: https://poda.syscoin.org/vh/06310294ee0af7f1ae4c8e19fa509264565fa82ba8c82a7a9074b2abf12a15d9
func (sc *SyscoinClient) Fetch(ctx echo.Context, id string) (*string, *HttpCustomError) {
	slog.Debug("Called FetchSyscoinPoDa")

	full_url := "https://poda.syscoin.org/vh/" + id

	res, err := http.Get(full_url)

	if err != nil {
		return nil, NewHttpCustomError(http.StatusInternalServerError, nil)
	}

	defer res.Body.Close()

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, NewHttpCustomError(http.StatusInternalServerError, nil)
	}

	// Convert the body to string
	str := string(body)

	slog.Debug("Called syscoin PoDa: ")

	return &str, nil
}
