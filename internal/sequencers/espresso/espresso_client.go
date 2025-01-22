package espresso

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strconv"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/sequencers/paiodecoder"
	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/tyler-smith/go-bip39"
)

type EspressoClient struct {
	EspressoUrl string
	GraphQLUrl  string
}

type EIP712Domain struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	ChainId           uint64 `json:"chainId"`
	VerifyingContract string `json:"verifyingContract"`
}

type EspressoMessage struct {
	Nonce   uint64 `json:"nonce"`
	Payload string `json:"payload"`
}

type Types struct {
	EIP712Domain    []TypeDetail `json:"EIP712Domain"`
	EspressoMessage []TypeDetail `json:"EspressoMessage"`
}

type TypeDetail struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type EspressoData struct {
	Account     string          `json:"account"`
	Domain      EIP712Domain    `json:"domain"`
	Types       Types           `json:"types"`
	PrimaryType string          `json:"primaryType"`
	Message     EspressoMessage `json:"message"`
}

type GraphQLQuery struct {
	Query string `json:"query"`
}

type GraphQLResponse struct {
	Data struct {
		Inputs struct {
			TotalCount int `json:"totalCount"`
		} `json:"inputs"`
	} `json:"data"`
}

const (
	HARDHAT         = 31337
	PURPOSE_INDEX   = 44
	COIN_TYPE_INDEX = 60
)

// Implement the hashing function based on EIP-712 requirements
func HashEIP712Message(data apitypes.TypedData) ([]byte, error) {
	hash, _, err := apitypes.TypedDataAndHash(data)
	if err != nil {
		return []byte(""), err
	}
	return hash, nil
}

// Sign the hash with the private key
func SignMessage(hash []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func (e *EspressoClient) SendInput(payload string, namespace int) (string, error) {
	mnemonic := "test test test test test test test test test test test junk"

	// Generate private key from mnemonic
	privateKey, err := getPrivateKeyFromMnemonic(mnemonic)

	if err != nil {
		log.Fatalf("Error getting privateKey %v", err)
	}

	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	log.Printf("From address %s", fromAddress.Hex())
	log.Printf("payload %s", payload)

	nonce, err := fetchNonce(fromAddress.Hex(), e.GraphQLUrl)

	if err != nil {
		log.Fatalf("espresso error getting nonce: %v", err)
	}

	// Build Message
	n, err := strconv.Atoi(nonce)
	if err != nil {
		panic(err)
	}
	maxGasPrice := 10
	data := paiodecoder.CreateTypedData(
		common.HexToAddress(devnet.ApplicationAddress),
		uint64(n), big.NewInt(int64(maxGasPrice)),
		common.Hex2Bytes(payload),
		big.NewInt(HARDHAT),
	)

	typedDataJSON, err := json.Marshal(data)
	if err != nil {
		log.Fatal("json error:", err)
	}
	log.Printf("data %s", typedDataJSON)

	// Hash the message
	messageHash, err := HashEIP712Message(data)
	if err != nil {
		log.Fatal("Error hashing message:", err)
	}

	// Sign the message
	signature, err := SignMessage(messageHash, privateKey)
	if err != nil {
		log.Fatal("Error signing message:", err)
	}

	if err != nil {
		log.Fatal("Error signing message:", err)
	}

	typedDataBase64 := base64.StdEncoding.EncodeToString(typedDataJSON)

	signature[64] += 27
	sigAndData := commons.SigAndData{
		Signature: "0x" + common.Bytes2Hex(signature),
		TypedData: typedDataBase64,
	}
	return e.SubmitSigAndData(namespace, sigAndData)
}

func (e EspressoClient) SubmitSigAndData(namespace int, sigAndData commons.SigAndData) (string, error) {
	jsonPayload, err := json.Marshal(sigAndData)

	if err != nil {
		log.Fatal("Error json.Marshal message:", err)
	}

	// Será que precisa de outro encoding base 64 aqui
	espressoPayload := map[string]interface{}{
		"namespace": namespace,
		"payload":   base64.StdEncoding.EncodeToString([]byte(jsonPayload)),
	}

	payloadBytes, err := json.Marshal(espressoPayload)
	if err != nil {
		return "", fmt.Errorf("Error serializing JSON: %v", err)
	}

	resp, err := http.Post(e.EspressoUrl+"/v0/submit/submit", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("Error sending HTTP Request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("Error reading response body: %v", err)
		}

		bodyString := string(bodyBytes)
		log.Print(bodyString)
		return "", fmt.Errorf("Request failed with status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading response body: %v", err)
	}

	bodyString := string(bodyBytes)

	return bodyString, nil
}

func fetchNonce(sender string, graphqlURL string) (string, error) {
	query := fmt.Sprintf(`
		{
			inputs(where: {msgSender: "%s" type: "Espresso"}) {
				totalCount
			}
		}`, sender)

	requestBody := GraphQLQuery{
		Query: query,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("Error serializng GraphQL query: %v", err)
	}

	resp, err := http.Post(graphqlURL+"/graphql", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("Error doing graphql request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading graphql response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Request failed with status: %d, corpo: %s", resp.StatusCode, string(body))
	}

	var graphqlResponse GraphQLResponse
	err = json.Unmarshal(body, &graphqlResponse)
	if err != nil {
		return "", fmt.Errorf("Error deserializing GraphQL response: %v", err)
	}

	nextNonce := graphqlResponse.Data.Inputs.TotalCount + 1
	return fmt.Sprintf("%d", nextNonce), nil
}

func getPrivateKeyFromMnemonic(mnemonic string) (*ecdsa.PrivateKey, error) {
	seed := bip39.NewSeed(mnemonic, "")

	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("Fail to generate master key: %w", err)
	}

	childKey, err := masterKey.Derive(hdkeychain.HardenedKeyStart + PURPOSE_INDEX)
	if err != nil {
		return nil, fmt.Errorf("Fail to derive key: %w", err)
	}
	childKey, err = childKey.Derive(hdkeychain.HardenedKeyStart + COIN_TYPE_INDEX)
	if err != nil {
		return nil, fmt.Errorf("Fail to derive key: %w", err)
	}
	childKey, err = childKey.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, fmt.Errorf("Fail to derive key: %w", err)
	}
	childKey, err = childKey.Derive(0)
	if err != nil {
		return nil, fmt.Errorf("Fail to derive key: %w", err)
	}
	childKey, err = childKey.Derive(0)
	if err != nil {
		return nil, fmt.Errorf("Fail to derive key: %w", err)
	}

	privKeyBytes, err := childKey.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("Fail to obtain private key: %w", err)
	}

	privateKey, err := crypto.ToECDSA(privKeyBytes.Serialize())
	if err != nil {
		return nil, fmt.Errorf("Fail to convert to ECDSA key: %w", err)
	}

	return privateKey, nil
}
