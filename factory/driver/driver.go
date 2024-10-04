package driver

import (
	"fmt"

	"github.com/openweb3-io/crosschain"
	"github.com/openweb3-io/crosschain/blockchain/ton"
)

type ClientCreator func(cfg string) (crosschain.IClient, error)

var (
	creatorMap = make(map[string]ClientCreator)
)

func RegisterClient(cfg string, creator ClientCreator) {
	creatorMap[cfg] = creator
}

func init() {
	RegisterClient("ton", func(cfg string) (crosschain.IClient, error) {
		return ton.NewClient(cfg)
	})
}

func NewClient(cfg string) (crosschain.IClient, error) {
	creator, ok := creatorMap[cfg]
	if !ok {
		return nil, fmt.Errorf("creator %s not found", cfg)
	}

	return creator(cfg)
}
