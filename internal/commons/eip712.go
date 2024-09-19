package commons

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/tyler-smith/go-bip39"
)

type SigAndData struct {
	Signature string `json:"signature"`
	TypedData string `json:"typedData"`
}

const (
	HARDHAT         = 31337
	PURPOSE_INDEX   = 44
	COIN_TYPE_INDEX = 60
)

// Implement the hashing function based on EIP-712 requirements
func HashEIP712Message(domain apitypes.TypedDataDomain, data apitypes.TypedData) ([]byte, error) {
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

func GetPrivateKeyFromMnemonic(mnemonic string) (*ecdsa.PrivateKey, error) {
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

func Main() []byte {
	espressoMessage := apitypes.TypedDataMessage{}
	espressoMessage["nonce"] = "1"
	espressoMessage["payload"] = "0xdeadbeef"

	chainId := math.NewHexOrDecimal256(HARDHAT)
	domain := apitypes.TypedDataDomain{
		Name:              "EspressoM",
		Version:           "1",
		ChainId:           chainId,
		VerifyingContract: "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
	}

	types := apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"}, // chainId should be uint256, not uint32
			{Name: "verifyingContract", Type: "address"},
		},
		"EspressoMessage": {
			{Name: "nonce", Type: "uint64"},
			{Name: "payload", Type: "string"},
		},
	}

	// Build Message
	data := apitypes.TypedData{
		Message:     espressoMessage,
		Domain:      domain,
		PrimaryType: "EspressoMessage",
		Types:       types,
	}

	// Hash the message
	messageHash, err := HashEIP712Message(domain, data)
	if err != nil {
		log.Fatal("Error hashing message:", err)
	}

	mnemonic := "test test test test test test test test test test test junk"
	// Private key for signing (this is just a sample, replace with actual private key)
	privateKey, err := GetPrivateKeyFromMnemonic(mnemonic)
	if err != nil {
		log.Fatalf("Error deriving private key: %v", err)
	}

	// Sign the message
	signature, err := SignMessage(messageHash, privateKey)
	if err != nil {
		log.Fatal("Error signing message:", err)
	}

	// Output the signature
	fmt.Printf("Signature: %x\n", signature)

	sigPubkey, err := crypto.Ecrecover(messageHash, signature)
	if err != nil {
		log.Fatal("Error signing message:", err)
	}

	pubkey, err := crypto.UnmarshalPubkey(sigPubkey)
	if err != nil {
		log.Fatal("Error signing message:", err)
	}
	address1 := crypto.PubkeyToAddress(*pubkey)
	fmt.Printf("SigPubkey: %s\n", common.Bytes2Hex(sigPubkey))
	fmt.Printf("Pubkey: %s\n", address1.Hex())

	typedDataJSON, err := json.Marshal(data)
	if err != nil {
		log.Fatal("Error signing message:", err)
	}
	typedDataBase64 := base64.StdEncoding.EncodeToString(typedDataJSON)

	signature[64] += 27
	sigAndData := SigAndData{
		Signature: "0x" + common.Bytes2Hex(signature),
		TypedData: typedDataBase64,
	}
	// fmt.Printf("TypedData %s\n", sigAndData.TypedData)
	jsonPayload, err := json.Marshal(sigAndData)
	if err != nil {
		log.Fatal("Error json.Marshal message:", err)
	}
	address, theData, err := ExtractSigAndData(string(jsonPayload))
	if err != nil {
		log.Fatal("Error ExtractSigAndData message:", err)
	}
	fmt.Println("msgSender", address)
	fmt.Println("The data: ", theData.Message)
	return signature
}

func ExtractSigAndData(raw string) (common.Address, apitypes.TypedData, error) {
	var sigAndData SigAndData
	if err := json.Unmarshal([]byte(raw), &sigAndData); err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, fmt.Errorf("unmarshal sigAndData: %w", err)
	}

	signature, err := hexutil.Decode(sigAndData.Signature)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, fmt.Errorf("decode signature: %w", err)
	}

	typedDataBytes, err := base64.StdEncoding.DecodeString(sigAndData.TypedData)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, fmt.Errorf("decode typed data: %w", err)
	}

	typedData := apitypes.TypedData{}
	if err := json.Unmarshal(typedDataBytes, &typedData); err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, fmt.Errorf("unmarshal typed data: %w", err)
	}

	dataHash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, fmt.Errorf("typed data hash: %w", err)
	}

	// update the recovery id
	// https://github.com/ethereum/go-ethereum/blob/55599ee95d4151a2502465e0afc7c47bd1acba77/internal/ethapi/api.go#L442
	signature[64] -= 27

	// get the pubkey used to sign this signature
	sigPubkey, err := crypto.Ecrecover(dataHash, signature)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, fmt.Errorf("ecrecover: %w", err)
	}
	fmt.Printf("SigPubkey: %s\n", common.Bytes2Hex(sigPubkey))
	pubkey, err := crypto.UnmarshalPubkey(sigPubkey)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, fmt.Errorf("unmarshal: %w", err)
	}
	address := crypto.PubkeyToAddress(*pubkey)

	return address, typedData, nil
}
