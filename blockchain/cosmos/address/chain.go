package address

import (
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	injethsecp256k1 "github.com/openweb3-io/crosschain/blockchain/cosmos/types/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"
	"github.com/openweb3-io/crosschain/blockchain/cosmos/types/evmos/ethermint/crypto/ethsecp256k1"
	xc "github.com/openweb3-io/crosschain/types"
)

func IsEVMOS(asset *xc.ChainConfig) bool {
	return xc.Blockchain(asset.Blockchain) == xc.BlockchainCosmosEvmos
}

func GetPublicKey(asset *xc.ChainConfig, publicKeyBytes []byte) cryptotypes.PubKey {
	if asset.Chain == xc.INJ {
		// injective has their own ethsecp256k1 type..
		return &injethsecp256k1.PubKey{Key: publicKeyBytes}
	}
	if IsEVMOS(asset) {
		return &ethsecp256k1.PubKey{Key: publicKeyBytes}
	}
	return &secp256k1.PubKey{Key: publicKeyBytes}
}
