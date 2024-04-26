package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	url := "https://raw.githubusercontent.com/cartesi/openapi-interfaces/v0.8.0/rollup.yaml"

	log.Println("Downloading OpenAPI from", url)
	resp, err := http.Get(url)
	if err != nil {
		panic("Failed to download OpenAPI from" + url + ":" + err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		panic("Failed to download OpenAPI from " + url + ": status code " + resp.Status)
	}

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		panic("Failed to read OpenAPI from " + url + ": " + err.Error())
	}

	log.Println("OpenAPI downloaded successfully")

	// Replace GioResponse with GioResponseRollup
	// Because oapi-codegen will generate the same name for both GioResponse from schema and GioResponse from client
	var str = string(data)
	str = strings.ReplaceAll(str, "GioResponse", "GioResponseRollup")

	err = os.WriteFile("rollup.yaml", []byte(str), 0644)
	if err != nil {
		panic("Failed to write OpenAPI to file: " + err.Error())
	}

	log.Println("OpenAPI written to file")
}
