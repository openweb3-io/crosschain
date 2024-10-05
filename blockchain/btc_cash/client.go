package bitcoin_cash

import (
	"github.com/openweb3-io/crosschain/blockchain/btc/client"
	xc_client "github.com/openweb3-io/crosschain/client"
	xc "github.com/openweb3-io/crosschain/types"
)

func NewClient(cfg *xc.ChainConfig) (xc_client.IClient, error) {
	cli, err := client.NewBitcoinClient(cfg)
	if err != nil {
		return cli, err
	}
	return cli.WithAddressDecoder(&BchAddressDecoder{}).(client.BtcClient), nil
}
