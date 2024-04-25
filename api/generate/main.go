package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	url := "https://raw.githubusercontent.com/cartesi/openapi-interfaces/v0.8.0/rollup.yaml"

	log.Println("Downloading OpenAPI from", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("Failed to download OpenAPI from", url, ":", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatal("Failed to download OpenAPI from", url, ": status code", resp.StatusCode)
		return
	}

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatal("Failed to read OpenAPI from", url, ":", err)
		return
	}

	log.Println("OpenAPI downloaded successfully")

	err = os.WriteFile("rollup.yaml", data, 0644)
	if err != nil {
		log.Fatal("Failed to write OpenAPI to file:", err)
		return
	}

	log.Println("OpenAPI written to file")
}
