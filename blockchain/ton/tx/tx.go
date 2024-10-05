package tx

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	xc_types "github.com/openweb3-io/crosschain/types"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type Tx struct {
	CellBuilder     *cell.Builder
	ExternalMessage *tlb.ExternalMessage
	signatures      []xc_types.TxSignature
}

func (tx *Tx) Serialize() ([]byte, error) {
	if tx.ExternalMessage.Body == nil {
		return nil, fmt.Errorf("TON tx not yet signed and cannot be serialized")
	}
	ext, err := tlb.ToCell(tx.ExternalMessage)
	if err != nil {
		return nil, err
	}
	bz := ext.ToBOCWithFlags(false)
	return bz, nil
}

func (tx *Tx) Hash() xc_types.TxHash {
	if tx.ExternalMessage.Body == nil {
		return nil
	}
	ext, err := tlb.ToCell(tx.ExternalMessage)
	if err != nil {
		return nil
	}

	// Only way to calculate the correct hash is to reserialize it
	bz := ext.ToBOC()
	parsed, err := cell.FromBOC(bz)
	if err != nil {
		return nil
	}
	hash := parsed.Hash()

	// TON supports loading transaction by either hex, base64-std, or base64url.
	// We choose hex as it's preferred in explorers and doesn't have special characters.
	return xc_types.TxHash(hex.EncodeToString(hash))
}

func (tx *Tx) GetSignatures() []xc_types.TxSignature {
	return tx.signatures
}

func (tx *Tx) AddSignatures(sigs ...xc_types.TxSignature) error {
	if tx.ExternalMessage.Body != nil {
		return fmt.Errorf("already signed TON tx")
	}

	tx.signatures = sigs
	msg := cell.BeginCell().MustStoreSlice(sigs[0], 512).MustStoreBuilder(tx.CellBuilder).EndCell()
	tx.ExternalMessage.Body = msg
	return nil
}

func (tx Tx) Sighashes() ([]xc_types.TxDataToSign, error) {
	hash := tx.CellBuilder.EndCell().Hash()
	return []xc_types.TxDataToSign{hash}, nil
}

func NewTx(fromAddr *address.Address, cellBuilder *cell.Builder, stateInitMaybe *tlb.StateInit) *Tx {
	return &Tx{
		CellBuilder: cellBuilder,
		ExternalMessage: &tlb.ExternalMessage{
			// The address recieving funds.  Not sure why this is needed here.
			DstAddr: fromAddr,
			// This gets set when getting signed
			Body: nil,
			// This is needed only when an account is first used.
			StateInit: stateInitMaybe,
		},
	}
}

// Normal to hex as it doesn't have any special characters
func Normalize(txhash string) string {
	txhash = strings.TrimPrefix(txhash, "0x")
	if bz, err := hex.DecodeString(txhash); err == nil {
		return hex.EncodeToString(bz)
	}
	if bz, err := base64.StdEncoding.DecodeString(txhash); err == nil {
		return hex.EncodeToString(bz)
	}
	if bz, err := base64.RawURLEncoding.DecodeString(txhash); err == nil {
		return hex.EncodeToString(bz)
	}
	return txhash
}
