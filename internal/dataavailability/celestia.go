package dataavailability

// TIA = 714
import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	client "github.com/celestiaorg/celestia-openrpc"
	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/celestiaorg/celestia-openrpc/types/share"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/rpc/client/http"
)

func GetProof() error {
	ctx := context.Background()
	trpc, err := http.New("tcp://localhost:26657", "/websocket")
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = trpc.Start()
	if err != nil {
		fmt.Println(err)
		return err
	}
	dcProof, err := trpc.DataRootInclusionProof(ctx, 15, 10, 20)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(dcProof.Proof.String())
	return nil
}

// SubmitBlob submits a blob containing "Hello, World!" to the 0xDEADBEEF namespace. It uses the default signer on the running node.
func SubmitBlob(ctx context.Context, url string, token string) error {
	client, err := client.NewClient(ctx, url, token)
	if err != nil {
		return err
	}

	// let's post to 0xDEADBEEF namespace
	namespace, err := share.NewBlobNamespaceV0([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	if err != nil {
		return err
	}

	// create a blob
	helloWorldBlob, err := blob.NewBlobV0(namespace, []byte("Hello, World! Cartesi Rocks!"))
	if err != nil {
		return err
	}

	base64Str := base64.StdEncoding.EncodeToString(helloWorldBlob.Commitment)
	slog.Debug("Blob Commitment", "Commitment", common.Bytes2Hex(helloWorldBlob.Commitment), "base64Str", base64Str)

	// if url != "" {
	// 	return nil
	// }
	// client.State.SubmitPayForBlob(ctx, math.Int, 1, []*blob.Blob{helloWorldBlob})
	// submit the blob to the network
	height, err := client.Blob.Submit(ctx, []*blob.Blob{helloWorldBlob}, blob.DefaultGasPrice())
	if err != nil {
		slog.Error("Submit", "error", err)
		return err
	}

	bProof, err := client.Blob.GetProof(ctx, height, namespace, helloWorldBlob.Commitment)
	if err != nil {
		return err
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
		return err
	}

	// slog.Debug("Blobs are equal?", "equal", bytes.Equal(helloWorldBlob.Commitment, retrievedBlobs[0].Commitment))

	slog.Debug("Blobs are equal?",
		"equal", bytes.Equal(helloWorldBlob.Commitment, retrievedBlob.Commitment),
		"commitment", helloWorldBlob.Commitment,
		"content", string(retrievedBlob.Data),
	)

	proof, err := client.Blob.GetProof(ctx, height, namespace, helloWorldBlob.Commitment)
	if err != nil {
		return err
	}

	json, err := retrievedBlob.MarshalJSON()
	if err != nil {
		return err
	}

	slog.Debug("Proof",
		"start", (*proof)[0].Start(),
		"end", (*proof)[0].End(),
		"index", string(json),
	)
	return nil
}
