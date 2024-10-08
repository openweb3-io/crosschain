package client

import (
	"github.com/openweb3-io/crosschain/blockchain/ton/client/liteserver"
	"github.com/openweb3-io/crosschain/blockchain/ton/client/tonapi"
	"github.com/openweb3-io/crosschain/client"
	xc "github.com/openweb3-io/crosschain/types"
)

type TonApiProvider string

var TonApi TonApiProvider = "tonapi"
var LiteServer TonApiProvider = "liteserver"

type TonClient interface {
	client.IClient
}

func NewClient(cfg *xc.ChainConfig) (TonClient, error) {
	switch TonApiProvider(cfg.Provider) {
	case TonApi:
		return tonapi.NewClient(cfg)
	case LiteServer:
		return liteserver.NewClient(cfg)
	default:
		return tonapi.NewClient(cfg)
	}
}
