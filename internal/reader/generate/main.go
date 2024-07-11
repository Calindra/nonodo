package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/calindra/nonodo/internal/commons"
)

const (
	urlReader = "https://github.com/cartesi/rollups-node/releases/download/v1.5.0-rc1/schema.graphql"
)

// Exit if there is any error.
func checkErr(context string, err any) {
	if err != nil {
		log.Fatal(context, ": ", err)
	}
}

func downloadSchema(url string) ([]byte, error) {
	log.Print("downloading schema from ", url)
	response, err := http.Get(url)
	checkErr("download schema", err)
	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()
		return nil, fmt.Errorf("invalid status: %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	checkErr("read schema", err)

	defer response.Body.Close()

	return body, nil
}

func main() {
	commons.ConfigureLog(slog.LevelDebug)

	body, err := downloadSchema(urlReader)
	checkErr("Error while download", err)

	var filemode os.FileMode = 0644
	err = os.WriteFile("../../api/reader.graphile.graphql", body, filemode)
	checkErr("Error while writing", err)

	slog.Info("Schema downloaded and saved to reader.graphile.yaml")
}
