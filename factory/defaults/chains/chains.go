package chains

import (
	_ "embed"

	"github.com/openweb3-io/crosschain/factory/defaults/common"
	xc "github.com/openweb3-io/crosschain/types"
)

func init() {
	maincfg := common.Unmarshal(mainnetData)
	testcfg := common.Unmarshal(testnetData)

	Mainnet = maincfg.Chains
	Testnet = testcfg.Chains
	defaultUrl := "https://connector.cordialapis.com"

	for _, chain := range Mainnet {
		if chain.Net == "" {
			chain.Net = string(maincfg.Network)
		}
		if chain.ConfirmationsFinal == 0 {
			chain.ConfirmationsFinal = 6
		}

		// default to using xc client
		chain.Clients = []*xc.ClientConfig{
			{
				Driver: xc.DriverCrosschain,
				URL:    defaultUrl,
				// default is mainnet
				Network: "",
			},
		}
	}
	for _, chain := range Testnet {
		if chain.Net == "" {
			chain.Net = string(testcfg.Network)
		}
		if chain.ConfirmationsFinal == 0 {
			chain.ConfirmationsFinal = 2
		}
		chain.Clients = []*xc.ClientConfig{
			{
				Driver:  xc.DriverCrosschain,
				URL:     defaultUrl,
				Network: xc.NotMainnets,
			},
		}
	}
}

//go:embed mainnet.yaml
var mainnetData string

//go:embed testnet.yaml
var testnetData string

var Mainnet map[string]*xc.ChainConfig
var Testnet map[string]*xc.ChainConfig
