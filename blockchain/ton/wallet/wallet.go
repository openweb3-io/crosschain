package wallet

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/openweb3-io/crosschain/signer"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/adnl"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type Version int

const (
	V1R1               Version = 11
	V1R2               Version = 12
	V1R3               Version = 13
	V2R1               Version = 21
	V2R2               Version = 22
	V3R1               Version = 31
	V3R2               Version = 32
	V3                         = V3R2
	V4R1               Version = 41
	V4R2               Version = 42
	V5R1               Version = 51
	HighloadV2R2       Version = 122
	HighloadV2Verified Version = 123
	HighloadV3         Version = 300
	Lockup             Version = 200
	Unknown            Version = 0
)

const (
	CarryAllRemainingBalance       = 128
	CarryAllRemainingIncomingValue = 64
	DestroyAccountIfZero           = 32
	IgnoreErrors                   = 2
	PayGasSeparately               = 1
)

func (v Version) String() string {
	if v == Unknown {
		return "unknown"
	}

	switch v {
	case HighloadV2R2:
		return "highload V2R2"
	case HighloadV2Verified:
		return "highload V2R2 verified"
	}

	if v/100 == 2 {
		return "lockup"
	}
	if v/10 > 0 && v/10 < 10 {
		return fmt.Sprintf("V%dR%d", v/10, v%10)
	}
	return fmt.Sprintf("%d", v)
}

var (
	walletCodeHex = map[Version]string{
		V1R1: _V1R1CodeHex, V1R2: _V1R2CodeHex, V1R3: _V1R3CodeHex,
		V2R1: _V2R1CodeHex, V2R2: _V2R2CodeHex,
		V3R1: _V3R1CodeHex, V3R2: _V3R2CodeHex,
		V4R1: _V4R1CodeHex, V4R2: _V4R2CodeHex,
		V5R1:         _V5R1CodeHex,
		HighloadV2R2: _HighloadV2R2CodeHex, HighloadV2Verified: _HighloadV2VerifiedCodeHex,
		HighloadV3: _HighloadV3CodeHex,
		Lockup:     _LockupCodeHex,
	}
	walletCodeBOC = map[Version][]byte{}
	walletCode    = map[Version]*cell.Cell{}
)

func init() {
	var err error

	for ver, codeHex := range walletCodeHex {
		walletCodeBOC[ver], err = hex.DecodeString(codeHex)
		if err != nil {
			panic(err)
		}
		walletCode[ver], err = cell.FromBOC(walletCodeBOC[ver])
		if err != nil {
			panic(err)
		}
	}
}

// defining some funcs this way to mock for tests
var randUint32 = func() uint32 {
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	return binary.LittleEndian.Uint32(buf)
}

var timeNow = time.Now

var (
	ErrUnsupportedWalletVersion = errors.New("wallet version is not supported")
	ErrTxWasNotConfirmed        = errors.New("transaction was not confirmed in a given deadline, but it may still be confirmed later")
	// Deprecated: use ton.ErrTxWasNotFound
	ErrTxWasNotFound = errors.New("requested transaction is not found")
)

type TonAPI interface {
	WaitForBlock(seqno uint32) ton.APIClientWrapped
	Client() ton.LiteClient
	CurrentMasterchainInfo(ctx context.Context) (*ton.BlockIDExt, error)
	GetAccount(ctx context.Context, block *ton.BlockIDExt, addr *address.Address) (*tlb.Account, error)
	SendExternalMessage(ctx context.Context, msg *tlb.ExternalMessage) error
	RunGetMethod(ctx context.Context, blockInfo *ton.BlockIDExt, addr *address.Address, method string, params ...interface{}) (*ton.ExecutionResult, error)
	ListTransactions(ctx context.Context, addr *address.Address, num uint32, lt uint64, txHash []byte) ([]*tlb.Transaction, error)
	FindLastTransactionByInMsgHash(ctx context.Context, addr *address.Address, msgHash []byte, maxTxNumToScan ...int) (*tlb.Transaction, error)
	FindLastTransactionByOutMsgHash(ctx context.Context, addr *address.Address, msgHash []byte, maxTxNumToScan ...int) (*tlb.Transaction, error)
}

type Message struct {
	Mode            uint8
	InternalMessage *tlb.InternalMessage
}

type SeqnoFetcher func(ctx context.Context, subWallet uint32) (uint32, error)

type Wallet struct {
	publicKey ed25519.PublicKey
	addr      *address.Address
	ver       VersionConfig

	// Can be used to operate multiple wallets with the same key and version.
	// use GetSubwallet if you need it.
	subwallet uint32

	// Stores a pointer to implementation of the version related functionality
	spec         any
	seqnoFetcher SeqnoFetcher
}

func FromAddress(seqnoFetcher SeqnoFetcher, addr *address.Address, version VersionConfig, pSubwallet *uint32) (*Wallet, error) {
	var subwallet uint32 = DefaultSubwallet
	if pSubwallet != nil {
		subwallet = *pSubwallet
	}

	// default subwallet depends on wallet type
	switch version.(type) {
	case ConfigV5R1:
		subwallet = 0
	}

	if seqnoFetcher == nil {
		seqnoFetcher = func(ctx context.Context, subWallet uint32) (uint32, error) {
			return 0, nil
		}
	}

	w := &Wallet{
		seqnoFetcher: seqnoFetcher,
		addr:         addr,
		ver:          version,
		subwallet:    subwallet,
	}

	var err error

	w.spec, err = getSpec(w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func FromPublicKey(api TonAPI, publicKey ed25519.PublicKey, version VersionConfig, pSubwallet *uint32) (*Wallet, error) {
	var subwallet uint32 = DefaultSubwallet
	if pSubwallet != nil {
		subwallet = *pSubwallet
	}

	// default subwallet depends on wallet type
	switch version.(type) {
	case ConfigV5R1:
		subwallet = 0
	}

	addr, err := AddressFromPubKey(publicKey, version, subwallet)
	if err != nil {
		return nil, err
	}

	w := &Wallet{
		// api:       api,
		publicKey: publicKey,
		addr:      addr,
		ver:       version,
		subwallet: subwallet,
	}

	w.spec, err = getSpec(w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func getSpec(w *Wallet) (any, error) {
	switch v := w.ver.(type) {
	case Version, ConfigV5R1:
		regular := SpecRegular{
			wallet:      w,
			messagesTTL: 60 * 3, // default ttl 3 min
		}

		/*
			seqnoFetcher := func(ctx context.Context, subWallet uint32) (uint32, error) {
				block, err := w.api.CurrentMasterchainInfo(ctx)
				if err != nil {
					return 0, fmt.Errorf("failed to get block: %w", err)
				}

				resp, err := w.api.WaitForBlock(block.SeqNo).RunGetMethod(ctx, block, w.addr, "seqno")
				if err != nil {
					if cErr, ok := err.(ton.ContractExecError); ok && cErr.Code == ton.ErrCodeContractNotInitialized {
						return 0, nil
					}
					return 0, fmt.Errorf("get seqno err: %w", err)
				}

				iSeq, err := resp.Int(0)
				if err != nil {
					return 0, fmt.Errorf("failed to parse seqno: %w", err)
				}
				return uint32(iSeq.Uint64()), nil
			}*/

		switch x := w.ver.(type) {
		case ConfigV5R1:
			if x.NetworkGlobalID == 0 {
				return nil, fmt.Errorf("NetworkGlobalID should be set in v5 config")
			}
			return &SpecV5R1{SpecRegular: regular, SpecSeqno: SpecSeqno{seqnoFetcher: w.seqnoFetcher}, config: x}, nil
		}

		switch v {
		case V3R1, V3R2:
			return &SpecV3{regular, SpecSeqno{seqnoFetcher: w.seqnoFetcher}}, nil
		case V4R1, V4R2:
			return &SpecV4R2{regular, SpecSeqno{seqnoFetcher: w.seqnoFetcher}}, nil
		case HighloadV2R2, HighloadV2Verified:
			return &SpecHighloadV2R2{regular, SpecQuery{}}, nil
		case HighloadV3:
			return nil, fmt.Errorf("use ConfigHighloadV3 for highload v3 spec")
		case V5R1:
			return nil, fmt.Errorf("use ConfigV5R1 for v5 spec")
		}
	case ConfigHighloadV3:
		return &SpecHighloadV3{wallet: w, config: v}, nil
	}

	return nil, fmt.Errorf("cannot init spec: %w", ErrUnsupportedWalletVersion)
}

// Address - returns old (bounce) version of wallet address
// DEPRECATED: because of address reform, use WalletAddress,
// it will return UQ format
func (w *Wallet) Address() *address.Address {
	return w.addr
}

// WalletAddress - returns new standard non bounce address
func (w *Wallet) WalletAddress() *address.Address {
	return w.addr.Bounce(false)
}

func (w *Wallet) GetSubwallet(ctx context.Context, subwallet uint32) (*Wallet, error) {
	addr, err := AddressFromPubKey(w.publicKey, w.ver, subwallet)
	if err != nil {
		return nil, err
	}

	sub := &Wallet{
		addr:      addr,
		ver:       w.ver,
		subwallet: subwallet,
	}

	sub.spec, err = getSpec(sub)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func (w *Wallet) GetSpec() any {
	return w.spec
}

func (w *Wallet) BuildMessages(ctx context.Context, withStateInit bool, messages []*Message) (_ *cell.Builder, err error) {
	/*
		var stateInit *tlb.StateInit
		if withStateInit {
			publicKey, err := w.signer.PublicKey(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get public key: %w", err)
			}

			stateInit, err = GetStateInit(publicKey, w.ver, w.subwallet)
			if err != nil {
				return nil, fmt.Errorf("failed to get state init: %w", err)
			}
		}
	*/

	var builder *cell.Builder
	switch v := w.ver.(type) {
	case Version, ConfigV5R1:
		if _, ok := v.(ConfigV5R1); ok {
			v = V5R1
		}

		switch v {
		case V3R2, V3R1, V4R2, V4R1, V5R1:
			builder, err = w.spec.(RegularBuilder).BuildMessage(ctx, !withStateInit, nil, messages)
			if err != nil {
				return nil, fmt.Errorf("build message err: %w", err)
			}
		case HighloadV2R2, HighloadV2Verified:
			builder, err = w.spec.(*SpecHighloadV2R2).BuildMessage(ctx, messages)
			if err != nil {
				return nil, fmt.Errorf("build message err: %w", err)
			}
		case HighloadV3:
			return nil, fmt.Errorf("use ConfigHighloadV3 for highload v3 spec")
		default:
			return nil, fmt.Errorf("send is not yet supported: %w", ErrUnsupportedWalletVersion)
		}
	case ConfigHighloadV3:
		builder, err = w.spec.(*SpecHighloadV3).BuildMessage(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("build message err: %w", err)
		}
	default:
		return nil, fmt.Errorf("send is not yet supported: %w", ErrUnsupportedWalletVersion)
	}

	return builder, nil
	/*
		return &tlb.ExternalMessage{
			DstAddr:   w.addr,
			StateInit: stateInit,
			Body:      msg,
		}, nil
	*/
}

func (w *Wallet) BuildTransfer(to *address.Address, amount tlb.Coins, bounce bool, comment string) (_ *Message, err error) {
	var body *cell.Cell
	if comment != "" {
		body, err = CreateCommentCell(comment)
		if err != nil {
			return nil, err
		}
	}

	return &Message{
		Mode: PayGasSeparately + IgnoreErrors,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      bounce,
			DstAddr:     to,
			Amount:      amount,
			Body:        body,
		},
	}, nil
}

func CreateCommentCell(text string) (*cell.Cell, error) {
	// comment ident
	root := cell.BeginCell().MustStoreUInt(0, 32)

	if err := root.StoreStringSnake(text); err != nil {
		return nil, fmt.Errorf("failed to build comment: %w", err)
	}

	return root.EndCell(), nil
}

const EncryptedCommentOpcode = 0x2167da4b

func DecryptCommentCell(commentCell *cell.Cell, sender *address.Address, ourKey ed25519.PrivateKey, theirKey ed25519.PublicKey) ([]byte, error) {
	slc := commentCell.BeginParse()
	op, err := slc.LoadUInt(32)
	if err != nil {
		return nil, fmt.Errorf("failed to load op code: %w", err)
	}

	if op != EncryptedCommentOpcode {
		return nil, fmt.Errorf("opcode not match encrypted comment")
	}

	xorKey, err := slc.LoadSlice(256)
	if err != nil {
		return nil, fmt.Errorf("failed to load xor key: %w", err)
	}
	for i := 0; i < 32; i++ {
		xorKey[i] ^= theirKey[i]
	}

	if !bytes.Equal(xorKey, ourKey.Public().(ed25519.PublicKey)) {
		return nil, fmt.Errorf("message was encrypted not for the given keys")
	}

	msgKey, err := slc.LoadSlice(128)
	if err != nil {
		return nil, fmt.Errorf("failed to load xor key: %w", err)
	}

	sharedKey, err := adnl.SharedKey(ourKey, theirKey)
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared key: %w", err)
	}

	h := hmac.New(sha512.New, sharedKey)
	h.Write(msgKey)
	x := h.Sum(nil)

	data, err := slc.LoadBinarySnake()
	if err != nil {
		return nil, fmt.Errorf("failed to load snake encrypted data: %w", err)
	}

	if len(data) < 32 || len(data)%16 != 0 {
		return nil, fmt.Errorf("invalid data")
	}

	c, err := aes.NewCipher(x[:32])
	if err != nil {
		return nil, err
	}
	enc := cipher.NewCBCDecrypter(c, x[32:48])
	enc.CryptBlocks(data, data)

	if data[0] > 31 {
		return nil, fmt.Errorf("invalid prefix size %d", data[0])
	}

	h = hmac.New(sha512.New, []byte(sender.String()))
	h.Write(data)
	if !bytes.Equal(msgKey, h.Sum(nil)[:16]) {
		return nil, fmt.Errorf("incorrect msg key")
	}

	return data[data[0]:], nil
}

func CreateEncryptedCommentCell(ctx context.Context, text string, senderAddr *address.Address, signer signer.Signer, theirKey ed25519.PublicKey) (*cell.Cell, error) {
	// encrypted comment op code
	root := cell.BeginCell().MustStoreUInt(EncryptedCommentOpcode, 32)

	sharedKey, err := signer.SharedKey(theirKey)
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared key: %w", err)
	}

	data := []byte(text)

	pfxSz := 16
	if len(data)%16 != 0 {
		pfxSz += 16 - (len(data) % 16)
	}

	pfx := make([]byte, pfxSz)
	pfx[0] = byte(len(pfx))
	if _, err = rand.Read(pfx[1:]); err != nil {
		return nil, fmt.Errorf("rand gen err: %w", err)
	}
	data = append(pfx, data...)

	h := hmac.New(sha512.New, []byte(senderAddr.String()))
	h.Write(data)
	msgKey := h.Sum(nil)[:16]

	h = hmac.New(sha512.New, sharedKey)
	h.Write(msgKey)
	x := h.Sum(nil)

	c, err := aes.NewCipher(x[:32])
	if err != nil {
		return nil, err
	}

	enc := cipher.NewCBCEncrypter(c, x[32:48])
	enc.CryptBlocks(data, data)

	xorKey, err := signer.PublicKey(ctx)
	if err != nil {
		return nil, err
	}

	for i := 0; i < 32; i++ {
		xorKey[i] ^= theirKey[i]
	}

	root.MustStoreSlice(xorKey, 256)
	root.MustStoreSlice(msgKey, 128)

	if err := root.StoreBinarySnake(data); err != nil {
		return nil, fmt.Errorf("failed to build comment: %w", err)
	}

	return root.EndCell(), nil
}

func SimpleMessage(to *address.Address, amount tlb.Coins, payload *cell.Cell) *Message {
	return &Message{
		Mode: PayGasSeparately + IgnoreErrors,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     to,
			Amount:      amount,
			Body:        payload,
		},
	}
}

// SimpleMessageAutoBounce - will determine bounce flag from address
func SimpleMessageAutoBounce(to *address.Address, amount tlb.Coins, payload *cell.Cell) *Message {
	return &Message{
		Mode: PayGasSeparately + IgnoreErrors,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      to.IsBounceable(),
			DstAddr:     to,
			Amount:      amount,
			Body:        payload,
		},
	}
}
