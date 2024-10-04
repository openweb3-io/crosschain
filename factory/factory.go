package factory

import (
	"github.com/openweb3-io/crosschain"
	"github.com/openweb3-io/crosschain/factory/driver"
)

type IFactory interface {
	NewClient(cfg string) (crosschain.IClient, error)
}

type Factory struct {
}

var _ IFactory = &Factory{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) NewClient(cfg string) (crosschain.IClient, error) {
	return driver.NewClient(cfg)
}
