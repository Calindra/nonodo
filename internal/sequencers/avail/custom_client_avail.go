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
	ctxApp context.Context
	url    string
}

// URL returns the URL the client connects to
func (c CustomClient) URL() string {
	return c.url
}

func (c CustomClient) Call(result interface{}, method string, args ...interface{}) error {
	return c.Client.CallContext(c.ctxApp, result, method, args...)
}

// Connect connects to the provided url
func Connect(ctx context.Context, url string) (*CustomClient, error) {
	slog.Info("avail: connecting to", "url", url)

	ctxDial, cancel := context.WithTimeout(ctx, config.Default().DialTimeout)
	defer cancel()

	c, err := gethrpc.DialContext(ctxDial, url)
	if err != nil {
		return nil, err
	}
	cc := CustomClient{*c, ctx, url}
	return &cc, nil
}

// Same as gsrpc.NewSubstrateAPI with context of the application
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
