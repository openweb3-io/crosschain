package ton_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"math/big"
	"testing"

	xcbuilder "github.com/openweb3-io/crosschain/builder"

	"github.com/croutondefi/stonfi-go"
	"github.com/openweb3-io/crosschain/blockchain/ton"
	"github.com/openweb3-io/crosschain/blockchain/ton/tx"
	"github.com/openweb3-io/crosschain/blockchain/ton/wallet"
	"github.com/openweb3-io/crosschain/signer"
	xc_types "github.com/openweb3-io/crosschain/types"

	"github.com/stretchr/testify/suite"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	_ton "github.com/xssnick/tonutils-go/ton"
)

const (
	USDTJettonMainnetAddress = "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs"
	USDTJettonTestnetAddress = "kQD0GKBM8ZbryVk2aESmzfU6b9b_8era_IkvBSELujFZPsyy"

	USDTJettonMainnetWalletAddress = "EQBPy4gmH8pf1pfBwbMw3PdtsO8Aj2rxmVfhM6jpAGHSmTnr"
)

type ClientTestSuite struct {
	suite.Suite
	client          *ton.Client
	account1PrivKey ed25519.PrivateKey
	account1PubKey  ed25519.PublicKey
	account1Address *address.Address
	account2PrivKey ed25519.PrivateKey
	account2PubKey  ed25519.PublicKey
	account2Address *address.Address
	account1Signer  signer.Signer
	account2Signer  signer.Signer
}

// get test coin from this telegram bot: https://web.telegram.org/k/#@testgiver_ton_bot
// testnet blockchain browser: https://testnet.tonscan.org/
// get test coin from https://faucet.tonxapi.com/
// exchange ton to jUSDT bridge.ton.org

const (
	AuthSecret            = "AEXRCJJGQBXCFWQAAAAD3RYTVUWCXT5JW6YN2QU7LHXMKPMOXHFB75P4JSD52AVOVQWPGNY"
	account1PubKeyBase64  = "DwYgZ731p93G922Gc9k/AEEJv3kqzcla+rBZ3NyVOXM="
	account1PrivKeyBase64 = "XsRM5LXm6T4xOIL+I7tSFCy6TIZBZZr04ofHdI5DSycPBiBnvfWn3cb3bYZz2T8AQQm/eSrNyVr6sFnc3JU5cw=="
	account2PubKeyBase64  = "7czJlRjDE3wZl4SdbTiMPjOBTaFouafXFDVPkZpnqs8="
	account2PrivKeyBase64 = "2Lc0RPv26SOIZa9Hhbb0wzv1O/njY1SpdOOU6fRE5xHtzMmVGMMTfBmXhJ1tOIw+M4FNoWi5p9cUNU+Rmmeqzw=="
)

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) SetupTest() {
	account1PubKeyBytes, err := base64.StdEncoding.DecodeString(account1PubKeyBase64)
	suite.Require().NoError(err)
	account1PubKey := ed25519.PublicKey(account1PubKeyBytes)

	account1PrivKeyBytes, err := base64.StdEncoding.DecodeString(account1PrivKeyBase64)
	suite.Require().NoError(err)
	account1PrivKey := ed25519.PrivateKey(account1PrivKeyBytes)

	account2PubKeyBytes, err := base64.StdEncoding.DecodeString(account2PubKeyBase64)
	suite.Require().NoError(err)
	account2PubKey := ed25519.PublicKey(account2PubKeyBytes)

	account2PrivKeyBytes, err := base64.StdEncoding.DecodeString(account2PrivKeyBase64)
	suite.Require().NoError(err)
	account2PrivKey := ed25519.PrivateKey(account2PrivKeyBytes)

	suite.account1PubKey = account1PubKey
	suite.account1PrivKey = account1PrivKey
	suite.account2PubKey = account2PubKey
	suite.account2PrivKey = account2PrivKey

	account1Address, err := wallet.AddressFromPubKey(account1PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)
	suite.account1Address = account1Address
	fmt.Printf("account1Address: %+v\n", account1Address.Dump())

	account2Address, err := wallet.AddressFromPubKey(account2PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)
	suite.account2Address = account2Address
	fmt.Printf("account1Address: %+v\n", account2Address.Dump())

	suite.account1Signer = ton.NewLocalSigner(account1PrivKey)
	suite.account2Signer = ton.NewLocalSigner(account2PrivKey)

	client, err := ton.NewClient(&xc_types.ChainConfig{
		AuthSecret: AuthSecret,
	})
	suite.Require().NoError(err)
	suite.client = client
}

func (suite *ClientTestSuite) TearDownTest() {
}

func (suite *ClientTestSuite) Test_Tranfser() {
	ctx := context.Background()

	from, err := wallet.AddressFromPubKey(suite.account1PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)
	to, err := wallet.AddressFromPubKey(suite.account2PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)

	readableAmount, err := xc_types.NewAmountHumanReadableFromStr("0.01")
	suite.Require().NoError(err)
	blockchainAmount := readableAmount.ToBlockchain(9)

	args, err := xcbuilder.NewTransferArgs(
		xc_types.Address(from.String()),
		xc_types.Address(to.String()),
		blockchainAmount,
		xcbuilder.WithMemo("test transfer ton"),
	)
	suite.Require().NoError(err)

	input, err := suite.client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	builder, err := ton.NewTxBuilder(&xc_types.ChainConfig{Chain: xc_types.TON, Decimals: 9})
	suite.Require().NoError(err)

	tx, err := builder.NewTransfer(args, input)
	suite.Require().NoError(err)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().GreaterOrEqual(len(sighashes), 1)

	signature, err := suite.account1Signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err)

	err = suite.client.BroadcastTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash: %v\n", tx.Hash())
}

func (suite *ClientTestSuite) Test_EstimateGas() {
	ctx := context.Background()

	from, err := wallet.AddressFromPubKey(suite.account1PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)
	to, err := wallet.AddressFromPubKey(suite.account2PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)

	builder, err := ton.NewTxBuilder(&xc_types.ChainConfig{Chain: xc_types.TON, Decimals: 9})
	suite.Require().NoError(err)

	args, err := xcbuilder.NewTransferArgs(
		xc_types.Address(from.String()),
		xc_types.Address(to.String()),
		xc_types.NewBigIntFromUint64(10000000),
		xcbuilder.WithMemo("test"),
	)
	suite.Require().NoError(err)

	input, err := suite.client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err)

	tx, err := builder.NewTransfer(args, input)
	suite.Require().NoError(err)

	amount, err := suite.client.EstimateGas(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("amount: %v\n", amount)
}

/**
 * work for mainnet
 */
func (suite *ClientTestSuite) Test_SwapFromTonToUSDT() {
	ctx := context.Background()
	// Address from swap

	isTestnet := false

	// jetton master
	var askJettonAddress *address.Address
	// proxy ton address
	var proxyTonAddress *address.Address
	var routerAddr *address.Address
	if isTestnet {
		// askJettonAddress := address.MustParseAddr("EQDB8JYMzpiOxjCx7leP5nYkchF72PdbWT1LV7ym1uAedINh") // STON testnet - STONT
		// EQBynBO23ywHy_CgarY9NK9FTz0yDsG82PtcbSTQgGoXwiuA

		askJettonAddress = address.MustParseAddr("kQD0GKBM8ZbryVk2aESmzfU6b9b_8era_IkvBSELujFZPsyy") // USDT contract testnet
		proxyTonAddress = address.MustParseAddr("EQAcOvXSnnOhCdLYc6up2ECYwtNNTzlmOlidBeCs5cFPVwuG")
		routerAddr = address.MustParseAddr("EQBsGx9ArADUrREB34W-ghgsCgBShvfUr4Jvlu-0KGc33Rbt") // testnet
	} else {
		proxyTonAddress = address.MustParseAddr("EQCM3B12QK1e4yZSf8GtBRT0aLMNyEsBc_DhVfRRtOEffLez") // mainnet
		routerAddr = address.MustParseAddr("EQB3ncyBUTjZUA5EnFKR5_EnOMI9V1tTEAAPaiU71gc4TiUt")      // mainnet
		askJettonAddress = address.MustParseAddr(USDTJettonMainnetAddress)                          // USDT contract mainnet
	}

	client := liteclient.NewConnectionPool()

	// from cfg
	// url := "https://ton-blockchain.github.io/testnet-global.config.json"
	url := "https://api.tontech.io/ton/wallet-mainnet.autoconf.json"
	err := client.AddConnectionsFromConfigUrl(context.Background(), url)
	suite.Require().NoError(err)
	liteApiClient := _ton.NewAPIClient(client)

	routerRevV1 := stonfi.NewRouterRevisionV1(liteApiClient, routerAddr)
	router := stonfi.NewRouter(liteApiClient, routerAddr, routerRevV1)

	rm, _ := xc_types.NewAmountHumanReadableFromStr("0.1")
	offerAmount := rm.ToBlockchain(9)

	data, err := router.BuildSwapProxyTonTxParams(ctx, stonfi.SwapProxyTonParams{
		UserWalletAddress: suite.account1Address,
		MinAskAmount:      big.NewInt(1), // min jetton swaped
		OfferAmount:       offerAmount.Int(),
		ProxyTonAddress:   proxyTonAddress,
		AskJettonAddress:  askJettonAddress,
		QueryId:           294082696817434,
	})
	suite.Require().NoError(err)

	w, err := wallet.FromAddress(nil, suite.account1Address, wallet.V4R2)
	suite.Require().NoError(err)

	cellBuilder, err := w.BuildMessages(ctx, false, []*wallet.Message{
		{
			Mode: wallet.PayGasSeparately + wallet.IgnoreErrors,
			InternalMessage: &tlb.InternalMessage{
				Bounce:  false,
				DstAddr: data.To,
				Amount:  tlb.FromNanoTON(data.Amount),
				Body:    data.Payload,
			},
		},
	})
	suite.Require().NoError(err)

	tx := tx.NewTx(w.Address(), cellBuilder, nil)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().GreaterOrEqual(len(sighashes), 1)

	signature, err := suite.account1Signer.Sign(sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err)

	err = suite.client.BroadcastTx(ctx, tx)
	suite.Require().NoError(err)
}

func (suite *ClientTestSuite) Test_TransferJetton() {
	ctx := context.Background()

	contractAddress := xc_types.ContractAddress(USDTJettonMainnetAddress)

	from, err := wallet.AddressFromPubKey(suite.account1PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)
	to, err := wallet.AddressFromPubKey(suite.account2PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)

	readableAmount, err := xc_types.NewAmountHumanReadableFromStr("0.01")
	suite.Require().NoError(err)
	amount := readableAmount.ToBlockchain(6)
	suite.Require().NoError(err)

	jettonBalance, err := suite.client.FetchBalanceForAsset(ctx, xc_types.Address(from.String()), contractAddress)
	suite.Require().NoError(err, "error FetchBalanceForAsset")

	if jettonBalance.Cmp(&amount) < 0 {
		suite.T().Fatal("insufficient amount")
	}

	// call BuildTransaction method
	builder, err := ton.NewTxBuilder(&xc_types.ChainConfig{Chain: xc_types.TON, Decimals: 9})
	suite.Require().NoError(err)

	args, err := xcbuilder.NewTransferArgs(
		xc_types.Address(from.String()),
		xc_types.Address(to.String()),
		amount,
		xcbuilder.WithMemo("test jetton"),
		xcbuilder.WithAsset(&xc_types.TokenAssetConfig{
			Decimals: 6,
			Contract: contractAddress,
		}),
	)
	suite.Require().NoError(err)

	input, err := suite.client.FetchTransferInput(ctx, args)
	suite.Require().NoError(err, "error FetchTransferInput")

	tx, err := builder.NewTransfer(args, input)
	suite.Require().NoError(err, "error BuildTransaction")

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().GreaterOrEqual(len(sighashes), 1)

	signature, err := suite.account1Signer.Sign(sighashes[0])
	suite.Require().NoError(err, "sign error")

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err, "add signature error")

	err = suite.client.BroadcastTx(ctx, tx)
	suite.Require().NoError(err, "SubmitTx failed")

	fmt.Printf("tx message hash: %v\n", tx.Hash())
}

func (suite *ClientTestSuite) TestFetchBalance() {
	contractAddress := xc_types.ContractAddress(USDTJettonMainnetAddress)

	ctx := context.Background()

	balance, err := suite.client.FetchBalance(ctx, xc_types.Address(suite.account1Address.String()))
	suite.Require().NoError(err)
	fmt.Printf("\n %s TON balance: %v\n", suite.account1Address.String(), balance)

	balance, err = suite.client.FetchBalanceForAsset(ctx, xc_types.Address(suite.account1Address.String()), contractAddress)
	suite.Require().NoError(err)
	fmt.Printf("\n %s jetton balance: %v\n", suite.account1Address.String(), balance)
}

func (suite *ClientTestSuite) TestFetchTonTxByHash() {
	ctx := context.Background()

	// hash := xc_types.TxHash("7f738ef013a970599563c761ac4047a06d9b160cc79e72cd30561902e83f2ecd")
	hash := xc_types.TxHash("f433109f09daf09d1f7d6e5ae1ff74adb88bcd9980f0158fb1d7e1426c087dc6") // jetton out txid
	// hash := xc_types.TxHash("09f977c21bb427b5c7c7bda414be625144b6c1ae187aa9bfacd6f58c3c617e3a") // out
	// hash := xc_types.TxHash("67982fb0800d56d9ae343dc0c9728b8a7f7d07ecbbdf2200eb3e9cdf50c9ba63") // in

	tx, err := suite.client.FetchTonTxByHash(ctx, hash)
	suite.Require().NoError(err)

	fmt.Printf("tx: %v\n", tx.Hash)

	legacyTx, err := suite.client.FetchLegacyTxInfo(ctx, hash)
	suite.Require().NoError(err)

	fmt.Printf("tx: %v\n", legacyTx.Amount)
}
