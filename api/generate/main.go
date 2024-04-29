package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func getYAML(v2 string, isV2 bool) []byte {
	log.Println("Downloading OpenAPI from", v2)
	resp, err := http.Get(v2)
	if err != nil {
		panic("Failed to download OpenAPI from" + v2 + ":" + err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		panic("Failed to download OpenAPI from " + v2 + ": status code " + resp.Status)
	}

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		panic("Failed to read OpenAPI from " + v2 + ": " + err.Error())
	}

	log.Println("OpenAPI downloaded successfully")

	if isV2 {
		// Replace GioResponse with GioResponseRollup
		// Because oapi-codegen will generate the same name for
		// both GioResponse from schema and GioResponse from client
		// https://github.com/deepmap/oapi-codegen/issues/386
		var str = string(data)
		str = strings.ReplaceAll(str, "GioResponse", "GioResponseRollup")
		return []byte(str)
	}

	return data
}

func main() {
	v2URL := "https://raw.githubusercontent.com/cartesi/openapi-interfaces/v0.8.0/rollup.yaml"
	v1URL := "https://raw.githubusercontent.com/cartesi/openapi-interfaces/v0.7.3/rollup.yaml"

	v1 := getYAML(v1URL, false)
	v2 := getYAML(v2URL, true)

	var filemode os.FileMode = 0644

	err := os.WriteFile("rollup.yaml", v2, filemode)
	if err != nil {
		panic("Failed to write OpenAPI v2 to file: " + err.Error())
	}

	err = os.WriteFile("rollup-v1.yaml", v1, filemode)
	if err != nil {
		panic("Failed to write OpenAPI v1 to file: " + err.Error())
	}

	log.Println("OpenAPI written to file")
}
