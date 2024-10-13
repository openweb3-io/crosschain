package client

import (
	grpc_client "github.com/openweb3-io/crosschain/blockchain/tron/client/grpc"
	http_client "github.com/openweb3-io/crosschain/blockchain/tron/client/http"
	"github.com/openweb3-io/crosschain/client"
	xc "github.com/openweb3-io/crosschain/types"
)

type Provider string

var Rest Provider = "rest"
var JsonRpc Provider = "jsonrpc"
var Grpc Provider = "grpc"

type TronClient interface {
	client.IClient
}

func NewClient(cfg *xc.ChainConfig) (TronClient, error) {
	switch Provider(cfg.Client.Provider) {
	case Rest:
		return http_client.NewClient(cfg)
	case Grpc:
		return grpc_client.NewClient(cfg)
	default:
		return http_client.NewClient(cfg)
	}
}
