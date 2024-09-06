package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/calindra/nonodo/internal/commons"
)

func getYAML(v2 string) ([]byte, error) {
	slog.Info("Downloading OpenAPI from", slog.String("url", v2))
	resp, err := http.Get(v2)
	if err != nil {
		return nil, fmt.Errorf("Failed to download OpenAPI from %s: %s", v2, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to download OpenAPI from %s: status code %s", v2, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read OpenAPI from %s: %s", v2, err.Error())
	}

	slog.Info("OpenAPI downloaded successfully")

	// Replace GioResponse with GioResponseRollup
	// Because oapi-codegen will generate the same name for
	// both GioResponse from schema and GioResponse from client
	// https://github.com/deepmap/oapi-codegen/issues/386
	str := string(data)
	str = strings.ReplaceAll(str, "GioResponse", "GioResponseRollup")
	return []byte(str), nil
}

func checkErr(context string, err error) {
	if err != nil {
		log.Fatal(context, ": ", err)
	}
}

func main() {
	commons.ConfigureLog(slog.LevelDebug)
	v2URL := "https://raw.githubusercontent.com/cartesi/openapi-interfaces/v0.9.0/rollup.yaml"
	inspectURL := "https://raw.githubusercontent.com/cartesi/rollups-node/v1.4.0/api/openapi/inspect.yaml"

	v2, wrong := getYAML(v2URL)
	checkErr("download v2", wrong)
	inspect, wrong := getYAML(inspectURL)
	checkErr("download inspect", wrong)

	var filemode os.FileMode = 0644

	err := os.WriteFile("rollup.yaml", v2, filemode)
	if err != nil {
		panic("Failed to write OpenAPI v2 to file: " + err.Error())
	}

	err = os.WriteFile("inspect.yaml", inspect, filemode)
	if err != nil {
		panic("Failed to write OpenAPI inspect to file: " + err.Error())
	}

	slog.Info("OpenAPI written to file")
}
