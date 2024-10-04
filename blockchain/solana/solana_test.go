package solana_test

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"math/big"
	"testing"

	solana_sdk "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/openweb3-io/crosschain/blockchain/solana"
	"github.com/openweb3-io/crosschain/signer"
	"github.com/openweb3-io/crosschain/types"
)

var (
	//DBomk9vPzgLWpDBvvQpJUAB1aFz8EHsPq6xEuA1cGMcV
	senderPrivk      = "3PiA3WZuqGKv1E5aGWYfjsYVXZWLEJiUzGtFHZ8SNXUkLBbX9goGAHouEhTeFGUiBXVvRkfkHRga7XPENyJ7c3nq"
	senderPrivateKey solana_sdk.PrivateKey

	//8FLngQGnatEDQwNBV27yFxuWDhvQfriaCL56fx84TxoN
	recipientPrivk      = "2vLh8LUmwr9LVbFrJXKLcYcgMXAy52X6PHqZ9yhLvVfW1Fz3k1uJjheLcpUvum5oLYv8xZX5AnEXoMAEZMUMLyja"
	recipientPrivateKey solana_sdk.PrivateKey
)

func init() {
	senderPrivateKey = solana_sdk.MustPrivateKeyFromBase58(senderPrivk)
	fmt.Printf("sender address: %s \nprivate: %s\n", senderPrivateKey.PublicKey(), senderPrivateKey)

	// recipientPrivateKey = solana_sdk.NewWallet().PrivateKey
	recipientPrivateKey = solana_sdk.MustPrivateKeyFromBase58(recipientPrivk)
	fmt.Printf("recipient address: %s \nprivate: %s\n", recipientPrivateKey.PublicKey(), recipientPrivateKey)

}

var localSignerCreator = func(pub string) (signer.Signer, error) {
	key := ""
	if senderPrivateKey.PublicKey().String() == pub {
		key = senderPrivk
	} else {
		key = recipientPrivk
	}

	k := ed25519.PrivateKey(solana_sdk.MustPrivateKeyFromBase58(key))
	localSigner := solana.NewLocalSigner(k)
	return localSigner, nil
}

func TestTranfser(t *testing.T) {
	ctx := context.Background()

	//testnet
	solanaApi := solana.New(rpc.TestNet_RPC, big.NewInt(0))

	input := types.TransferArgs{
		FromAddress: senderPrivateKey.PublicKey().String(),
		ToAddress:   recipientPrivateKey.PublicKey().String(), // must exist
		Token:       "SOL",
		Amount:      big.NewInt(35),
	}

	aSigner, err := localSignerCreator(input.FromAddress)
	if err != nil {
		t.Fatal(err)
	}
	input.Signer = aSigner

	ret, err := solanaApi.BuildTransaction(ctx, &input)
	if err != nil {
		t.Fatal("BuildTransaction: ", err)
	}

	fee, err := solanaApi.EstimateGas(ctx, &input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("estimate SOL gas: %d\n", fee)

	if err := solanaApi.BroadcastSignedTx(ctx, ret); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("tx: %s\n", ret.Hash())
}

func TestSPLTranfser(t *testing.T) {
	ctx := context.Background()

	//testnet
	solanaApi := solana.New(rpc.TestNet_RPC, big.NewInt(0))

	contractAddress := "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr"

	// feePayer := senderPrivateKey.PublicKey().String()
	// feePayer := recipientPrivateKey.PublicKey().String()
	input := types.TransferArgs{
		FromAddress:     senderPrivateKey.PublicKey().String(),          //这里填写sol的主地址，转账时程序自动找到合约的关联账户地址
		ToAddress:       "AyqkhCrb8gt3PqiVMCshSy4to8wQcHzXtfCKbJ42qJLp", //这里写sol的主地址，自动会创建关联地址
		Token:           "USDC-Dev",
		ContractAddress: &contractAddress, //token address usdc-dev
		TokenDecimals:   6,
		// FeePayer:        &feePayer,
		Amount: big.NewInt(35),
	}

	aSigner, err := localSignerCreator(input.FromAddress)
	if err != nil {
		t.Fatal(err)
	}
	input.Signer = aSigner

	if input.FeePayer != nil {
		feeSigner, err := localSignerCreator(*input.FeePayer)
		if err != nil {
			t.Fatal(err)
		}
		input.FeePayerSigner = feeSigner
	}

	ret, err := solanaApi.BuildTransaction(ctx, &input)
	if err != nil {
		t.Fatal("BuildTransaction: ", err)
	}

	fee, err := solanaApi.EstimateGas(ctx, &input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("estimate SOL gas: %d\n", fee)

	if err := solanaApi.BroadcastSignedTx(ctx, ret); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("tx: %s\n", ret.Hash())
}

func TestSPLTranfserSetFeePayer(t *testing.T) {
	ctx := context.Background()

	//testnet
	solanaApi := solana.New(rpc.TestNet_RPC, big.NewInt(0))

	contractAddress := "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr"

	feePayer := recipientPrivateKey.PublicKey().String()
	amount := types.NewBigIntFromInt64(1)
	input := types.TransferArgs{
		From:            senderPrivateKey.PublicKey().String(),          //这里填写sol的主地址，转账时程序自动找到合约的关联账户地址
		To:              "AyqkhCrb8gt3PqiVMCshSy4to8wQcHzXtfCKbJ42qJLp", //这里写sol的主地址，自动会创建关联地址
		Token:           "USDC-Dev",
		ContractAddress: &contractAddress, //token address usdc-dev
		TokenDecimals:   6,
		FeePayer:        &feePayer,
		Amount:          amount,
	}

	aSigner, err := localSignerCreator(input.FromAddress)
	if err != nil {
		t.Fatal(err)
	}
	input.Signer = aSigner

	if input.FeePayer != nil {
		feeSigner, err := localSignerCreator(*input.FeePayer)
		if err != nil {
			t.Fatal(err)
		}
		input.FeePayerSigner = feeSigner
	}

	ret, err := solanaApi.BuildTransaction(ctx, &input)
	if err != nil {
		t.Fatal("BuildTransaction: ", err)
	}

	fee, err := solanaApi.EstimateGas(ctx, &input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("estimate SOL gas: %d\n", fee)

	if err := solanaApi.BroadcastSignedTx(ctx, ret); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("tx: %s\n", ret.Hash())
}

func TestGetBalance(t *testing.T) {
	solanaApi := solana.New(rpc.TestNet_RPC, big.NewInt(0))
	ctx := context.Background()

	contractAddress := "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr"

	out, err := solanaApi.GetBalance(ctx, senderPrivateKey.PublicKey().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("\n %s SOL balance: %v", senderPrivateKey.PublicKey().String(), out)

	out, err = solanaApi.GetBalance(ctx, senderPrivateKey.PublicKey().String(), &contractAddress)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("\n %s SPL token balance: %v", senderPrivateKey.PublicKey().String(), out)
}
