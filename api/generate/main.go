package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func getYAML(v2 string) []byte {
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

	// Replace GioResponse with GioResponseRollup
	// Because oapi-codegen will generate the same name for
	// both GioResponse from schema and GioResponse from client
	// https://github.com/deepmap/oapi-codegen/issues/386
	var str = string(data)
	str = strings.ReplaceAll(str, "GioResponse", "GioResponseRollup")
	return []byte(str)
}

func main() {
	// v2URL := "https://raw.githubusercontent.com/cartesi/openapi-interfaces/v0.8.0/rollup.yaml"
	v2URL := "https://raw.githubusercontent.com/cartesi/openapi-interfaces/fix/http-server/rollup.yaml"

	v2 := getYAML(v2URL)

	var filemode os.FileMode = 0644

	err := os.WriteFile("rollup.yaml", v2, filemode)
	if err != nil {
		panic("Failed to write OpenAPI v2 to file: " + err.Error())
	}

	log.Println("OpenAPI written to file")
}
