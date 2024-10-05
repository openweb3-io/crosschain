package client

import (
	"strings"

	"github.com/openweb3-io/crosschain/blockchain/btc/address"
	"github.com/openweb3-io/crosschain/blockchain/btc/client/blockbook"
	"github.com/openweb3-io/crosschain/blockchain/btc/client/blockchair"
	"github.com/openweb3-io/crosschain/blockchain/btc/client/native"
	"github.com/openweb3-io/crosschain/client"
	xc "github.com/openweb3-io/crosschain/types"
)

type BitcoinClient string

var Native BitcoinClient = "native"
var Blockchair BitcoinClient = "blockchair"
var Blockbook BitcoinClient = "blockbook"

type BtcClient interface {
	client.IClient
	address.WithAddressDecoder
}

func NewClient(cfg *xc.ChainConfig) (BtcClient, error) {
	cli, err := NewBitcoinClient(cfg)
	if err != nil {
		return cli, err
	}
	return cli.WithAddressDecoder(&address.BtcAddressDecoder{}).(BtcClient), nil
}
func NewBitcoinClient(cfg *xc.ChainConfig) (BtcClient, error) {
	if strings.Contains(cfg.URL, "api.blockchair.com") {
		return blockchair.NewBlockchairClient(cfg)
	}

	switch BitcoinClient(cfg.Provider) {
	case Native:
		return native.NewNativeClient(cfg)
	case Blockchair:
		return blockchair.NewBlockchairClient(cfg)
	case Blockbook:
		return blockbook.NewClient(cfg)
	default:
		return blockbook.NewClient(cfg)
	}
}
