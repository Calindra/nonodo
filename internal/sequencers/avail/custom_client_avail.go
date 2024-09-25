package avail

import (
	"context"
	"log/slog"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/config"
	gethrpc "github.com/centrifuge/go-substrate-rpc-client/v4/gethrpc"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc"
)

type CustomClient struct {
	gethrpc.Client
	url string
}

// URL returns the URL the client connects to
func (c CustomClient) URL() string {
	return c.url
}

func (c CustomClient) Close() {
	c.Client.Close()
}

// Connect connects to the provided url
func Connect(ctx context.Context, url string) (*CustomClient, error) {
	slog.Info("avail: connecting to", "url", url)

	ctx, cancel := context.WithTimeout(ctx, config.Default().DialTimeout)
	defer cancel()

	c, err := gethrpc.DialContext(ctx, url)
	if err != nil {
		return nil, err
	}
	cc := CustomClient{*c, url}
	return &cc, nil
}

func NewSubstrateAPICtx(ctx context.Context, url string) (*gsrpc.SubstrateAPI, error) {
	cl, err := Connect(ctx, url)
	if err != nil {
		return nil, err
	}

	newRPC, err := rpc.NewRPC(cl)
	if err != nil {
		return nil, err
	}

	return &gsrpc.SubstrateAPI{
		RPC:    newRPC,
		Client: cl,
	}, nil
}
