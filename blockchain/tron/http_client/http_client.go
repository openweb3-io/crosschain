package httpclient

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
)

// Implement basic tron client that use's TRON's http api.
// This API is exposed on many public endpoints and is supported by private RPC providers.

// Bytes marshals/unmarshals as a JSON string with NO 0x prefix.
type Bytes []byte

var _ json.Unmarshaler = &Bytes{}

func (b *Bytes) UnmarshalJSON(inputBz []byte) error {
	var err error
	input := string(inputBz)
	input = strings.TrimPrefix(input, "\"")
	input = strings.TrimSuffix(input, "\"")
	input = strings.TrimPrefix(input, "0x")
	*b, err = hex.DecodeString(input)
	return err
}

type Resource string

const (
	ResourceBandwidth = Resource("BANDWIDTH")
	ResourceEnergy    = Resource("ENERGY")
)

type Client struct {
	baseUrl *url.URL
	client  *http.Client
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Error   string `json:"Error"`
}

type ContractParameter[T any] struct {
	Value   T      `json:"value"`
	TypeUrl string `json:"type_url"`
}

type ContractData[T any] struct {
	Parameter ContractParameter[T] `json:"parameter"`
	Type      string               `json:"type"`
}

type Receipt struct {
	NetFee uint64 `json:"net_fee"`
}

type TransactionRawData[T any] struct {
	Contract          []ContractData[T] `json:"contract"`
	RefBlockBytes     Bytes             `json:"ref_block_bytes"`
	RefBlockHashBytes Bytes             `json:"ref_block_hash"`
	Expiration        uint64            `json:"expiration"`
	FeeLimit          uint64            `json:"fee_limit"`
	Timestamp         uint64            `json:"timestamp"`
}

type CreateTransactionResponse struct {
	Error
	RawData    TransactionRawData[map[string]any] `json:"raw_data"`
	RawDataHex Bytes                              `json:"raw_data_hex"`
}

type GetTransactionIDResponse struct {
	Error
	RawData    TransactionRawData[map[string]any] `json:"raw_data"`
	RawDataHex Bytes                              `json:"raw_data_hex"`
	TxID       Bytes                              `json:"txID"`
	Signature  []Bytes                            `json:"signature"`
}

type GetTransactionInfoById struct {
	Error
	Id              Bytes    `json:"id"`
	Fee             uint64   `json:"fee"`
	BlockNumber     uint64   `json:"blockNumber"`
	BlockTimeStamp  uint64   `json:"blockTimeStamp"`
	ContractResult  []string `json:"contractResult"`
	Receipt         Receipt  `json:"receipt"`
	ContractAddress string   `json:"contract_address"`

	Logs                 []*Log                 `json:"log"`
	InternalTransactions []*InternalTransaction `json:"internal_transactions"`
}

type Log struct {
	Address Bytes   `json:"address"`
	Topics  []Bytes `json:"topics"`
	Data    Bytes   `json:"data"`
}
type InternalTransaction struct {
	Hash              Bytes `json:"hash"`
	CallerAddress     Bytes `json:"caller_address"`
	TransferToAddress Bytes `json:"transferTo_address"`
	Note              Bytes `json:"note"`
}
type BlockHeaderRawData struct {
	Number    uint64 `json:"number"`
	Version   uint64 `json:"version"`
	Timestamp uint64 `json:"timestamp"`
	// other fields...
}

type BlockHeader struct {
	RawData          BlockHeaderRawData `json:"raw_data"`
	WitnessSignature Bytes              `json:"witness_signature"`
}

type BlockResponse struct {
	Error
	BlockHeader BlockHeader `json:"block_header"`
	BlockId     string      `json:"blockID"`
}

type TriggerConstantContractResponse struct {
	Error          `json:"result"`
	ConstantResult []Bytes `json:"constant_result"`
}

type EstimateEnergyResponse struct {
	Error `json:"result"`
	// "result: {"result: true},\
	EnergyRequired int64 `json:"energy_required"`
}

type AccountResource struct {
	DelegatedFrozenBalanceForEnergy           int64 `json:"delegated_frozen_balance_for_energy"`
	AcquiredDelegatedFrozenBalanceForEnergy   int64 `json:"acquired_delegated_frozen_balance_for_energy"`
	DelegatedFrozenV2BalanceForEnergy         int64 `json:"delegated_frozenV2_balance_for_energy"`
	AcquiredDelegatedFrozenV2BalanceForEnergy int64 `json:"acquired_delegated_frozenV2_balance_for_energy"`
	EnergyWindowSize                          int64 `json:"energy_window_size"`
	EnergyWindowOptimized                     bool  `json:"energy_window_optimized"`
	EnergyUsage                               int64 `json:"energy_usage"`
	LatestConsumeTimeForEnergy                int64 `json:"latest_consume_time_for_energy"`
}

type FrozenV2 struct {
	Type   string `json:"type"`
	Amount int64  `json:"amount"`
}

type UnfrozenV2 struct {
	Type               string `json:"type"`
	UnfreezeAmount     int64  `json:"unfreeze_amount"`
	UnfreezeExpireTime int64  `json:"unfreeze_expire_time"`
}

type FreeAssetNetUsageV2 struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

type Vote struct {
	VoteAddress string `json:"vote_address"`
	VoteCount   int64  `json:"vote_count"`
}

type PermissionKey struct {
	Address string `json:"address"`
	Weight  int64  `json:"weight"`
}

type Permission struct {
	Type           string          `json:"type"`
	Id             int64           `json:"id"`
	PermissionName string          `json:"permission_name"`
	Threshold      int64           `json:"threshold"`
	Operations     string          `json:"operations"`
	Keys           []PermissionKey `json:"keys"`
}

type AssetV2 struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

type GetAccountResponse struct {
	Error
	Address                                      string                `json:"address"`
	CreateTime                                   int64                 `json:"create_time"`
	Balance                                      int64                 `json:"balance"`
	DelegatedFrozenBalanceForBandwidth           int64                 `json:"delegated_frozen_balance_for_bandwidth"`
	AcquiredDelegatedFrozenBalanceForBandwidth   int64                 `json:"acquired_delegated_frozen_balance_for_bandwidth"`
	AccountResource                              AccountResource       `json:"account_resource"`
	DelegatedFrozenV2BalanceForBandwidth         int64                 `json:"delegated_frozenV2_balance_for_bandwidth"`
	AcquiredDelegatedFrozenV2BalanceForBandwidth int64                 `json:"acquired_delegated_frozenV2_balance_for_bandwidth"`
	FrozenV2                                     []FrozenV2            `json:"frozenV2"`
	UnfrozenV2                                   []UnfrozenV2          `json:"unfrozenV2"`
	NetWindowSize                                int64                 `json:"net_window_size"`
	NetWindowOptimized                           bool                  `json:"net_window_optimized"`
	FreeAssetNetUsageV2                          []FreeAssetNetUsageV2 `json:"free_asset_net_usageV2"`
	Votes                                        []Vote                `json:"votes"`
	LatestOprationTime                           int64                 `json:"latest_opration_time"`
	LatestConsumeTime                            int64                 `json:"latest_consume_time"`
	LatestConsumeFreeTime                        int64                 `json:"latest_consume_free_time"`
	IsWitness                                    bool                  `json:"is_witness"`
	Allowance                                    int64                 `json:"allowance"`
	LatestWithdrawTime                           int64                 `json:"latest_withdraw_time"`
	OwnerPermission                              Permission            `json:"owner_permission"`
	WitnessPermission                            Permission            `json:"witness_permission"`
	ActivePermission                             []Permission          `json:"active_permission"`
	AssetV2                                      []AssetV2             `json:"assetV2"`
	AssetOptimized                               bool                  `json:"asset_optimized"`
}

type GetAccountResourceResponse struct {
	Error
	FreeNetUsed          int64            `json:"freeNetUsed,omitempty"`
	FreeNetLimit         int64            `json:"freeNetLimit,omitempty"`
	NetUsed              int64            `json:"NetUsed,omitempty"`
	NetLimit             int64            `json:"NetLimit,omitempty"`
	TotalNetLimit        int64            `json:"TotalNetLimit,omitempty"`
	TotalNetWeight       int64            `json:"TotalNetWeight,omitempty"`
	TotalTronPowerWeight int64            `json:"totalTronPowerWeight,omitempty"`
	TronPowerLimit       int64            `json:"tronPowerLimit,omitempty"`
	TronPowerUsed        int64            `json:"tronPowerUsed,omitempty"`
	EnergyUsed           int64            `json:"EnergyUsed,omitempty"`
	EnergyLimit          int64            `json:"EnergyLimit,omitempty"`
	TotalEnergyLimit     int64            `json:"TotalEnergyLimit,omitempty"`
	TotalEnergyWeight    int64            `json:"TotalEnergyWeight,omitempty"`
	AssetNetUsed         []map[string]any `json:"assetNetUsed,omitempty"`
	AssetNetLimit        []map[string]any `json:"assetNetLimit,omitempty"`
}

type GetCanWithdrawUnfreezeAmountResponse struct {
	Error
	Amount int64 `json:"amount,omitempty"`
}

type GetChainParametersResponse struct {
	Error
	ChainParameter []ChainParameter
}

type ChainParameter struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

type TransactionResponse[T any] struct {
	Error
	Visible    bool                  `json:"visible"`
	TxID       string                `json:"txID"`
	RawData    TransactionRawData[T] `json:"raw_data"`
	RawDataHex string                `json:"raw_data_hex"`
}

type FreezeBalanceV2Request struct {
	OwnerAddress  string `json:"owner_address"`
	Resource      string `json:"resource"`
	FrozenBalance int64  `json:"frozen_balance"`
	PermissionId  *int32 `json:"Permission_id,omitempty"`
	Visible       *bool  `json:"visible,omitempty"`
}

type FreezeBalanceV2ContractParameterValue struct {
	Resource      string `json:"resource"`
	FrozenBalance int64  `json:"frozen_balance"` // official document using 'int'
	OwnerAddress  string `json:"owner_address"`
}

type UnfreezeBalanceV2Request struct {
	OwnerAddress    string `json:"owner_address"`
	Resource        string `json:"resource"`
	UnfreezeBalance int64  `json:"unfreeze_balance"`
	PermissionId    *int32 `json:"Permission_id,omitempty"`
	Visible         *bool  `json:"visible,omitempty"`
}

type UnfreezeBalanceV2ContractParameterValue struct {
	Resource      string `json:"resource"`
	FrozenBalance int64  `json:"frozen_balance"` // official document using 'int'
	OwnerAddress  string `json:"owner_address"`
}

type WithdrawExpireUnfreezeRequest struct {
	OwnerAddress string `json:"owner_address"`
	PermissionId *int32 `json:"Permission_id,omitempty"`
	Visible      *bool  `json:"visible,omitempty"`
}

type WithdrawExpireUnfreezeContractParameterValue struct {
	OwnerAddress string `json:"owner_address"`
}

type DelegateResourceRequest struct {
	OwnerAddress    string   `json:"owner_address"`
	ReceiverAddress string   `json:"receiver_address"`
	Resource        Resource `json:"resource"`
	Balance         int64    `json:"balance"`
	Lock            *bool    `json:"lock,omitempty"`
	LockPeriod      *int64   `json:"lock_period,omitempty"`
	PermissionId    *int32   `json:"Permission_id,omitempty"`
	Visible         *bool    `json:"visible,omitempty"`
}

type DelegateResourceContractParameterValue struct {
	OwnerAddress    string   `json:"owner_address"`
	ReceiverAddress string   `json:"receiver_address"`
	Resource        Resource `json:"resource"`
	Balance         int64    `json:"balance"`
	Lock            *bool    `json:"lock,omitempty"`
	LockPeriod      *int64   `json:"lock_period,omitempty"`
}

type UnDelegateResourceRequest struct {
	OwnerAddress    string   `json:"owner_address"`
	ReceiverAddress string   `json:"receiver_address"`
	Resource        Resource `json:"resource"`
	Balance         int64    `json:"balance"`
	PermissionId    *int32   `json:"Permission_id,omitempty"`
	Visible         *bool    `json:"visible,omitempty"`
}

type UnDelegateResourceContractParameterValue struct {
	OwnerAddress    string   `json:"owner_address"`
	ReceiverAddress string   `json:"receiver_address"`
	Resource        Resource `json:"resource"`
	Balance         int64    `json:"balance"`
}

func NewHttpClient(baseUrl string) (*Client, error) {
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	baseUrl = strings.TrimSuffix(baseUrl, "/wallet")
	baseUrl = strings.TrimSuffix(baseUrl, "/jsonrpc")
	u, err := url.Parse(baseUrl)

	// may want to pass externally to support additional
	// headers or something.
	client := &http.Client{}

	return &Client{
		baseUrl: u,
		client:  client,
	}, err
}

func parseResponse[T any](res *http.Response, dest T) (T, error) {
	bz, err := io.ReadAll(res.Body)
	if err != nil {
		return dest, err
	}
	err = json.Unmarshal(bz, dest)

	// b, _ := json.MarshalIndent(bz, "", "\t")
	// fmt.Println(string(b))

	// decoder := json.NewDecoder(res.Body)
	// err := decoder.Decode(dest)
	return dest, err
}

func checkError(res Error) error {
	if len(res.Code) > 0 && len(res.Message) > 0 {
		return fmt.Errorf("%s: %s", res.Code, res.Message)
	}
	if len(res.Error) > 0 {
		return fmt.Errorf("%s", res.Error)
	}
	return nil
}

func postRequest(ctx context.Context, url string, body any) (*http.Request, error) {
	bz, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bz))
	if err != nil {
		return req, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func getRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return req, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (c *Client) Url(path string) string {
	return c.baseUrl.JoinPath(path).String()
}

func (c *Client) CreateTransaction(ctx context.Context, from string, to string, amount int) (*CreateTransactionResponse, error) {
	req, err := postRequest(ctx, c.Url("wallet/createtransaction"), map[string]interface{}{
		"owner_address": from,
		"to_address":    to,
		"amount":        amount,
		"visible":       true,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &CreateTransactionResponse{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}
	// if parsed.

	return parsed, nil
}

func (c *Client) BroadcastHex(ctx context.Context, txHex string) (*CreateTransactionResponse, error) {
	req, err := postRequest(ctx, c.Url("wallet/broadcasthex"), map[string]interface{}{
		"transaction": txHex,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &CreateTransactionResponse{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}

	return parsed, nil
}

func (c *Client) GetTransactionByID(ctx context.Context, txHash string) (*GetTransactionIDResponse, error) {
	req, err := postRequest(ctx, c.Url("wallet/gettransactionbyid"), map[string]interface{}{
		"value":   txHash,
		"visible": true,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &GetTransactionIDResponse{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}
	if len(parsed.TxID) == 0 {
		return parsed, fmt.Errorf("could not find tx: %s", txHash)
	}

	return parsed, nil
}

func (c *Client) GetTransactionInfoByID(ctx context.Context, txHash string) (*GetTransactionInfoById, error) {
	req, err := postRequest(ctx, c.Url("wallet/gettransactioninfobyid"), map[string]interface{}{
		"value": txHash,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &GetTransactionInfoById{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}
	if len(parsed.Id) == 0 {
		return parsed, fmt.Errorf("could not find tx info: %s", txHash)
	}

	return parsed, nil
}

func (c *Client) GetBlockByNum(ctx context.Context, num uint64) (*BlockResponse, error) {
	req, err := postRequest(ctx, c.Url("wallet/getblockbynum"), map[string]interface{}{
		"num": num,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &BlockResponse{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}
	if len(parsed.BlockId) == 0 {
		return parsed, fmt.Errorf("could not find block by num: %d", num)
	}

	return parsed, nil
}

func (c *Client) EstimateEnergy(
	ctx context.Context,
	ownerAddress string,
	contract string,
	funcSelector string,
	jsonString string,
	amount int64,
) (*EstimateEnergyResponse, error) {
	param, err := abi.LoadFromJSON(jsonString)
	if err != nil {
		return nil, err
	}

	dataBytes, err := abi.Pack(funcSelector, param)
	if err != nil {
		return nil, err
	}

	req, err := postRequest(ctx, c.Url("wallet/estimateenergy"), map[string]any{
		"owner_address":    ownerAddress,
		"contract_address": contract,
		// "function_selector": funcSelector,
		// "parameter":  jsonString,
		"data":       hex.EncodeToString(dataBytes),
		"call_value": amount,
		"visible":    true,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &EstimateEnergyResponse{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}

	return parsed, nil
}

func (c *Client) TriggerConstantContracts(ctx context.Context, ownerAddress string, contract string, funcSelector string, param string) (*TriggerConstantContractResponse, error) {
	req, err := postRequest(ctx, c.Url("wallet/triggerconstantcontract"), map[string]interface{}{
		"owner_address":     ownerAddress,
		"contract_address":  contract,
		"constant":          true,
		"function_selector": funcSelector,
		"parameter":         param,
		"visible":           true,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &TriggerConstantContractResponse{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}

	return parsed, nil
}

func (c *Client) ReadTrc20Balance(ctx context.Context, fromAddress string, contract string) (*big.Int, error) {
	addrB, err := common.DecodeCheck(fromAddress)
	if err != nil {
		return &big.Int{}, err
	}
	addrHex := hex.EncodeToString(addrB)
	contractB, err := common.DecodeCheck(contract)
	if err != nil {
		return &big.Int{}, err
	}
	req := "0000000000000000000000000000000000000000000000000000000000000000"[len(addrHex):] + addrHex
	ownerAddress := hex.EncodeToString(addrB)
	contractHex := hex.EncodeToString(contractB)
	_, _ = ownerAddress, contractHex

	response, err := c.TriggerConstantContracts(ctx, fromAddress, contract, "balanceOf(address)", req)
	if err != nil {
		return &big.Int{}, err
	}

	value := big.NewInt(0)
	if len(response.ConstantResult) == 0 {
		return value, fmt.Errorf("no balance returned reading balance for: %s", contract)
	}
	return value.SetBytes(response.ConstantResult[0]), nil
}

func (c *Client) GetAccount(ctx context.Context, address string) (*GetAccountResponse, error) {
	req, err := postRequest(ctx, c.Url("wallet/getaccount"), map[string]interface{}{
		"address": address,
		"visible": true,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &GetAccountResponse{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}
	if len(parsed.Address) == 0 {
		return parsed, fmt.Errorf("could not find account: %s", address)
	}

	return parsed, nil
}

func (c *Client) GetChainParameters(ctx context.Context) (*GetChainParametersResponse, error) {
	req, err := getRequest(ctx, c.Url("wallet/getchainparameters"))

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &GetChainParametersResponse{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}

	return parsed, nil
}

func (c *Client) GetAccountResource(ctx context.Context, address string) (*GetAccountResourceResponse, error) {
	req, err := postRequest(ctx, c.Url("wallet/getaccountresource"), map[string]interface{}{
		"address": address,
		"visible": true,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &GetAccountResourceResponse{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}

	return parsed, nil
}

func (c *Client) GetCanWithdrawUnfreezeAmount(ctx context.Context, address string, timestamp time.Time) (*GetCanWithdrawUnfreezeAmountResponse, error) {
	req, err := postRequest(ctx, c.Url("wallet/getcanwithdrawunfreezeamount"), map[string]interface{}{
		"owner_address": address,
		"timestamp":     timestamp.UnixMilli(),
		"visible":       true,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &GetCanWithdrawUnfreezeAmountResponse{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}

	return parsed, nil
}

func (c *Client) FreezeBalanceV2(ctx context.Context, address string, resource Resource, amount *big.Int) (*TransactionResponse[FreezeBalanceV2ContractParameterValue], error) {
	visible := true
	req, err := postRequest(ctx, c.Url("wallet/freezebalancev2"), &FreezeBalanceV2Request{
		OwnerAddress:  address,
		Resource:      string(resource),
		FrozenBalance: amount.Int64(),
		Visible:       &visible,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &TransactionResponse[FreezeBalanceV2ContractParameterValue]{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}

	return parsed, nil
}

func (c *Client) UnfreezeBalanceV2(ctx context.Context, address string, resource Resource, amount *big.Int) (*TransactionResponse[UnfreezeBalanceV2ContractParameterValue], error) {
	visible := true
	req, err := postRequest(ctx, c.Url("wallet/unfreezebalancev2"), &UnfreezeBalanceV2Request{
		OwnerAddress:    address,
		Resource:        string(resource),
		UnfreezeBalance: amount.Int64(),
		Visible:         &visible,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &TransactionResponse[UnfreezeBalanceV2ContractParameterValue]{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}

	return parsed, nil
}

func (c *Client) WithdrawExpireUnfreeze(ctx context.Context, address string) (*TransactionResponse[WithdrawExpireUnfreezeContractParameterValue], error) {
	visible := true
	req, err := postRequest(ctx, c.Url("wallet/withdrawexpireunfreeze"), &WithdrawExpireUnfreezeRequest{
		OwnerAddress: address,
		Visible:      &visible,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &TransactionResponse[WithdrawExpireUnfreezeContractParameterValue]{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}

	return parsed, nil
}

func (c *Client) DelegateResource(ctx context.Context, ownerAddress, receiverAddress string, resource Resource, amount *big.Int) (*TransactionResponse[DelegateResourceContractParameterValue], error) {
	visible := true
	req, err := postRequest(ctx, c.Url("wallet/delegateresource"), &DelegateResourceRequest{
		OwnerAddress:    ownerAddress,
		ReceiverAddress: receiverAddress,
		Resource:        resource,
		Balance:         amount.Int64(),
		Visible:         &visible,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &TransactionResponse[DelegateResourceContractParameterValue]{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}

	return parsed, nil
}

func (c *Client) UnDelegateResource(ctx context.Context, ownerAddress, receiverAddress string, resource Resource, amount *big.Int) (*TransactionResponse[UnDelegateResourceContractParameterValue], error) {
	visible := true
	req, err := postRequest(ctx, c.Url("wallet/undelegateresource"), &UnDelegateResourceRequest{
		OwnerAddress:    ownerAddress,
		ReceiverAddress: receiverAddress,
		Resource:        resource,
		Balance:         amount.Int64(),
		Visible:         &visible,
	})

	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) {
		if body != nil {
			_ = body.Close()
		}
	}(resp.Body)

	parsed, err := parseResponse(resp, &TransactionResponse[UnDelegateResourceContractParameterValue]{})
	if err != nil {
		return nil, err
	}
	if err = checkError(parsed.Error); err != nil {
		return nil, err
	}

	return parsed, nil
}
