package tron

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	tronClient "github.com/fbsobreira/gotron-sdk/pkg/client"
	tronApi "github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/pkg/errors"

	"github.com/openweb3-io/crosschain/builder"
	"github.com/openweb3-io/crosschain/types"
	_types "github.com/openweb3-io/crosschain/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const TX_TIMEOUT = 2 * time.Hour

type Client struct {
	endpoint string
	chainId  *big.Int
	client   *tronClient.GrpcClient
}

func NewClient(
	endpoint string,
	chainId *big.Int,
) *Client {
	if endpoint == "" {
		endpoint = "grpc.trongrid.io:50051"
	}

	client := tronClient.NewGrpcClient(endpoint)

	if err := client.Start(grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		log.Fatalf("error dial rpc, err %v", err)
	}

	return &Client{
		endpoint: endpoint,
		chainId:  chainId,
		client:   client,
	}
}

func (client *Client) FetchTransferInput(ctx context.Context, args *builder.TransferArgs) (types.TxInput, error) {
	input := new(TxInput)
	input.From = args.From
	input.To = args.To
	input.ContractAddress = args.ContractAddress
	input.Amount = &args.Amount

	// Getting blockhash details from the CreateTransfer endpoint as TRON uses an unusual hashing algorithm (SHA2256SM3), so we can't do a minimal
	// retrieval and just get the blockheaders

	tx, err := client.client.Transfer(string(input.From), string(input.To), input.Amount.Int().Int64())
	// dummyTx, err := client.client.CreateTransaction(string(args.GetFrom()), string(args.GetTo()), 5)
	if err != nil {
		return nil, err
	}
	dummyTx := tx.Transaction

	input.RefBlockBytes = dummyTx.RawData.RefBlockBytes
	input.RefBlockHash = dummyTx.RawData.RefBlockHash
	// set timeout period
	input.Timestamp = time.Now().Unix()
	input.Expiration = time.Now().Add(TX_TIMEOUT).Unix()

	return input, nil
}

func (a *Client) FetchBalance(ctx context.Context, address types.Address) (*types.BigInt, error) {
	account, err := a.client.GetAccount(string(address))
	if err != nil {
		return nil, err
	}
	balance := types.NewBigIntFromInt64(account.Balance)
	return &balance, nil
}

func (a *Client) FetchBalanceForAsset(ctx context.Context, address types.Address, contractAddress *types.Address) (*types.BigInt, error) {
	balance, err := a.client.TRC20ContractBalance(string(address), string(*contractAddress))
	if err != nil {
		return nil, err
	}
	return (*types.BigInt)(balance), nil
}

func (a *Client) EstimateGas(ctx context.Context, tx types.Tx) (amount *types.BigInt, err error) {
	_tx := tx.(*Tx)

	input := _tx.input

	bandwithUsage := types.NewBigIntFromInt64(200)
	/*
		if txInput.Gas != nil {
			bandwithUsage = *txInput.Gas
		}
	*/

	params, err := a.client.Client.GetChainParameters(ctx, &tronApi.EmptyMessage{})
	if err != nil {
		return nil, errors.Wrap(err, "get chain params")
	}

	var transactionFee types.BigInt
	var energyFee types.BigInt

	for _, v := range params.ChainParameter {
		if v.Key == "getTransactionFee" {
			transactionFee = types.NewBigIntFromInt64(v.Value)
		}
		if v.Key == "getEnergyFee" {
			energyFee = types.NewBigIntFromInt64(v.Value)
		}
	}

	if input.ContractAddress == nil {
		//普通trx转账只需要带宽
		totalCost := (&transactionFee).Mul(&bandwithUsage)
		return &totalCost, nil
	} else {
		estimate, err := a.client.EstimateEnergy(
			string(input.From),
			string(*input.ContractAddress),
			"transfer(address,uint256)",
			fmt.Sprintf(`[{"address": "%s"},{"uint256": "%v"}]`, input.To, input.Amount),
			0, "", 0,
		)
		if err != nil {
			return nil, err
		}

		energyUsage := types.NewBigIntFromInt64(estimate.EnergyRequired)
		bandwidthCost := transactionFee.Mul(&bandwithUsage)
		energyCost := energyFee.Add(&energyUsage)
		totalCost := bandwidthCost.Add(&energyCost)

		return &totalCost, nil
	}

}

func (a *Client) SubmitTx(ctx context.Context, _tx _types.Tx) error {
	tx := _tx.(*Tx)
	if _, err := a.client.Broadcast(tx.tronTx); err != nil {
		return err
	}

	return nil
}
