package evm_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/openweb3-io/crosschain/blockchain/evm"
	"github.com/openweb3-io/crosschain/signer"
	_types "github.com/openweb3-io/crosschain/types"
)

var (
	endpoint = "https://sepolia.infura.io/v3/79f87ec10fba42d7a85aaa69406c6f96"
	chainId  = big.NewInt(11155111)
)

func init() {

}

var localSignerCreator = func(key string) (signer.Signer, error) {
	pk := "8e812436a0e3323166e1f0e8ba79e19e217b2c4a53c970d4cca0cfb1078979df"

	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		return nil, err
	}

	priv := crypto.ToECDSAUnsafe(pkBytes)
	signer := evm.NewLocalSigner(priv)

	fmt.Printf("address: %s \n", crypto.PubkeyToAddress(priv.PublicKey))
	return signer, nil
}

func TestTranfser(t *testing.T) {

	ctx := context.Background()

	//testnet Holesky 17000
	//testnet sepolia 11155111
	evmApi := evm.New(endpoint, chainId)

	input := _types.TransferArgs{
		From:   "0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8",
		To:     "0x388C818CA8B9251b393131C08a736A67ccB19297",
		Amount: big.NewInt(3000),
		Gas:    big.NewInt(21000),
	}

	input.Signer, _ = localSignerCreator(input.FromAddress)

	msg, err := evmApi.BuildTransaction(ctx, &input)
	if err != nil {
		t.Fatal(err)
	}

	if err := evmApi.BroadcastSignedTx(ctx, msg); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("output: %x\n", msg.Hash())
}

func TestTranfserERC20(t *testing.T) {
	fromAddress := "0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8"

	ctx := context.Background()

	signer, err := localSignerCreator(fromAddress)
	if err != nil {
		t.Fatal(err)
	}

	//testnet Holesky 17000
	//testnet sepolia 11155111
	evmApi := evm.New(endpoint, chainId)

	contractAddress := "0x779877A7B0D9E8603169DdbD7836e478b4624789"
	input := _types.TransferArgs{
		FromAddress:     fromAddress,
		ToAddress:       "0x388C818CA8B9251b393131C08a736A67ccB19297",
		ContractAddress: &contractAddress, //LINK
		TokenDecimals:   18,
		Amount:          big.NewInt(6000),
		Gas:             big.NewInt(43000),
		Signer:          signer,
	}

	input.Signer, _ = localSignerCreator(input.FromAddress)

	msg, err := evmApi.BuildTransaction(ctx, &input)
	if err != nil {
		t.Fatal(err)
	}

	if err := evmApi.BroadcastSignedTx(ctx, msg); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("output: %x\n", msg.Hash())
}

func TestGetBalance(t *testing.T) {
	evmApi := evm.New(endpoint, chainId)
	address := "0x50B0c2B3bcAd53Eb45B57C4e5dF8a9890d002Cc8"
	contractAddress := "0x779877A7B0D9E8603169DdbD7836e478b4624789"

	ctx := context.Background()
	out, err := evmApi.GetBalance(ctx, address, nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("\n %s eth balance: %v", address, out)

	out, err = evmApi.GetBalance(ctx, address, &contractAddress)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("\n %s contract balance: %v", address, out)

}
