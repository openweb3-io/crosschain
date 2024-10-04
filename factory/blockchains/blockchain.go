package blockchains

import (
	"fmt"

	"github.com/openweb3-io/crosschain/blockchain/ton"
	xc_client "github.com/openweb3-io/crosschain/client"
	"github.com/openweb3-io/crosschain/types"
)

type ClientCreator func(cfg types.IAsset) (xc_client.IClient, error)

var (
	creatorMap = make(map[types.NativeAsset]ClientCreator)
)

func RegisterClient(cfg types.NativeAsset, creator ClientCreator) {
	creatorMap[cfg] = creator
}

func init() {
	RegisterClient("ton", func(cfg types.IAsset) (xc_client.IClient, error) {
		return ton.NewClient(cfg)
	})
}

func NewClient(cfg types.IAsset) (xc_client.IClient, error) {
	creator, ok := creatorMap[cfg.GetChain().Chain]
	if !ok {
		return nil, fmt.Errorf("creator %s not found", cfg)
	}

	return creator(cfg)
}
