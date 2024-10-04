package ton

import (
	"fmt"

	"github.com/openweb3-io/crosschain/types"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type Tx struct {
	CellBuilder     *cell.Builder
	ExternalMessage *tlb.ExternalMessage
	signatures      []types.TxSignature
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

func (tx *Tx) Hash() types.TxHash {
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
	return types.TxHash(hash)
}

func (tx *Tx) GetSignatures() []types.TxSignature {
	return tx.signatures
}

func (tx *Tx) AddSignatures(sigs ...types.TxSignature) error {
	if tx.ExternalMessage.Body != nil {
		return fmt.Errorf("already signed TON tx")
	}

	tx.signatures = sigs
	msg := cell.BeginCell().MustStoreSlice(sigs[0], 512).MustStoreBuilder(tx.CellBuilder).EndCell()
	tx.ExternalMessage.Body = msg
	return nil
}

func (tx Tx) Sighashes() ([]types.TxDataToSign, error) {
	hash := tx.CellBuilder.EndCell().Hash()
	return []types.TxDataToSign{hash}, nil
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
