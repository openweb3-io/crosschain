package builder

import (
	xc_types "github.com/openweb3-io/crosschain/types"
	"go.uber.org/zap"
)

// All possible builder arguments go in here, privately available.
// Then the public BuilderArgs can typecast and select which arguments are needed.
type builderOptions struct {
	memo           *string
	timestamp      *int64
	gasFeePriority *xc_types.GasFeePriority
	publicKey      *[]byte

	validator    *string
	stakeOwner   *xc_types.Address
	stakeAccount *string

	asset *xc_types.IAsset
}

// All ArgumentBuilders should provide base arguments for transactions
type TransactionOptions interface {
	GetMemo() (string, bool)
	GetTimestamp() (int64, bool)
	GetPriority() (xc_types.GasFeePriority, bool)
	GetPublicKey() ([]byte, bool)
}

var _ TransactionOptions = &builderOptions{}

func get[T any](arg *T) (T, bool) {
	if arg == nil {
		var zero T
		return zero, false
	}
	return *arg, true
}

// Transaction options
func (opts *builderOptions) GetMemo() (string, bool)     { return get(opts.memo) }
func (opts *builderOptions) GetTimestamp() (int64, bool) { return get(opts.timestamp) }
func (opts *builderOptions) GetPriority() (xc_types.GasFeePriority, bool) {
	return get(opts.gasFeePriority)
}
func (opts *builderOptions) GetPublicKey() ([]byte, bool) { return get(opts.publicKey) }

// Other options
func (opts *builderOptions) GetValidator() (string, bool)            { return get(opts.validator) }
func (opts *builderOptions) GetStakeOwner() (xc_types.Address, bool) { return get(opts.stakeOwner) }
func (opts *builderOptions) GetStakeAccount() (string, bool)         { return get(opts.stakeAccount) }

func (opts *builderOptions) GetAsset() (xc_types.IAsset, bool) { return get(opts.asset) }

type BuilderOption func(opts *builderOptions) error

func WithMemo(memo string) BuilderOption {
	return func(opts *builderOptions) error {
		opts.memo = &memo
		return nil
	}
}
func WithTimestamp(ts int64) BuilderOption {
	return func(opts *builderOptions) error {
		opts.timestamp = &ts
		return nil
	}
}
func WithPriority(priority xc_types.GasFeePriority) BuilderOption {
	return func(opts *builderOptions) error {
		opts.gasFeePriority = &priority
		return nil
	}
}
func WithPublicKey(publicKey []byte) BuilderOption {
	return func(opts *builderOptions) error {
		opts.publicKey = &publicKey
		return nil
	}
}

// Set an alternative owner of the stake from the from address
func WithStakeOwner(owner xc_types.Address) BuilderOption {
	return func(opts *builderOptions) error {
		opts.stakeOwner = &owner
		return nil
	}
}
func WithValidator(validator string) BuilderOption {
	return func(opts *builderOptions) error {
		opts.validator = &validator
		return nil
	}
}
func WithStakeAccount(account string) BuilderOption {
	return func(opts *builderOptions) error {
		opts.stakeAccount = &account
		return nil
	}
}

func WithAsset(asset xc_types.IAsset) BuilderOption {
	return func(opts *builderOptions) error {
		opts.asset = &asset
		return nil
	}
}

// Previously the crosschain abstraction would require callers to set options
// directly on the transaction input, if the interface was implemented on the input type.
// However, this is very clear or easy to use.  This function bridges the gap, to allow
// callers to use a more natural interface with options.  Chain transaction builders can
// call this to safely set provided options on the old transaction input setters.
func SetTxInputOptions(txInput xc_types.TxInput, options TransactionOptions, amount xc_types.BigInt) {
	if priority, ok := options.GetPriority(); ok && priority != "" {
		err := txInput.SetGasFeePriority(priority)
		if err != nil {
			zap.S().Error("failed to set gas fee priority", zap.Error(err))
		}
	}
	if pubkey, ok := options.GetPublicKey(); ok {
		if withPubkey, ok := txInput.(xc_types.TxInputWithPublicKey); ok {
			withPubkey.SetPublicKey(pubkey)
		}
	}

	if withAmount, ok := txInput.(xc_types.TxInputWithAmount); ok {
		withAmount.SetAmount(amount)
	}
	if memo, ok := options.GetMemo(); ok {
		if withMemo, ok := txInput.(xc_types.TxInputWithMemo); ok {
			withMemo.SetMemo(memo)
		}
	}
	if timeStamp, ok := options.GetTimestamp(); ok {
		if withUnix, ok := txInput.(xc_types.TxInputWithUnix); ok {
			withUnix.SetUnix(timeStamp)
		}
	}
}
