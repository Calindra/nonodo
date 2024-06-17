package dataavailability

// TIA = 714
import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math/big"
	stdhttp "net/http"
	"os"
	"strings"

	client "github.com/celestiaorg/celestia-openrpc"
	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/celestiaorg/celestia-openrpc/types/share"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/labstack/echo/v4"
	"github.com/tendermint/tendermint/rpc/client/http"

	"github.com/calindra/nonodo/internal/contracts"
)

var CELESTIA_RELAY_ADDRESS common.Address = common.HexToAddress("0x9246F2Ca979Ef55FcacB5C4D3F46D36399da760e")

// var CELESTIA_RELAY_ADDRESS common.Address = common.HexToAddress("0x096a7847B754647e06A887BeF0192689148A0C33")

type GioReqParams struct {
	Namespace []byte
	Height    *big.Int
	Start     *big.Int
	End       *big.Int
}

// SubmitBlob submits a blob containing "Hello, World!" to the 0xDEADBEEF namespace. It uses the default signer on the running node.
func SubmitBlob(ctx context.Context, url string, token string, namespaceHex string, rawData []byte) (height uint64, start uint64, end uint64, err error) {
	client, err := client.NewClient(ctx, url, token)
	if err != nil {
		return 0, 0, 0, err
	}

	// let's post to 0xDEADBEEF namespace
	nms := common.Hex2Bytes(namespaceHex)
	namespace, err := share.NewBlobNamespaceV0(nms)
	if err != nil {
		return 0, 0, 0, err
	}

	// create a blob
	helloWorldBlob, err := blob.NewBlobV0(namespace, rawData)
	if err != nil {
		return 0, 0, 0, err
	}

	base64Str := base64.StdEncoding.EncodeToString(helloWorldBlob.Commitment)
	slog.Debug("Blob Commitment", "Commitment", common.Bytes2Hex(helloWorldBlob.Commitment), "base64Str", base64Str)

	// if url != "" {
	// 	return nil
	// }
	// client.State.SubmitPayForBlob(ctx, math.Int, 1, []*blob.Blob{helloWorldBlob})
	// submit the blob to the network
	height, err = client.Blob.Submit(ctx, []*blob.Blob{helloWorldBlob}, blob.DefaultGasPrice())
	if err != nil {
		slog.Error("Submit", "error", err)
		return 0, 0, 0, err
	}

	bProof, err := client.Blob.GetProof(ctx, height, namespace, helloWorldBlob.Commitment)
	if err != nil {
		return 0, 0, 0, err
	}

	slog.Debug("Blob was included at",
		"height", height,
		"start", (*bProof)[0].Start(),
		"end", (*bProof)[0].End(),
	)

	// fetch the blob back from the network
	// retrievedBlobs, err := client.Blob.GetAll(ctx, height, []share.Namespace{namespace})
	retrievedBlob, err := client.Blob.Get(ctx, height, namespace, helloWorldBlob.Commitment)
	if err != nil {
		return 0, 0, 0, err
	}

	// slog.Debug("Blobs are equal?", "equal", bytes.Equal(helloWorldBlob.Commitment, retrievedBlobs[0].Commitment))

	slog.Debug("Blobs are equal?",
		"equal", bytes.Equal(helloWorldBlob.Commitment, retrievedBlob.Commitment),
		"commitment", helloWorldBlob.Commitment,
		"content", string(retrievedBlob.Data),
	)

	proof, err := client.Blob.GetProof(ctx, height, namespace, helloWorldBlob.Commitment)
	if err != nil {
		return 0, 0, 0, err
	}

	json, err := retrievedBlob.MarshalJSON()
	if err != nil {
		return 0, 0, 0, err
	}

	slog.Debug("Proof",
		"start", (*proof)[0].Start(),
		"end", (*proof)[0].End(),
		"index", string(json),
	)
	return height, uint64((*proof)[0].Start()), uint64((*proof)[0].End()), nil
}

func getABI() (*abi.ABI, error) {
	jsonABI := `[
		{
			"constant": true,
			"inputs": [
				{
					"name": "namespace",
					"type": "bytes29"
				},
				{
					"name": "height",
					"type": "uint256"
				},
				{
					"name": "start",
					"type": "uint256"
				},
				{
					"name": "end",
					"type": "uint256"
				}
			],
			"name": "CelestiaRequest",
			"outputs": [
				{
					"name": "",
					"type": "bytes"
				}
			],
			"payable": false,
			"stateMutability": "pure",
			"type": "function"
		}
	]`
	parsed, err := abi.JSON(strings.NewReader(jsonABI))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseParams(id string) (*GioReqParams, error) {
	abiParsed, err := getABI()
	if err != nil {
		return nil, err
	}
	values, err := abiParsed.Methods["CelestiaRequest"].Inputs.UnpackValues(common.Hex2Bytes(id[2:]))
	if err != nil {
		slog.Error("Error unpacking blob.", "err", err)
		return nil, err
	}
	namespace := values[0].([29]uint8)
	height := values[1].(*big.Int)
	start := values[2].(*big.Int)
	res := GioReqParams{
		Namespace: namespace[:],
		Height:    height,
		Start:     start,
	}
	return &res, nil
}

func GetBlob(ctx context.Context, id string, url string, token string) ([]byte, error) {
	gioReqParams, err := parseParams(id)
	if err != nil {
		return nil, err
	}
	namespaceV0, err := share.NewBlobNamespaceV0(gioReqParams.Namespace[22:])
	if err != nil {
		return nil, err
	}
	slog.Debug("CelestiaRequest",
		"namespaceV0", common.Bytes2Hex(namespaceV0[:]),
		"height", gioReqParams.Height,
		"start", gioReqParams.Start,
	)
	client, err := client.NewClient(ctx, url, token)
	if err != nil {
		return nil, err
	}
	// namespaceV0, err = share.NewBlobNamespaceV0([]byte{0xDE, 0xAD, 0xBE, 0xEF})

	celestiaHeight := gioReqParams.Height.Uint64()
	celestiaNamespace := []share.Namespace{namespaceV0}
	retrievedBlobs, err := client.Blob.GetAll(ctx, celestiaHeight, celestiaNamespace)
	if err != nil {
		return nil, err
	}
	// find the blob with the start = json.index
	json, err := retrievedBlobs[0].MarshalJSON()
	if err != nil {
		return nil, err
	}
	slog.Debug("Blobs",
		"len", len(retrievedBlobs),
		"data", string(retrievedBlobs[0].Blob.Data),
		"data", string(json),
	)

	return retrievedBlobs[0].Blob.Data, nil
}

func connections() (eth *ethclient.Client, trpc *http.HTTP, err error) {
	ethEndpoint := "https://arbitrum-sepolia-rpc.publicnode.com"
	trpcEndpoint := "https://celestia-mocha-rpc.publicnode.com:443"

	eth, err = ethclient.Dial(ethEndpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to the Ethereum node: %w", err)
	}
	trpc, err = http.New(trpcEndpoint, "/websocket")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to the Tendermint RPC: %w", err)
	}

	return eth, trpc, nil
}

// GetShareProof returns the share proof for the given share pointer.
// Ready to be used with the DAVerifier library.
// RE: https://docs.celestia.org/developers/blobstream-proof-queries#example-rollup-that-uses-the-daverifier
func GetShareProof(ctx context.Context, height uint64, start uint64, end uint64) (shareProofFinal *contracts.SharesProof, blockDataRoot [32]byte, err error) {
	var maxHeight uint64 = 10_000_000

	eth, trpc, err := connections()

	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("failed to connect to the Ethereum node or trpc: %w", err)
	}

	defer eth.Close()

	// 1. Get the data commitment
	dataCommitment, err := GetDataCommitment(eth, int64(height), maxHeight)

	if err != nil {
		return nil, [32]byte{}, err
	}

	h := int64(height)

	// 2. Get the block
	blockRes, err := trpc.Block(ctx, &h)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("failed to get block: %w", err)
	}

	// 3. get data root inclusion commitment
	dcProof, err := trpc.DataRootInclusionProof(ctx, height, dataCommitment.StartBlock, dataCommitment.EndBlock)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("failed to get data root inclusion proof: %w", err)
	}

	// 4. get share proof
	shareProof, err := trpc.ProveShares(ctx, height, start, end)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("failed to get share proof: %w", err)
	}

	nonce := dataCommitment.ProofNonce.Uint64()

	blockDataRoot = [32]byte(blockRes.Block.DataHash)

	slog.Info("ShareProof", "Length", len(shareProof.ShareProofs), "Start", shareProof.ShareProofs[0].Start, "End", shareProof.ShareProofs[0].End)

	return &contracts.SharesProof{
		Data:             shareProof.Data,
		ShareProofs:      toNamespaceMerkleMultiProofs(shareProof.ShareProofs),
		Namespace:        *namespace(shareProof.NamespaceID, uint8(shareProof.NamespaceVersion)),
		RowRoots:         toRowRoots(shareProof.RowProof.RowRoots),
		RowProofs:        toRowProofs(shareProof.RowProof.Proofs),
		AttestationProof: toAttestationProof(nonce, height, blockDataRoot, dcProof.Proof),
	}, blockDataRoot, nil
}

func CallCelestiaRelay(ctx context.Context, height uint64, start uint64, end uint64, dappAddress common.Address, execLayerData []byte, ethEndpointRPC string) error {
	var chainId int64 = 31337
	pk_celestia := os.Getenv("PK_CELESTIA")

	if pk_celestia == "" {
		return fmt.Errorf("missing Celestia private key")
	}

	// Connect to an Ethereum node
	eth, err := ethclient.Dial(ethEndpointRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to the Ethereum node: %w", err)
	}
	defer eth.Close()

	proofs, root, err := GetShareProof(ctx, height, start, end)

	if err != nil {
		return err
	}

	// Load your private key
	privateKey, err := crypto.HexToECDSA(pk_celestia)
	if err != nil {
		return fmt.Errorf("failed to create private key: %w", err)
	}

	// Get the public key address
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("failed to create public key: %w", err)
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Get the nonce (number of transactions sent by the sender)
	nonce, err := eth.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	// Set the gas price
	gasPrice, err := eth.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	// Set up the transaction options
	gasLimit := 3000000
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(chainId))
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(gasLimit)
	auth.GasPrice = gasPrice

	// Create a new instance of the contract
	celestiaRelayAddress := CELESTIA_RELAY_ADDRESS
	envAddress := os.Getenv("CELESTIA_RELAY_ADDRESS")
	if envAddress != "" {
		celestiaRelayAddress = common.HexToAddress(envAddress)
	}
	relay, err := contracts.NewCelestiaRelay(celestiaRelayAddress, eth)

	if err != nil {
		return err
	}

	// Call the contract
	slog.Debug("call relay shares", "dappAddress", dappAddress)
	trx, err := relay.RelayShares(auth, dappAddress, *proofs, root, execLayerData)

	if err != nil {
		return err
	}

	slog.Info("Transaction", "trx", trx)

	return nil
}

func FetchFromTendermint(ctx context.Context, id string) (*string, error) {
	gioReqParams, err := parseParams(id)
	if err != nil {
		return nil, err
	}
	trpcEndpoint := "https://celestia-mocha-rpc.publicnode.com:443"
	trpc, err := http.New(trpcEndpoint, "/websocket")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the Tendermint RPC: %w", err)
	}
	blockHeight := gioReqParams.Height.Uint64()
	shareStart := gioReqParams.Start.Uint64()
	shareEnd := shareStart + 1
	slog.Debug("Fetch data",
		"blockHeight", blockHeight,
		"shareStart", shareStart,
		"shareEnd", shareEnd,
	)

	shareProof, err := trpc.ProveShares(ctx, blockHeight, shareStart, shareEnd)
	if err != nil {
		return nil, err
	}
	namespace := shareProof.Data[0][:29]
	dataRawLen := shareProof.Data[0][30:34]
	dataLen := binary.BigEndian.Uint32(dataRawLen)
	data := shareProof.Data[0][34 : 34+dataLen]
	slog.Debug("Result",
		"namespace", common.Bytes2Hex(namespace),
		"dataLen", binary.BigEndian.Uint32(dataRawLen),
		"data", common.Bytes2Hex(data),
		"data", string(data),
	)
	dataAsHex := common.Bytes2Hex(data)
	return &dataAsHex, nil
}

type CelestiaClient struct{}

// Fetch implements Fetch.
func (c *CelestiaClient) Fetch(ctx echo.Context, id string) (*string, *HttpCustomError) {
	var cont context.Context = ctx.Request().Context()
	data, err := FetchFromTendermint(cont, id)
	if err == nil {
		return data, nil
	}

	token := os.Getenv("TIA_AUTH_TOKEN")
	url := os.Getenv("TIA_URL")

	if token == "" || url == "" {
		slog.Error("missing celestia configuration")
		return nil, NewHttpCustomError(stdhttp.StatusInternalServerError, nil)
	}

	blob, err := GetBlob(cont, id, url, token)

	if err != nil {
		msg := err.Error()
		return nil, NewHttpCustomError(stdhttp.StatusBadRequest, &msg)
	}

	result := common.Bytes2Hex(blob)

	return &result, nil
}

func NewCelestiaClient() Fetch {
	return &CelestiaClient{}
}
