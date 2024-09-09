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
	"net/http"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tyler-smith/go-bip39"
)

type EspressoClient struct {
	EspressoUrl string
}

// Estrutura para o domain no EIP712
type EIP712Domain struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	ChainId           uint32 `json:"chainId"`
	VerifyingContract string `json:"verifyingContract"`
}

// Estrutura para a mensagem Espresso
type EspressoMessage struct {
	Nonce   uint64 `json:"nonce"`
	Payload string `json:"payload"`
}

// Estrutura para o tipo EIP712
type Types struct {
	EIP712Domain    []TypeDetail `json:"EIP712Domain"`
	EspressoMessage []TypeDetail `json:"EspressoMessage"`
}

// Detalhes de cada tipo no EIP712
type TypeDetail struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Estrutura principal que contém o typedData
type EspressoData struct {
	Account     string          `json:"account"`
	Domain      EIP712Domain    `json:"domain"`
	Types       Types           `json:"types"`
	PrimaryType string          `json:"primaryType"`
	Message     EspressoMessage `json:"message"`
}

func (e *EspressoClient) SendInput(payload string, namespace int) {
	mnemonic := "test test test test test test test test test test test junk"
	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatalf("Error connection to the node: %v", err)
	}

	// Generate private key from mnemonic
	privateKey, err := getPrivateKeyFromMnemonic(mnemonic)
	if err != nil {
		log.Fatalf("Erro ao derivar chave privada: %v", err)
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
	nonce, err := fetchNonce(client, privateKey)
	if err != nil {
		return "", fmt.Errorf("erro ao buscar nonce: %v", err)
	}

	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

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
			ChainId:           31337,
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

	fmt.Println("Input sent to Espresso sucessfully!")
	return response, nil
}

func fetchNonce(client *ethclient.Client, privateKey *ecdsa.PrivateKey) (uint64, error) {
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return 0, fmt.Errorf("erro ao obter nonce: %v", err)
	}
	return nonce, nil
}

func signTypedData(privateKey *ecdsa.PrivateKey, typedData EspressoData) (string, error) {
	dataBytes, err := json.Marshal(typedData)
	if err != nil {
		return "", fmt.Errorf("erro ao serializar typed data: %v", err)
	}

	// Criar hash da mensagem
	hash := crypto.Keccak256Hash(dataBytes)

	// Assinar o hash com a chave privada
	signature, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return "", fmt.Errorf("erro ao assinar os dados: %v", err)
	}

	return fmt.Sprintf("0x%x", signature), nil
}

func submitToEspresso(e *EspressoClient, namespace int, signedMessage string) (string, error) {
	payload := map[string]interface{}{
		"namespace": namespace,
		"payload":   base64.StdEncoding.EncodeToString([]byte(signedMessage)),
	}

	// Serializar em JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("erro ao serializar payload: %v", err)
	}

	resp, err := http.Post(e.EspressoUrl+"/v0/submit/submit", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("erro ao enviar requisição HTTP: %v", err)
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

	// Usar parâmetros da rede principal
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("falha ao gerar chave mestre: %w", err)
	}

	// Caminho derivado: m/44'/60'/0'/0/0 (primeira conta)
	childKey, err := masterKey.Child(hdkeychain.HardenedKeyStart + 44)
	if err != nil {
		return nil, fmt.Errorf("falha ao derivar chave: %w", err)
	}
	childKey, err = childKey.Child(hdkeychain.HardenedKeyStart + 60)
	if err != nil {
		return nil, fmt.Errorf("falha ao derivar chave: %w", err)
	}
	childKey, err = childKey.Child(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, fmt.Errorf("falha ao derivar chave: %w", err)
	}
	childKey, err = childKey.Child(0)
	if err != nil {
		return nil, fmt.Errorf("falha ao derivar chave: %w", err)
	}
	childKey, err = childKey.Child(0)
	if err != nil {
		return nil, fmt.Errorf("falha ao derivar chave: %w", err)
	}

	// Serializar a chave privada
	privKeyBytes, err := childKey.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter chave privada: %w", err)
	}

	// Converter para ECDSA
	privateKey, err := crypto.ToECDSA(privKeyBytes.Serialize())
	if err != nil {
		return nil, fmt.Errorf("falha ao converter para chave ECDSA: %w", err)
	}

	return privateKey, nil
}

func createSignedMessage(signature string, typedData EspressoData) (string, error) {
	// Serializa o TypedData para JSON
	typedDataJSON, err := json.Marshal(typedData)
	if err != nil {
		return "", err
	}

	// Codifica o JSON do TypedData em base64
	typedDataBase64 := base64.StdEncoding.EncodeToString(typedDataJSON)

	// Cria o objeto final com assinatura e dados codificados
	signedMessage := SigAndData{
		Signature: signature,
		TypedData: typedDataBase64,
	}

	// Serializa o objeto final para JSON
	signedMessageJSON, err := json.Marshal(signedMessage)
	if err != nil {
		return "", err
	}

	return string(signedMessageJSON), nil
}
