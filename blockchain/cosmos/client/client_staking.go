package client

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/types/query"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/openweb3-io/crosschain/blockchain/cosmos/tx_input"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xclient "github.com/openweb3-io/crosschain/client"
	xc "github.com/openweb3-io/crosschain/types"
)

func (client *Client) FetchStakeBalance(ctx context.Context, args xclient.StakedBalanceArgs) ([]*xclient.StakedBalance, error) {
	q := stakingtypes.NewQueryClient(client.Ctx)
	delegations, err := q.DelegatorDelegations(ctx, &stakingtypes.QueryDelegatorDelegationsRequest{
		DelegatorAddr: string(args.GetFrom()),
		Pagination: &query.PageRequest{
			Limit: 1000,
		},
	})
	if err != nil {
		return nil, err
	}

	unbonding, err := q.DelegatorUnbondingDelegations(ctx, &stakingtypes.QueryDelegatorUnbondingDelegationsRequest{
		DelegatorAddr: string(args.GetFrom()),
		Pagination: &query.PageRequest{
			Limit: 1000,
		},
	})
	if err != nil {
		return nil, err
	}

	balances := []*xclient.StakedBalance{}
	for _, bal := range delegations.DelegationResponses {
		balances = append(balances, xclient.NewStakedBalance(
			xc.BigInt(*bal.Balance.Amount.BigInt()),
			xclient.Active,
			bal.Delegation.ValidatorAddress,
			"",
		))
	}
	for _, bal := range unbonding.UnbondingResponses {
		for _, entry := range bal.Entries {
			state := xclient.Deactivating
			if time.Since(entry.CompletionTime) > 0 {
				state = xclient.Inactive
			}
			amount := xc.BigInt(*entry.Balance.BigInt())
			balances = append(balances, xclient.NewStakedBalance(
				amount,
				state,
				bal.ValidatorAddress,
				"",
			))
		}
	}
	return balances, nil
}

func (client *Client) FetchStakingInput(ctx context.Context, args xcbuilder.StakeArgs) (xc.StakeTxInput, error) {
	asset, _ := args.GetAsset()
	baseTxInput, err := client.FetchBaseTxInput(ctx, args.GetFrom(), asset)
	if err != nil {
		return nil, err
	}
	return &tx_input.StakingInput{
		TxInput: *baseTxInput,
	}, nil
}

func (client *Client) FetchUnstakingInput(ctx context.Context, args xcbuilder.StakeArgs) (xc.UnstakeTxInput, error) {
	asset, _ := args.GetAsset()
	baseTxInput, err := client.FetchBaseTxInput(ctx, args.GetFrom(), asset)
	if err != nil {
		return nil, err
	}
	return &tx_input.UnstakingInput{
		TxInput: *baseTxInput,
	}, nil
}

func (client *Client) FetchWithdrawInput(ctx context.Context, args xcbuilder.StakeArgs) (xc.WithdrawTxInput, error) {
	asset, _ := args.GetAsset()
	baseTxInput, err := client.FetchBaseTxInput(ctx, args.GetFrom(), asset)
	if err != nil {
		return nil, err
	}
	return &tx_input.WithdrawInput{
		TxInput: *baseTxInput,
	}, nil
}
