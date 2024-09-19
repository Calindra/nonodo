package espresso

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/signer/core"
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
	// Cria um transactor com a chave privada
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(HARDHAT)) // Mainnet

	if err != nil {
		log.Fatalf("Failed to create transactor: %v", err)
	}

	// Defina o domínio EIP-712
	domain := apitypes.TypedDataDomain{
		Name:              "EspressoM",
		Version:           "1",
		ChainId:           math.NewHexOrDecimal256(HARDHAT),
		VerifyingContract: "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
	}

	// Deriva o endereço do signatário a partir da chave privada
	fromAddress := auth.From

	nonce, err := fetchNonce(fromAddress.Hex(), e.GraphQLUrl)
	if err != nil {
		return "", fmt.Errorf("Error getting nonce: %v", err)
	}

	log.Printf("O tipo de nonce é: %T\n", nonce)

	espressoMessage := map[string]interface{}{
		"nonce":   math.NewHexOrDecimal256(nonce),
		"payload": "0x" + payload,
	}

	// Defina os tipos EIP-712
	typesMap := map[string][]apitypes.Type{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"EspressoMessage": {
			{Name: "nonce", Type: "uint256"},
			{Name: "payload", Type: "string"},
		},
	}

	typedData := apitypes.TypedData{
		Types:       typesMap,
		PrimaryType: "EspressoMessage",
		Domain:      domain,
		Message:     espressoMessage,
	}

	// Crie o SignerAPI
	api := core.NewSignerAPI(accounts.NewManager(nil), int64(HARDHAT), true, nil, nil, false, nil)

	// Assine os dados
	signature, err := api.SignTypedData(context.Background(), common.NewMixedcaseAddress(fromAddress), typedData)

	if err != nil {
		log.Fatalf("Error signing typed data: %v", err)
	}

	signedMessage, err := createSignedMessage(string(signature), typedData)

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

func fetchNonce(sender string, graphqlURL string) (int64, error) {
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

	// Verifique o corpo da resposta para depuração
	log.Printf("Corpo da resposta GraphQL: %s", string(body))

	var graphqlResponse GraphQLResponse
	err = json.Unmarshal(body, &graphqlResponse)
	if err != nil {
		return 0, fmt.Errorf("Error deserializing GraphQL response: %v", err)
	}

	// Verifique se o TotalCount está correto
	log.Printf("TotalCount retornado: %d", graphqlResponse.Data.Inputs.TotalCount)

	// Incrementando o nonce
	nextNonce := graphqlResponse.Data.Inputs.TotalCount + 1
	log.Printf("Nonce gerado: %d", nextNonce)

	return int64(nextNonce), nil
}

func hashEIP712Domain(domain EIP712Domain) common.Hash {
	domainTypeHash := crypto.Keccak256Hash([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	nameHash := crypto.Keccak256Hash([]byte(domain.Name))
	versionHash := crypto.Keccak256Hash([]byte(domain.Version))
	chainIdBytes := new(big.Int).SetUint64(domain.ChainId).Bytes()
	verifyingContractBytes := common.HexToAddress(domain.VerifyingContract).Bytes()

	return crypto.Keccak256Hash(
		domainTypeHash.Bytes(),
		nameHash.Bytes(),
		versionHash.Bytes(),
		crypto.Keccak256Hash(chainIdBytes).Bytes(),
		crypto.Keccak256Hash(verifyingContractBytes).Bytes(),
	)
}

func hashEspressoMessage(message EspressoMessage) common.Hash {
	messageTypeHash := crypto.Keccak256Hash([]byte("EspressoMessage(uint64 nonce,string payload)"))
	nonceBytes := new(big.Int).SetUint64(message.Nonce).Bytes()
	payloadHash := crypto.Keccak256Hash([]byte(message.Payload))

	return crypto.Keccak256Hash(
		messageTypeHash.Bytes(),
		crypto.Keccak256Hash(nonceBytes).Bytes(),
		payloadHash.Bytes(),
	)
}

// Função principal para assinar os dados
func signTypedData(privateKey *ecdsa.PrivateKey, typedData EspressoData) (string, error) {
	// Hash do domínio
	domainSeparator := hashEIP712Domain(typedData.Domain)

	// Hash da mensagem
	messageHash := hashEspressoMessage(typedData.Message)

	// Prefixo EIP-712
	typedDataHash := crypto.Keccak256Hash(
		[]byte("\x19\x01"),
		domainSeparator.Bytes(),
		messageHash.Bytes(),
	)

	// Assinar o hash combinado
	signature, err := crypto.Sign(typedDataHash.Bytes(), privateKey)
	if err != nil {
		return "", fmt.Errorf("Error signing data: %v", err)
	}

	// Ajustar o valor 'v' na assinatura EIP-712 (v = 27 ou 28)
	signature[64] += 27

	// Retornar a assinatura no formato hexadecimal
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

func createSignedMessage(signature string, typedData apitypes.TypedData) (string, error) {
	typedDataJSON, err := json.Marshal(typedData)
	if err != nil {
		return "", err
	}

	typedDataBase64 := base64.StdEncoding.EncodeToString(typedDataJSON)

	signedMessage := map[string]interface{}{
		"signature": signature,
		"typedData": typedDataBase64,
	}

	signedMessageJSON, err := json.Marshal(signedMessage)
	if err != nil {
		return "", err
	}

	return string(signedMessageJSON), nil
}
