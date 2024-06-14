// (c) Cartesi and individual authors (see AUTHORS)
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)
//
// This file was obtained from github.com/cartesi/rollups-node
//
// AUTHORS file:
// Gabriel de Quadros Ligneul <8294320+gligneul@users.noreply.github.com>
// Guilherme Dantas <guidanoli@proton.me>
// Zehui Zheng <work996mail@gmail.com>

// This binary generates the Go bindings for the Cartesi Rollups contracts.
// This binary should be called with `go generate` in the parent dir.
// First, it downloads the Cartesi Rollups npm package containing the contracts.
// Then, it generates the bindings using abi-gen.
// Finally, it stores the bindings in the current directory.
package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

const (
	rollupsContractsUrl = "https://registry.npmjs.org/@cartesi/rollups/-/rollups-2.0.0-rc.3.tgz"
	baseContractsPath   = "package/export/artifacts/contracts/"
	bindingPkg          = "contracts"
)

type contractBinding struct {
	jsonPath string
	typeName string
	outFile  string
}

var bindings = []contractBinding{
	{
		jsonPath: baseContractsPath + "inputs/InputBox.sol/InputBox.json",
		typeName: "InputBox",
		outFile:  "input_box.go",
	},
	{
		jsonPath: baseContractsPath + "common/Inputs.sol/Inputs.json",
		typeName: "Inputs",
		outFile:  "inputs.go",
	},
	{
		jsonPath: baseContractsPath + "dapp/Application.sol/Application.json",
		typeName: "Application",
		outFile:  "application.go",
	},
	{
		jsonPath: baseContractsPath + "common/Outputs.sol/Outputs.json",
		typeName: "Outputs",
		outFile:  "outputs.go",
	},
}

func main() {
	contractsZip := downloadContracts(rollupsContractsUrl)
	defer contractsZip.Close()
	contractsTar := unzip(contractsZip)
	defer contractsTar.Close()

	files := make(map[string]bool)
	for _, b := range bindings {
		files[b.jsonPath] = true
	}
	contents := readFilesFromTar(contractsTar, files)

	for _, b := range bindings {
		content := contents[b.jsonPath]
		if content == nil {
			log.Fatal("missing contents for ", b.jsonPath)
		}
		generateBinding(b, content)
	}
}

// Exit if there is any error.
func checkErr(context string, err any) {
	if err != nil {
		log.Fatal(context, ": ", err)
	}
}

// Download the contracts from rollupsContractsUrl.
// Return the buffer with the contracts.
func downloadContracts(url string) io.ReadCloser {
	log.Print("downloading contracts from ", url)
	response, err := http.Get(url)
	checkErr("download tgz", err)
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		log.Fatal("invalid status: ", response.Status)
	}
	return response.Body
}

// Decompress the buffer with the contracts.
func unzip(r io.Reader) io.ReadCloser {
	log.Print("unziping contracts")
	gzipReader, err := gzip.NewReader(r)
	checkErr("unziping", err)
	return gzipReader
}

// Read the required files from the tar.
// Return a map with the file contents.
func readFilesFromTar(r io.Reader, files map[string]bool) map[string][]byte {
	contents := make(map[string][]byte)
	tarReader := tar.NewReader(r)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		checkErr("read tar", err)
		if files[header.Name] {
			contents[header.Name], err = io.ReadAll(tarReader)
			checkErr("read tar", err)
		}
	}
	return contents
}

// Get the .abi key from the json
func getAbi(rawJson []byte) []byte {
	var contents struct {
		Abi json.RawMessage `json:"abi"`
	}
	err := json.Unmarshal(rawJson, &contents)
	checkErr("decode json", err)
	return contents.Abi
}

// Generate the Go bindings for the contracts.
func generateBinding(b contractBinding, content []byte) {
	var (
		sigs    []map[string]string
		abis    = []string{string(getAbi(content))}
		bins    = []string{""}
		types   = []string{b.typeName}
		libs    = make(map[string]string)
		aliases = make(map[string]string)
	)
	code, err := bind.Bind(types, abis, bins, sigs, bindingPkg, bind.LangGo, libs, aliases)
	checkErr("generate binding", err)
	const fileMode = 0600
	err = os.WriteFile(b.outFile, []byte(code), fileMode)
	checkErr("write binding file", err)
	log.Print("generated binding ", b.outFile)
}
