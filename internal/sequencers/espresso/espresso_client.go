package espresso

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tyler-smith/go-bip39"
)

type EspressoClient struct {
	EspressoUrl string
	GraphQLUrl  string
}

type EIP712Domain struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	ChainId           uint32 `json:"chainId"`
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

func (e *EspressoClient) SendInput(payload string, namespace int) {
	mnemonic := "test test test test test test test test test test test junk"
	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatalf("Error connection to the node: %v", err)
	}

	// Generate private key from mnemonic
	privateKey, err := getPrivateKeyFromMnemonic(mnemonic)
	if err != nil {
		log.Fatalf("Error deriving private key: %v", err)
	}

	// Sign and Send Espresso Input
	response, err := addEspressoInput(e, client, privateKey, namespace, payload)
	if err != nil {
		log.Fatalf("Error sending to Espresso: %v", err)
	}
	fmt.Println("Transaction received:", response)
}

func addEspressoInput(e *EspressoClient, client *ethclient.Client, privateKey *ecdsa.PrivateKey, namespace int, payload string) (string, error) {
	// Get nonce
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	nonce, err := fetchNonce(fromAddress.Hex(), e.GraphQLUrl)
	if err != nil {
		return "", fmt.Errorf("Error getting nonce: %v", err)
	}

	espressoMessage := EspressoMessage{
		Nonce:   nonce,
		Payload: "0x" + payload,
	}

	// Build Message
	typedData := EspressoData{
		Account: fromAddress.Hex(),
		Message: espressoMessage,
		Domain: EIP712Domain{
			Name:              "EspressoM",
			Version:           "1",
			ChainId:           HARDHAT,
			VerifyingContract: "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
		},
		PrimaryType: "EspressoMessage",
		Types: Types{
			EIP712Domain: []TypeDetail{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint32"},
				{Name: "verifyingContract", Type: "address"},
			},
			EspressoMessage: []TypeDetail{
				{Name: "nonce", Type: "uint64"},
				{Name: "payload", Type: "string"},
			},
		},
	}

	// Sign typed data
	signature, err := signTypedData(privateKey, typedData)

	if err != nil {
		return "", fmt.Errorf("Error getting signature: %v", err)
	}

	signedMessage, err := createSignedMessage(signature, typedData)

	if err != nil {
		return "", fmt.Errorf("Error signing message: %v", err)
	}

	// Send signed message to Espresso
	response, err := submitToEspresso(e, namespace, signedMessage)
	if err != nil {
		return "", fmt.Errorf("Error sending input to Espresso: %v", err)
	}

	fmt.Println("Input sent to Espresso successfully!")
	return response, nil
}

func fetchNonce(sender string, graphqlURL string) (uint64, error) {
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
		return 0, fmt.Errorf("Error serializng GraphQL query: %v", err)
	}

	resp, err := http.Post(graphqlURL+"/graphql", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, fmt.Errorf("Error doing graphql request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("Error reading graphql response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Request failed with status: %d, corpo: %s", resp.StatusCode, string(body))
	}

	var graphqlResponse GraphQLResponse
	err = json.Unmarshal(body, &graphqlResponse)
	if err != nil {
		return 0, fmt.Errorf("Error deserializing GraphQL response: %v", err)
	}

	nextNonce := graphqlResponse.Data.Inputs.TotalCount + 1
	return uint64(nextNonce), nil
}

func signTypedData(privateKey *ecdsa.PrivateKey, typedData EspressoData) (string, error) {
	dataBytes, err := json.Marshal(typedData)
	if err != nil {
		return "", fmt.Errorf("Error serializing typed data: %v", err)
	}

	hash := crypto.Keccak256Hash(dataBytes)

	signature, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return "", fmt.Errorf("Error signing data: %v", err)
	}

	return fmt.Sprintf("0x%x", signature), nil
}

func submitToEspresso(e *EspressoClient, namespace int, signedMessage string) (string, error) {
	payload := map[string]interface{}{
		"namespace": namespace,
		"payload":   base64.StdEncoding.EncodeToString([]byte(signedMessage)),
	}

	payloadBytes, err := json.Marshal(payload)
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

func getPrivateKeyFromMnemonic(mnemonic string) (*ecdsa.PrivateKey, error) {
	seed := bip39.NewSeed(mnemonic, "")

	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("Fail to generate master key: %w", err)
	}

	childKey, err := masterKey.Child(hdkeychain.HardenedKeyStart + PURPOSE_INDEX)
	if err != nil {
		return nil, fmt.Errorf("Fail to derive key: %w", err)
	}
	childKey, err = childKey.Child(hdkeychain.HardenedKeyStart + COIN_TYPE_INDEX)
	if err != nil {
		return nil, fmt.Errorf("Fail to derive key: %w", err)
	}
	childKey, err = childKey.Child(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, fmt.Errorf("Fail to derive key: %w", err)
	}
	childKey, err = childKey.Child(0)
	if err != nil {
		return nil, fmt.Errorf("Fail to derive key: %w", err)
	}
	childKey, err = childKey.Child(0)
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

func createSignedMessage(signature string, typedData EspressoData) (string, error) {
	typedDataJSON, err := json.Marshal(typedData)
	if err != nil {
		return "", err
	}

	typedDataBase64 := base64.StdEncoding.EncodeToString(typedDataJSON)

	signedMessage := SigAndData{
		Signature: signature,
		TypedData: typedDataBase64,
	}

	signedMessageJSON, err := json.Marshal(signedMessage)
	if err != nil {
		return "", err
	}

	return string(signedMessageJSON), nil
}
