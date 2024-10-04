package factory

import (
	xc_client "github.com/openweb3-io/crosschain/client"
	"github.com/openweb3-io/crosschain/factory/blockchains"
	"github.com/openweb3-io/crosschain/types"
)

type IFactory interface {
	NewClient(cfg types.IAsset) (xc_client.IClient, error)
}

type Factory struct {
}

var _ IFactory = &Factory{}

func NewDefaultFactory() *Factory {
	return &Factory{}
}

func (f *Factory) NewClient(cfg types.IAsset) (xc_client.IClient, error) {
	return blockchains.NewClient(cfg)
}
