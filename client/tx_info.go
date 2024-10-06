package client

import (
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"time"

	"github.com/openweb3-io/crosschain/normalize"
	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/tidwall/btree"
	"go.uber.org/zap"
)

type TransactionName string
type AssetName string
type AddressName string

func NewTransactionName(chain xc_types.NativeAsset, txHash string) TransactionName {
	txHash = normalize.TransactionHash(txHash, chain)
	name := filepath.Join("chains", string(chain), "transactions", txHash)
	return TransactionName(name)
}

func NewAssetName(chain xc_types.NativeAsset, contractOrNativeAsset string) AssetName {
	if contractOrNativeAsset == "" {
		contractOrNativeAsset = string(chain)
	}
	if contractOrNativeAsset != string(chain) {
		contractOrNativeAsset = normalize.Normalize(contractOrNativeAsset, chain)
	}
	name := filepath.Join("chains", string(chain), "assets", contractOrNativeAsset)
	return AssetName(name)
}

func NewAddressName(chain xc_types.NativeAsset, address string) AddressName {
	if address == "" {
		address = string(chain)
	}
	if address != string(chain) {
		address = normalize.Normalize(address, chain)
	}
	name := filepath.Join("chains", string(chain), "addresses", address)
	return AddressName(name)
}

func (name TransactionName) Chain() string {
	p := strings.Split(string(name), "/")
	if len(p) > 3 {
		return p[1]
	} else {
		return ""
	}
}

type Balance struct {
	Asset    AssetName                     `json:"asset"`
	Contract xc_types.ContractAddress      `json:"contract"`
	Balance  xc_types.BigInt               `json:"balance"`
	Amount   *xc_types.AmountHumanReadable `json:"amount,omitempty"`
}

func NewBalance(chain xc_types.NativeAsset, contract xc_types.ContractAddress, balance xc_types.BigInt, decimals *int) *Balance {
	assetName := NewAssetName(chain, string(contract))
	var amount *xc_types.AmountHumanReadable
	return &Balance{
		assetName,
		contract,
		balance,
		amount,
	}
}

type LegacyBalances map[AssetName]xc_types.BigInt
type TransferSource struct {
	From   AddressName     `json:"from"`
	Asset  AssetName       `json:"asset"`
	Amount xc_types.BigInt `json:"amount"`
}

type BalanceChange struct {
	Asset    AssetName                     `json:"asset"`
	Contract xc_types.ContractAddress      `json:"contract"`
	Balance  xc_types.BigInt               `json:"balance"`
	Amount   *xc_types.AmountHumanReadable `json:"amount,omitempty"`
	Address  AddressName                   `json:"address"`
}
type Transfer struct {
	// required: source debits
	From []*BalanceChange `json:"from"`
	// required: destination credits
	To []*BalanceChange `json:"to"`

	Memo string `json:"memo,omitempty"`

	chain xc_types.NativeAsset
}

type Block struct {
	// required: set the blockheight of the transaction
	Height uint64 `json:"height"`
	// required: set the hash of the block of the transaction
	Hash string `json:"hash"`
	// required: set the time of the block of the transaction
	Time time.Time `json:"time"`
}

type Stake struct {
	Balance   xc_types.BigInt `json:"balance"`
	Validator string          `json:"validator"`
	Account   string          `json:"account"`
	Address   string          `json:"address"`
}
type Unstake struct {
	Balance   xc_types.BigInt `json:"balance"`
	Validator string          `json:"validator"`
	Account   string          `json:"account"`
	Address   string          `json:"address"`
}

type StakeEvent interface {
	GetValidator() string
}

var _ StakeEvent = &Stake{}
var _ StakeEvent = &Unstake{}

func (s *Stake) GetValidator() string {
	return s.Validator
}
func (s *Unstake) GetValidator() string {
	return s.Validator
}

// This should roughly match stoplight
type TxInfo struct {
	Name TransactionName `json:"name"`
	// required: set the transaction hash/id
	Hash string `json:"hash"`
	// required: set the chain
	Chain xc_types.NativeAsset `json:"chain"`

	// required: set the block info
	Block *Block `json:"block"`

	// required: set any movements
	Transfers []*Transfer `json:"transfers"`

	// output-only: calculate via .CalcuateFees() method
	Fees []*Balance `json:"fees"`

	// Native staking events
	Stakes   []*Stake   `json:"stakes,omitempty"`
	Unstakes []*Unstake `json:"unstakes,omitempty"`

	// required: set the confirmations at time of querying the info
	Confirmations uint64 `json:"confirmations"`
	// optional: set the error of the transaction if there was an error
	Error *string `json:"error,omitempty"`
}

func NewBlock(height uint64, hash string, time time.Time) *Block {
	return &Block{
		height,
		hash,
		time,
	}
}

func NewBalanceChange(chain xc_types.NativeAsset, contract xc_types.ContractAddress, address xc_types.Address, balance xc_types.BigInt, decimals *int) *BalanceChange {
	if contract == "" {
		contract = xc_types.ContractAddress(chain)
	}
	asset := NewAssetName(chain, string(contract))
	addressName := NewAddressName(chain, string(address))
	var amount *xc_types.AmountHumanReadable

	return &BalanceChange{
		asset,
		contract,
		balance,
		amount,
		addressName,
	}
}

func NewTxInfo(block *Block, chain xc_types.NativeAsset, hash string, confirmations uint64, err *string) *TxInfo {
	transfers := []*Transfer{}
	fees := []*Balance{}
	var stakes []*Stake = nil
	var unstakes []*Unstake = nil
	name := NewTransactionName(chain, hash)
	return &TxInfo{
		name,
		hash,
		chain,
		block,
		transfers,
		fees,
		stakes,
		unstakes,
		confirmations,
		err,
	}
}
func (info *TxInfo) AddSimpleTransfer(from xc_types.Address, to xc_types.Address, contract xc_types.ContractAddress, balance xc_types.BigInt, decimals *int, memo string) {
	tf := NewTransfer(info.Chain)
	tf.SetMemo(memo)
	tf.AddSource(from, contract, balance, decimals)
	tf.AddDestination(to, contract, balance, decimals)
	info.Transfers = append(info.Transfers, tf)
}

func (info *TxInfo) AddFee(from xc_types.Address, contract xc_types.ContractAddress, balance xc_types.BigInt, decimals *int) {
	tf := NewTransfer(info.Chain)
	tf.AddSource(from, contract, balance, decimals)
	// no destination
	info.Transfers = append(info.Transfers, tf)
}
func (info *TxInfo) AddTransfer(transfer *Transfer) {
	info.Transfers = append(info.Transfers, transfer)
}

func (info *TxInfo) CalculateFees() []*Balance {
	// use btree map to get deterministic order
	var netBalances = btree.NewMap[AssetName, *big.Int](1)
	contracts := map[AssetName]xc_types.ContractAddress{}
	for _, tf := range info.Transfers {
		for _, from := range tf.From {
			netBalances.Set(from.Asset, xc_types.NewBigIntFromUint64(0).Int())
			contracts[from.Asset] = from.Contract
		}
		for _, to := range tf.To {
			netBalances.Set(to.Asset, xc_types.NewBigIntFromUint64(0).Int())
			contracts[to.Asset] = to.Contract
		}
	}
	for _, tf := range info.Transfers {
		for _, from := range tf.From {
			bal, _ := netBalances.GetMut(from.Asset)
			bal.Add(bal, from.Balance.Int())
		}

		for _, to := range tf.To {
			bal, _ := netBalances.GetMut(to.Asset)
			bal.Sub(bal, to.Balance.Int())
		}
	}
	balances := []*Balance{}
	zero := big.NewInt(0)
	netBalances.Ascend("", func(asset AssetName, net *big.Int) bool {
		if net.Cmp(zero) != 0 {
			balances = append(balances, NewBalance(info.Chain, contracts[asset], xc_types.BigInt(*net), nil))
		}
		return true
	})
	return balances
}

func NewTransfer(chain xc_types.NativeAsset) *Transfer {
	// avoid serializing null's in json
	from := []*BalanceChange{}
	to := []*BalanceChange{}
	memo := ""
	return &Transfer{from, to, memo, chain}
}

func (tf *Transfer) AddSource(from xc_types.Address, contract xc_types.ContractAddress, balance xc_types.BigInt, decimals *int) {
	tf.From = append(tf.From, NewBalanceChange(tf.chain, contract, from, balance, decimals))
}
func (tf *Transfer) AddDestination(to xc_types.Address, contract xc_types.ContractAddress, balance xc_types.BigInt, decimals *int) {
	tf.To = append(tf.To, NewBalanceChange(tf.chain, contract, to, balance, decimals))
}
func (tf *Transfer) SetMemo(memo string) {
	tf.Memo = memo
}

type LegacyTxInfoMappingType string

var Utxo LegacyTxInfoMappingType = "utxo"
var Account LegacyTxInfoMappingType = "account"

func TxInfoFromLegacy(chain xc_types.NativeAsset, legacyTx *xc_types.LegacyTxInfo, mappingType LegacyTxInfoMappingType) TxInfo {
	var errMsg *string
	if legacyTx.Status == xc_types.TxStatusFailure {
		msg := "transaction failed"
		errMsg = &msg
	}
	if legacyTx.Error != "" {
		errMsg = &legacyTx.Error
	}

	txInfo := NewTxInfo(
		NewBlock(uint64(legacyTx.BlockIndex), legacyTx.BlockHash, time.Unix(legacyTx.BlockTime, 0)),
		chain,
		legacyTx.TxID,
		uint64(legacyTx.Confirmations),
		errMsg,
	)

	if mappingType == Utxo {
		// utxo movements should be mapped as one large multitransfer
		tf := NewTransfer(chain)
		for _, source := range legacyTx.Sources {
			tf.AddSource(source.Address, source.ContractAddress, source.Amount, nil)
		}

		for _, dest := range legacyTx.Destinations {
			tf.AddDestination(dest.Address, dest.ContractAddress, dest.Amount, nil)
		}
		txInfo.AddTransfer(tf)
	} else {
		// map as one-to-one
		for i, dest := range legacyTx.Destinations {
			fromAddr := legacyTx.From
			if i < len(legacyTx.Sources) {
				fromAddr = legacyTx.Sources[i].Address
			}

			txInfo.AddSimpleTransfer(fromAddr, dest.Address, dest.ContractAddress, dest.Amount, nil, dest.Memo)
		}
	}
	zero := big.NewInt(0)
	if legacyTx.Fee.Cmp((*xc_types.BigInt)(zero)) != 0 {
		txInfo.AddFee(legacyTx.From, legacyTx.FeeContract, legacyTx.Fee, nil)
	}

	txInfo.Fees = txInfo.CalculateFees()

	for _, ev := range legacyTx.GetStakeEvents() {
		switch ev := ev.(type) {
		case *Stake:
			txInfo.Stakes = append(txInfo.Stakes, ev)
		case *Unstake:
			txInfo.Unstakes = append(txInfo.Unstakes, ev)
		default:
			zap.S().Warn("unknown stake event type: " + fmt.Sprintf("%T", ev))
		}
	}
	return *txInfo
}
