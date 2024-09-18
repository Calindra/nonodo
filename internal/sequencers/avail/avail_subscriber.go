package avail

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/calindra/nonodo/internal/sequencers/avail/config"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
)

// The following example shows how to connect to a node and listen for a new blocks
func main() {
	var configJSON string
	var config config.Config
	flag.StringVar(&configJSON, "config", "", "config json file")
	flag.Parse()

	if configJSON == "" {
		log.Println("No config file provided. Exiting...")
		os.Exit(0)
	}

	err := config.GetConfig(configJSON)
	if err != nil {
		panic(fmt.Sprintf("cannot get config:%v", err))
	}

	api, err := gsrpc.NewSubstrateAPI(config.ApiURL)
	if err != nil {
		panic(fmt.Sprintf("cannot create api client:%v", err))
	}

	subscription, err := api.RPC.Chain.SubscribeNewHeads()
	if err != nil {
		panic(fmt.Sprintf("cannot subscribe:%v", err))
	}

	// number of blocks to wait
	waitForBlocks := 1
	count := 0
	for i := range subscription.Chan() {
		count++
		fmt.Printf("Chain is at block: #%v\n", i.Number)
		blockHash, err := api.RPC.Chain.GetBlockHash(746430)
		if err != nil {
			panic(err)
		}
		block, err := api.RPC.Chain.GetBlock(blockHash)
		if err != nil {
			panic(err)
		}
		for index, ext := range block.Block.Extrinsics {
			fmt.Printf("ext[%d].AppID=%d\n", index, ext.Signature.AppID.Int64())
			json, err := ext.MarshalJSON()
			if err != nil {
				panic(err)
			}
			strJson := string(json)
			fmt.Println("json=" + strJson)
			args := string(ext.Method.Args)
			fmt.Println("args=" + args)
		}
		if count == waitForBlocks {
			break
		}
	}

	subscription.Unsubscribe()
}
