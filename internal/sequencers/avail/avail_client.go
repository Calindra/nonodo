package avail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/sequencers/paiodecoder"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	DEFAULT_EVM_MNEMONIC    = "test test test test test test test test test test test junk"
	DEFAULT_USER_ADDRESS    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	DEFAULT_GRAPHQL_URL     = "http://localhost:8080"
	DEFAULT_AVAIL_RPC_URL   = "wss://turing-rpc.avail.so/ws"
	DEFAULT_APP_ID          = 91
	DEFAULT_CHAINID_HARDHAT = 31337
)

type AvailClient struct {
	apiURL     string
	GraphQLUrl string
	chainId    int
	appId      int
}

func NewAvailClient(graphQLUrl string, chainId int, appId int) (*AvailClient, error) {
	apiURL, hasEnv := os.LookupEnv("AVAIL_RPC_URL")
	if !hasEnv {
		apiURL = DEFAULT_AVAIL_RPC_URL
	}
	if graphQLUrl == "" {
		graphQLUrl = DEFAULT_GRAPHQL_URL
	}
	client := AvailClient{apiURL, graphQLUrl, chainId, appId}
	return &client, nil
}

func (av *AvailClient) Submit712(ctx context.Context, payload string, dappAddress string, maxGasPrice uint64) (*types.Hash, error) {
	nonce, err := fetchNonce(DEFAULT_USER_ADDRESS, av.GraphQLUrl)

	if err != nil {
		log.Fatalf("Avail error getting nonce: %v", err)
	}
	n, err := strconv.Atoi(nonce)
	if err != nil {
		return nil, err
	}
	typedData := paiodecoder.CreateTypedData(
		common.HexToAddress(dappAddress),
		uint64(n),
		big.NewInt(int64(maxGasPrice)), []byte(payload), big.NewInt(int64(av.chainId)),
	)

	// Hash the message
	messageHash, err := commons.HashEIP712Message(typedData)
	if err != nil {
		log.Fatal("Error hashing message:", err)
	}

	// Private key for signing (this is just a sample, replace with actual private key)
	privateKey, err := commons.GetPrivateKeyFromMnemonic(DEFAULT_EVM_MNEMONIC)
	if err != nil {
		log.Fatalf("Error deriving private key: %v", err)
	}

	// Sign the message
	signature, err := commons.SignMessage(messageHash, privateKey)
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

	typedDataJSON, err := json.Marshal(typedData)
	if err != nil {
		log.Fatal("Error signing message:", err)
	}
	typedDataBase64 := base64.StdEncoding.EncodeToString(typedDataJSON)

	signature[64] += 27
	sigAndData := commons.SigAndData{
		Signature: "0x" + common.Bytes2Hex(signature),
		TypedData: typedDataBase64,
	}
	jsonPayload, err := json.Marshal(sigAndData)
	if err != nil {
		log.Fatal("Error json.Marshal message:", err)
	}
	return av.DefaultSubmit(ctx, string(jsonPayload))
}

func (av *AvailClient) DefaultSubmit(ctx context.Context, data string) (*types.Hash, error) {
	apiURL := os.Getenv("AVAIL_RPC_URL")
	if apiURL == "" {
		apiURL = DEFAULT_AVAIL_RPC_URL
	}

	seed := os.Getenv("AVAIL_MNEMONIC")

	if seed == "" {
		seed = DEFAULT_EVM_MNEMONIC
	}

	return av.SubmitData(ctx, data, apiURL, seed, av.appId)
}

// SubmitData creates a transaction and makes a Avail data submission
func (av *AvailClient) SubmitData(ctx context.Context, data string, ApiURL string, Seed string, AppID int) (*types.Hash, error) {
	fmt.Printf("AppID=%d\n", AppID)

	api, err := NewSubstrateAPICtx(ctx, av.apiURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create api:%w", err)
	}
	defer api.Client.Close()

	if AppID == 0 {
		return nil, fmt.Errorf("wrong app id")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, fmt.Errorf("cannot get metadata:%w", err)
	}

	// Set data and appID according to need
	appID := 0

	// if app id is greater than 0 then it must be created before submitting data
	if AppID != 0 {
		appID = AppID
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return nil, fmt.Errorf("cannot get block hash:%w", err)
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return nil, fmt.Errorf("cannot get runtime version:%w", err)
	}

	networkNumber := 42
	keyringPair, err := signature.KeyringPairFromSecret(Seed, uint16(networkNumber))
	if err != nil {
		return nil, fmt.Errorf("cannot create KeyPair:%w", err)
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create storage key:%w", err)
	}

	var accountInfo types.AccountInfo
	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		return nil, fmt.Errorf("cannot get latest storage:%w", err)
	}
	nonce := uint32(accountInfo.Nonce)
	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		AppID:              types.NewUCompactFromUInt(uint64(AppID)),
		TransactionVersion: rv.TransactionVersion,
	}

	c, err := types.NewCall(meta, "DataAvailability.submit_data", types.NewBytes([]byte(data)))
	if err != nil {
		return nil, fmt.Errorf("cannot create new call:%w", err)
	}

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	// Sign the transaction using Alice's default account
	err = ext.Sign(keyringPair, o)
	if err != nil {
		return nil, fmt.Errorf("cannot sign:%w", err)
	}

	// Send the extrinsic
	hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	if err != nil {
		return nil, fmt.Errorf("cannot submit extrinsic:%w", err)
	}
	fmt.Printf("Data submitted: %v against appID %v  sent with hash %#x\n", data, appID, hash)

	return &hash, nil
}

// GraphQL
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

func fetchNonce(sender string, graphqlURL string) (string, error) {
	query := fmt.Sprintf(`
		{
			inputs(where: {msgSender: "%s" type: "Avail"}) {
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

func ReadTimestampFromBlock(block *types.SignedBlock) (uint64, error) {
	timestampSectionIndex := uint8(TIMESTAMP_SECTION_INDEX)
	timestampMethodIndex := uint8(0)
	coreAppID := int64(0)
	for _, ext := range block.Block.Extrinsics {
		appID := ext.Signature.AppID.Int64()

		mi := ext.Method.CallIndex.MethodIndex
		si := ext.Method.CallIndex.SectionIndex

		if appID == coreAppID && si == uint8(timestampSectionIndex) && mi == uint8(timestampMethodIndex) {
			timestamp := DecodeTimestamp(common.Bytes2Hex(ext.Method.Args))
			return timestamp, nil
		}
	}
	return 0, fmt.Errorf("block %d without timestamp", block.Block.Header.Number)
}

func DecodeTimestamp(hexStr string) uint64 {
	decoded, err := hex.DecodeString(padHexStringRight(hexStr))
	if err != nil {
		fmt.Println("Error decoding hex:", err)
		return 0
	}
	return decodeCompactU64(decoded)
}

// nolint
func decodeCompactU64(data []byte) uint64 {
	firstByte := data[0]
	if firstByte&0b11 == 0b00 { // Single byte (6-bit value)
		return uint64(firstByte >> 2)
	} else if firstByte&0b11 == 0b01 { // Two bytes (14-bit value)
		return uint64(firstByte>>2) | uint64(data[1])<<6
	} else if firstByte&0b11 == 0b10 { // Four bytes (30-bit value)
		return uint64(firstByte>>2) | uint64(data[1])<<6 | uint64(data[2])<<14 | uint64(data[3])<<22
	} else { // Eight bytes (64-bit value)
		return uint64(data[1]) | uint64(data[2])<<8 | uint64(data[3])<<16 | uint64(data[4])<<24 |
			uint64(data[5])<<32 | uint64(data[6])<<40 | uint64(data[7])<<48
	}
}

// nolint
func encodeCompactU64(value uint64) []byte {
	var result []byte

	if value < (1 << 6) { // Single byte (6-bit value)
		result = []byte{byte(value<<2) | 0b00}
	} else if value < (1 << 14) { // Two bytes (14-bit value)
		result = []byte{
			byte((value&0x3F)<<2) | 0b01,
			byte(value >> 6),
		}
	} else if value < (1 << 30) { // Four bytes (30-bit value)
		result = []byte{
			byte((value&0x3F)<<2) | 0b10,
			byte(value >> 6),
			byte(value >> 14),
			byte(value >> 22),
		}
	} else { // Eight bytes (64-bit value)
		result = []byte{
			0b11, // First byte indicates 8-byte encoding
			byte(value),
			byte(value >> 8),
			byte(value >> 16),
			byte(value >> 24),
			byte(value >> 32),
			byte(value >> 40),
			byte(value >> 48),
			byte(value >> 56),
		}
	}

	return result
}

func padHexStringRight(hexStr string) string {
	if len(hexStr) > 1 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}

	// Right pad with zeros to ensure it's 16 characters long (8 bytes)
	for len(hexStr) < 16 {
		hexStr += "0"
	}

	return hexStr
}
