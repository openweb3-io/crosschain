package ton_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"math/big"
	"testing"

	"github.com/croutondefi/stonfi-go"
	"github.com/openweb3-io/crosschain/blockchain/ton"
	"github.com/openweb3-io/crosschain/blockchain/ton/wallet"
	"github.com/openweb3-io/crosschain/signer"
	"github.com/openweb3-io/crosschain/types"
	"github.com/test-go/testify/suite"
	"github.com/tonkeeper/tonapi-go"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	_ton "github.com/xssnick/tonutils-go/ton"
)

const (
	USDTJettonMainnetAddress = "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs"
	USDTJettonTestnetAddress = "kQD0GKBM8ZbryVk2aESmzfU6b9b_8era_IkvBSELujFZPsyy"
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
	liteApiClient   *_ton.APIClient
}

// get test coin from this telegram bot: https://web.telegram.org/k/#@testgiver_ton_bot
// testnet blockchain browser: https://testnet.tonscan.org/
// get test coin from https://faucet.tonxapi.com/
// exchange ton to jUSDT bridge.ton.org

const (
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

	client := liteclient.NewConnectionPool()
	// url := "https://ton-blockchain.github.io/testnet-global.config.json"
	url := "https://api.tontech.io/ton/wallet-mainnet.autoconf.json"
	err = client.AddConnectionsFromConfigUrl(context.Background(), url)
	suite.Require().NoError(err)
	suite.liteApiClient = _ton.NewAPIClient(client)

	tonApi, err := tonapi.New(tonapi.WithToken("AEXRCJJGQBXCFWQAAAAD3RYTVUWCXT5JW6YN2QU7LHXMKPMOXHFB75P4JSD52AVOVQWPGNY"))
	// tonApi, err := tonapi.NewClient(tonapi.TestnetTonApiURL)
	suite.Require().NoError(err)

	suite.client = ton.NewClient(tonApi, suite.liteApiClient)
}

func (suite *ClientTestSuite) TearDownTest() {
}

func (suite *ClientTestSuite) Test_Tranfser() {
	ctx := context.Background()

	from, err := wallet.AddressFromPubKey(suite.account1PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)
	to, err := wallet.AddressFromPubKey(suite.account2PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)

	readableAmount, err := types.NewAmountHumanReadableFromStr("0.01")
	suite.Require().NoError(err)
	blockchainAmount := readableAmount.ToBlockchain(9)

	input, err := suite.client.FetchTransferInput(ctx, &types.TransferArgs{
		From:   from.String(),
		To:     to.String(),
		Amount: blockchainAmount,
		Memo:   "test transfer ton",
		// Token:  "TON",
	})
	suite.Require().NoError(err)

	builder := ton.NewTxBuilder(suite.liteApiClient)
	tx, err := builder.BuildTransaction(ctx, input)
	suite.Require().NoError(err)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().GreaterOrEqual(len(sighashes), 1)

	signature, err := suite.account1Signer.Sign(ctx, sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err)

	err = suite.client.BroadcastSignedTx(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("tx hash(base64): %v\n", base64.StdEncoding.EncodeToString(tx.Hash()))
}

func (suite *ClientTestSuite) aTest_EstimateGas() {
	ctx := context.Background()

	from, err := wallet.AddressFromPubKey(suite.account1PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)
	to, err := wallet.AddressFromPubKey(suite.account2PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)

	builder := ton.NewTxBuilder(suite.liteApiClient)

	input, err := suite.client.FetchTransferInput(ctx, &types.TransferArgs{
		From:   from.String(),
		To:     to.String(),
		Amount: types.NewBigIntFromUint64(10000000),
		Memo:   "test",
		// Token:  "TON",
	})
	suite.Require().NoError(err)

	tx, err := builder.BuildTransaction(ctx, input)
	suite.Require().NoError(err)

	amount, err := suite.client.EstimateGas(ctx, tx)
	suite.Require().NoError(err)

	fmt.Printf("amount: %v\n", amount)
}

/**
 * work for mainnet
 */
func (suite *ClientTestSuite) aTest_SwapFromTonToUSDT() {
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

	routerRevV1 := stonfi.NewRouterRevisionV1(suite.liteApiClient, routerAddr)
	router := stonfi.NewRouter(suite.liteApiClient, routerAddr, routerRevV1)

	rm, _ := types.NewAmountHumanReadableFromStr("0.1")
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

	w, err := wallet.FromAddress(ctx, suite.liteApiClient, suite.account1Address, wallet.V4R2)
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

	tx := ton.NewTx(w.Address(), cellBuilder, nil)

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().GreaterOrEqual(len(sighashes), 1)

	signature, err := suite.account1Signer.Sign(ctx, sighashes[0])
	suite.Require().NoError(err)

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err)

	err = suite.client.BroadcastSignedTx(ctx, tx)
	suite.Require().NoError(err)
}

func (suite *ClientTestSuite) Test_TransferJetton() {
	ctx := context.Background()

	contractAddress := USDTJettonMainnetAddress

	from, err := wallet.AddressFromPubKey(suite.account1PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)
	to, err := wallet.AddressFromPubKey(suite.account2PubKey, wallet.V4R2, wallet.DefaultSubwallet)
	suite.Require().NoError(err)

	readableAmount, err := types.NewAmountHumanReadableFromStr("0.01")
	suite.Require().NoError(err)
	amount := readableAmount.ToBlockchain(6)
	suite.Require().NoError(err)

	jettonBalance, err := suite.client.GetBalanceForAsset(ctx, types.Address(from.String()), types.Address(contractAddress))
	suite.Require().NoError(err, "error GetBalanceForAsset")

	if jettonBalance.Cmp(&amount) < 0 {
		suite.T().Fatal("insufficient amount")
	}

	// call BuildTransaction method
	builder := ton.NewTxBuilder(suite.liteApiClient)
	input, err := suite.client.FetchTransferInput(ctx, &types.TransferArgs{
		ContractAddress: &contractAddress,
		From:            from.String(),
		To:              to.String(),
		Amount:          amount,
		Memo:            "test jetton",
		// Token:           "USDT",
		TokenDecimals: 6,
	})
	suite.Require().NoError(err, "error FetchTransferInput")

	tx, err := builder.BuildTransaction(ctx, input)
	suite.Require().NoError(err, "error BuildTransaction")

	sighashes, err := tx.Sighashes()
	suite.Require().NoError(err)
	suite.Require().GreaterOrEqual(len(sighashes), 1)

	signature, err := suite.account1Signer.Sign(ctx, sighashes[0])
	suite.Require().NoError(err, "sign error")

	err = tx.AddSignatures(signature)
	suite.Require().NoError(err, "add signature error")

	err = suite.client.BroadcastSignedTx(ctx, tx)
	suite.Require().NoError(err, "BroadcastSignedTx failed")
}

func (suite *ClientTestSuite) TestGetBalance() {
	contract := "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs"

	ctx := context.Background()

	balance, err := suite.client.GetBalance(ctx, types.Address(suite.account1Address.String()))
	suite.Require().NoError(err)
	fmt.Printf("\n %s TON balance: %v\n", suite.account1Address.String(), balance)

	balance, err = suite.client.GetBalanceForAsset(ctx, types.Address(suite.account1Address.String()), types.Address(contract))
	suite.Require().NoError(err)
	fmt.Printf("\n %s jetton balance: %v\n", suite.account1Address.String(), balance)
}
