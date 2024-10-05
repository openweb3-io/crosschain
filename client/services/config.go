package services

import (
	xc "github.com/openweb3-io/crosschain/types"
)

type KilnConfig struct {
	BaseUrl  string `mapstructure:"base_url,omitempty" json:"base_url,omitempty" yaml:"base_url,omitempty" toml:"base_url,omitempty"`
	ApiToken string `mapstructure:"api_token,omitempty" json:"api_token,omitempty" yaml:"api_token,omitempty" toml:"api_token,omitempty"`
}
type FigmentConfig struct {
	BaseUrl  string `mapstructure:"base_url,omitempty" json:"base_url,omitempty" yaml:"base_url,omitempty" toml:"base_url,omitempty"`
	ApiToken string `mapstructure:"api_token,omitempty" json:"api_token,omitempty" yaml:"api_token,omitempty" toml:"api_token,omitempty"`
	Network  string `mapstructure:"network,omitempty" json:"network,omitempty" yaml:"network,omitempty" toml:"network,omitempty"`
}
type TwinstakeConfig struct {
	BaseUrl string `mapstructure:"base_url,omitempty" json:"base_url,omitempty" yaml:"base_url,omitempty" toml:"base_url,omitempty"`

	Username string `mapstructure:"username" json:"username" yaml:"username" toml:"username"`
	Password string `mapstructure:"password,omitempty" json:"password,omitempty" yaml:"password,omitempty" toml:"password,omitempty"`
	ClientId string `mapstructure:"client_id" json:"client_id" yaml:"client_id" toml:"client_id"`
	Region   string `mapstructure:"region" json:"region" yaml:"region" toml:"region"`
}

type ServicesConfig struct {
	Kiln      KilnConfig      `mapstructure:"kiln" json:"kiln" yaml:"kiln" toml:"kiln"`
	Twinstake TwinstakeConfig `mapstructure:"twinstake" json:"twinstake" yaml:"twinstake" toml:"twinstake"`
	Figment   FigmentConfig   `mapstructure:"figment" json:"figment" yaml:"figment" toml:"figment"`
}

func (c *ServicesConfig) GetApiSecret(provider xc.StakingProvider) string {
	switch provider {
	case xc.Kiln:
		return c.Kiln.ApiToken
	case xc.Figment:
		return c.Figment.ApiToken
	case xc.Twinstake:
		// TODO, twinstake has a login process
		return ""
	}
	return ""
}

func DefaultConfig(network string) *ServicesConfig {
	cfg := &ServicesConfig{
		Kiln: KilnConfig{
			BaseUrl:  "https://api.kiln.fi",
			ApiToken: "env:KILN_API_TOKEN",
		},
		Twinstake: TwinstakeConfig{
			BaseUrl:  "https://api.twinstake.io",
			Username: "env:TWINSTAKE_USERNAME",
			Password: "env:TWINSTAKE_PASSWORD",
			ClientId: "env:TWINSTAKE_CLIENT_ID",
			Region:   "eu-west-3", // reported default on twinstakes website
		},
		Figment: FigmentConfig{
			BaseUrl:  "https://api.figment.io",
			ApiToken: "env:FIGMENT_API_TOKEN",
			Network:  "mainnet",
		},
	}
	if network == "testnet" {
		cfg.Kiln.BaseUrl = "https://api.testnet.kiln.fi"
		cfg.Twinstake.BaseUrl = "https://testnet.api.twinstake.io"
		cfg.Figment.Network = "holesky"
	}

	return cfg
}
