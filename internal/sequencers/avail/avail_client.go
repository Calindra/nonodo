package avail

import (
	"fmt"
	"os"
	"strconv"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func Submit712(data string) {

}

func DefaultSubmit(data string) error {
	apiURL := os.Getenv("AVAIL_RPC_URL")
	if apiURL == "" {
		apiURL = "wss://turing-rpc.avail.so/ws"
	}
	seed := os.Getenv("AVAIL_MNEMONIC")
	if seed == "" {
		return fmt.Errorf("missing AVAIL_MNEMONIC environment variable")
	}
	strAppID := os.Getenv("AVAIL_APP_ID")
	if strAppID == "" {
		strAppID = "91"
	}
	appID, err := strconv.Atoi(strAppID)
	if err != nil {
		return fmt.Errorf("AVAIL_APP_ID is not a number: %s", strAppID)
	}
	return SubmitData(data, apiURL, seed, appID)
}

// SubmitData creates a transaction and makes a Avail data submission
func SubmitData(data string, ApiURL string, Seed string, AppID int) error {
	fmt.Printf("AppID=%d\n", AppID)
	if AppID == 0 {
		return nil
	}
	api, err := gsrpc.NewSubstrateAPI(ApiURL)
	if err != nil {
		return fmt.Errorf("cannot create api:%w", err)
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return fmt.Errorf("cannot get metadata:%w", err)
	}

	// Set data and appID according to need
	appID := 0

	// if app id is greater than 0 then it must be created before submitting data
	if AppID != 0 {
		appID = AppID
	}

	c, err := types.NewCall(meta, "DataAvailability.submit_data", types.NewBytes([]byte(data)))
	if err != nil {
		return fmt.Errorf("cannot create new call:%w", err)
	}

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return fmt.Errorf("cannot get block hash:%w", err)
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return fmt.Errorf("cannot get runtime version:%w", err)
	}

	keyringPair, err := signature.KeyringPairFromSecret(Seed, 42)
	if err != nil {
		return fmt.Errorf("cannot create KeyPair:%w", err)
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	if err != nil {
		return fmt.Errorf("cannot create storage key:%w", err)
	}

	var accountInfo types.AccountInfo
	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		return fmt.Errorf("cannot get latest storage:%w", err)
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
	// Sign the transaction using Alice's default account
	err = ext.Sign(keyringPair, o)
	if err != nil {
		return fmt.Errorf("cannot sign:%w", err)
	}

	// Send the extrinsic
	hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	if err != nil {
		return fmt.Errorf("cannot submit extrinsic:%w", err)
	}
	fmt.Printf("Data submitted: %v against appID %v  sent with hash %#x\n", data, appID, hash)

	return nil
}
