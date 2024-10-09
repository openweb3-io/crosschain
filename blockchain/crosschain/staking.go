package crosschain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/openweb3-io/crosschain/blockchain/crosschain/types"
	xcbuilder "github.com/openweb3-io/crosschain/builder"
	xcclient "github.com/openweb3-io/crosschain/client"
	"github.com/openweb3-io/crosschain/factory/blockchains"
	xc "github.com/openweb3-io/crosschain/types"
)

// Crosschain client assumes that all staking providers might need a post-unstake request-exit.
// By doing this, it can defer to the remote to decide if this is needed for the given provider.
var _ xcclient.ManualUnstakingClient = &Client{}

func (client *Client) FetchStakeBalance(ctx context.Context, args xcclient.StakedBalanceArgs) ([]*xcclient.StakedBalance, error) {
	chain := client.cfg.Chain

	params := url.Values{}
	if validator, ok := args.GetValidator(); ok {
		params.Set("validator", validator)
	}
	if account, ok := args.GetAccount(); ok {
		params.Set("account", account)
	}
	if client.StakingProvider != "" {
		params.Set("provider", string(client.StakingProvider))
	}

	apiURL := fmt.Sprintf("%s/v1/chains/%s/addresses/%s/staking?%s", client.URL, chain, args.GetFrom(), params.Encode())
	res, err := client.ApiCallWithUrl(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	r := []*xcclient.StakedBalance{}
	err = json.Unmarshal(res, &r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (client *Client) FetchStakingInput(ctx context.Context, args xcbuilder.StakeArgs) (xc.StakeTxInput, error) {
	chain := client.cfg.Chain

	var req = &types.StakingInputReq{
		From:     string(args.GetFrom()),
		Balance:  args.GetAmount().String(),
		Provider: client.StakingProvider,
	}
	req.Validator, _ = args.GetValidator()
	req.Account, _ = args.GetStakeAccount()

	apiURL := fmt.Sprintf("%s/v1/chains/%s/stakes", client.URL, chain)
	res, err := client.ApiCallWithUrl(ctx, "POST", apiURL, req)
	if err != nil {
		return nil, err
	}

	r := types.TxInputRes{}
	err = json.Unmarshal(res, &r)
	if err != nil {
		return nil, err
	}
	return blockchains.UnmarshalStakingInput([]byte(r.TxInput))
}
func (client *Client) FetchUnstakingInput(ctx context.Context, args xcbuilder.StakeArgs) (xc.UnstakeTxInput, error) {
	chain := client.cfg.Chain

	var req = &types.StakingInputReq{
		From:     string(args.GetFrom()),
		Balance:  args.GetAmount().String(),
		Provider: client.StakingProvider,
	}
	req.Validator, _ = args.GetValidator()
	req.Account, _ = args.GetStakeAccount()

	apiURL := fmt.Sprintf("%s/v1/chains/%s/unstakes", client.URL, chain)
	res, err := client.ApiCallWithUrl(ctx, "POST", apiURL, req)
	if err != nil {
		return nil, err
	}

	r := types.TxInputRes{}
	err = json.Unmarshal(res, &r)
	if err != nil {
		return nil, err
	}
	return blockchains.UnmarshalUnstakingInput([]byte(r.TxInput))
}
func (client *Client) FetchWithdrawInput(ctx context.Context, args xcbuilder.StakeArgs) (xc.WithdrawTxInput, error) {
	chain := client.cfg.Chain

	var req = &types.StakingInputReq{
		From:     string(args.GetFrom()),
		Balance:  args.GetAmount().String(),
		Provider: client.StakingProvider,
	}
	req.Validator, _ = args.GetValidator()
	req.Account, _ = args.GetStakeAccount()

	apiURL := fmt.Sprintf("%s/v1/chains/%s/withdraws", client.URL, chain)
	res, err := client.ApiCallWithUrl(ctx, "POST", apiURL, req)
	if err != nil {
		return nil, err
	}

	r := types.TxInputRes{}
	err = json.Unmarshal(res, &r)
	if err != nil {
		return nil, err
	}
	return blockchains.UnmarshalWithdrawingInput([]byte(r.TxInput))
}

// func (client *Client) FetchWithdrawInput(ctx context.Context, args xcbuilder.StakeArgs) (xc.WithdrawTxInput, error) {
func (cli *Client) CompleteManualUnstaking(ctx context.Context, unstake *xcclient.Unstake) error {
	chain := cli.cfg.Chain

	var req = &types.StakingInputReq{
		Provider:  cli.StakingProvider,
		From:      unstake.Address,
		Balance:   unstake.Balance.String(),
		Validator: unstake.Validator,
		Account:   unstake.Account,
	}

	apiURL := fmt.Sprintf("%s/v1/chains/%s/request-exit", cli.URL, chain)
	res, err := cli.ApiCallWithUrl(ctx, "POST", apiURL, req)
	if err != nil {
		return err
	}
	_ = res
	return nil
}