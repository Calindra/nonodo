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
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

const (
	celestiaUrl         = "https://raw.githubusercontent.com/miltonjonat/rollups-celestia/main/onchain/deployments/sepolia/CelestiaRelay.json"
	openzeppelin        = "https://registry.npmjs.org/@openzeppelin/contracts/-/contracts-5.1.0.tgz"
	rollupsContractsUrl = "https://registry.npmjs.org/@cartesi/rollups/-/rollups-2.0.0-rc.12.tgz"
	baseContractsPath   = "package/export/artifacts/contracts/"
	bindingPkg          = "contracts"
)

type contractBinding struct {
	jsonPath string
	typeName string
	outFile  string
}

var bindingsOpenZeppelin = []contractBinding{
	{
		jsonPath: "package/build/contracts/EIP712.json",
		typeName: "EIP712",
		outFile:  "eip712.go",
	},
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
	{
		jsonPath: baseContractsPath + "dapp/IApplicationFactory.sol/IApplicationFactory.json",
		typeName: "IApplicationFactory",
		outFile:  "application_factory.go",
	},
	{
		jsonPath: baseContractsPath + "consensus/authority/IAuthorityFactory.sol/IAuthorityFactory.json",
		typeName: "IAuthorityFactory",
		outFile:  "authority_factory.go",
	},
	{
		jsonPath: baseContractsPath + "consensus/IConsensus.sol/IConsensus.json",
		typeName: "IConsensus",
		outFile:  "consensus.go",
	},
}

func main() {
	commons.ConfigureLog(slog.LevelDebug)
	contractsZip, err := downloadContracts(rollupsContractsUrl)
	checkErr("download contracts", err)
	defer contractsZip.Close()
	contractsTar, err := unzip(contractsZip)
	checkErr("unzip contracts", err)
	defer contractsTar.Close()

	contractsOpenZeppelin, err := downloadContracts(openzeppelin)
	checkErr("download contracts", err)
	defer contractsOpenZeppelin.Close()
	contractsTarOpenZeppelin, err := unzip(contractsOpenZeppelin)
	checkErr("unzip contracts", err)
	defer contractsTarOpenZeppelin.Close()

	contractJson := downloadJsonContract(celestiaUrl)
	defer contractJson.Close()

	files := make(map[string]bool)
	for _, b := range bindings {
		files[b.jsonPath] = true
	}
	contents, err := readFilesFromTar(contractsTar, files)
	checkErr("read files from tar", err)

	filesZ := make(map[string]bool)
	for _, b := range bindingsOpenZeppelin {
		filesZ[b.jsonPath] = true
	}
	contentZ, err := readFilesFromTar(contractsTarOpenZeppelin, filesZ)
	checkErr("read files from tar", err)

	for contract, content := range contentZ {
		contents[contract] = content
	}

	content := readJson(contractJson)
	contents[baseContractsPath+"sepolia/CelestiaRelay.json"] = content
	bindings = append(bindings, contractBinding{
		jsonPath: baseContractsPath + "sepolia/CelestiaRelay.json",
		typeName: "CelestiaRelay",
		outFile:  "celestia_relay.go",
	})
	bindings = append(bindings, bindingsOpenZeppelin...)

	for _, b := range bindings {
		content := contents[b.jsonPath]
		if content == nil {
			log.Fatal("missing contents for ", b.jsonPath)
		}
		generateBinding(b, content)
	}

	slog.Info("done")
}

// Exit if there is any error.
func checkErr(context string, err any) {
	if err != nil {
		log.Fatal(context, ": ", err)
	}
}

func downloadJsonContract(url string) io.ReadCloser {
	log.Print("downloading contracts from ", url)
	response, err := http.Get(url)
	checkErr("download json", err)
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		log.Fatal("invalid status: ", response.Status)
	}
	return response.Body
}

// Download the contracts from rollupsContractsUrl.
// Return the buffer with the contracts.
func downloadContracts(url string) (io.ReadCloser, error) {
	slog.Info("downloading contracts from ", slog.String("url", url))
	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download contracts from %s: %s", url, err.Error())
	}
	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()
		return nil, fmt.Errorf("failed to download contracts from %s: status code %s", url, response.Status)
	}
	return response.Body, nil
}

// Decompress the buffer with the contracts.
func unzip(r io.Reader) (io.ReadCloser, error) {
	slog.Info("unziping contracts")
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return gzipReader, nil
}

func readJson(r io.Reader) []byte {
	content, err := io.ReadAll(r)
	checkErr("read json", err)
	return content
}

// Read the required files from the tar.
// Return a map with the file contents.
func readFilesFromTar(r io.Reader, files map[string]bool) (map[string][]byte, error) {
	contents := make(map[string][]byte)
	tarReader := tar.NewReader(r)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, fmt.Errorf("error while reading tar: %s", err)
		}
		if files[header.Name] {
			contents[header.Name], err = io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("error while reading file inside tar: %s", err)
			}
		}
	}
	return contents, nil
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
	slog.Info("generated binding ", slog.String("file", b.outFile))
}
