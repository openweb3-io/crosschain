# Crosschain

Crosschain's main design principle is to isolate functionality into separate Client, Signer and TxBuilder interfaces.  
In addition to providing unified interfaces, it allows blockchains to be safely used in secure contexts.

## Example usage

First install the `xc` utility which will quickly demonstrate usage of the library.

```bash
go install -v ./cmd/xc/...
```

Manually interact with blockchains

Usage:
  xc [command]

Available Commands:
  address     Derive an address from the PRIVATE_KEY environment variable.
  balance     Check balance of an asset.  Reported as big integer, not accounting for any decimals.
  chains      List information on all supported chains.
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  staking     Staking commands
  transfer    Create and broadcast a new transaction transferring funds. The amount should be a decimal amount.
  tx-info     Check an existing transaction on chain.
  tx-input    Check inputs for a new transaction.

Flags:
      --chain string      Chain to use. Required.
      --config string     Path to config.yaml configuration file.
  -h, --help              help for xc
      --not-mainnet       Do not use mainnets, instead use a test or dev network.
      --provider string   Provider to use for chain client.  Only valid for BTC chains.
      --rpc string        RPC url to use. Optional.
  -v, --verbose count     Set verbosity.
```

### Generate or import a wallet

Set `PRIVATE_KEY` env and confirm you address is correct on the target chain you want to use.

```bash
export PRIVATE_KEY=...
xc address --chain SOL
```

### Send a transfer

```bash
xc transfer <destination-address> 0.1 -v --chain SOL
```

Add `--contract` for token transfers.

```bash
xc transfer <destination-address> 0.1 -v --chain SOL --contract EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v --decimals 6
```

Add `--rpc` to use your own RPC node or use a devnet or testnet network.

```bash
xc transfer <destination-address> 0.1 -v --chain SOL --rpc "https://api.devnet.solana.com"
```

### Stake an asset

Stake 0.1 SOL on mainnet.

```
xc staking stake --amount 0.1 --chain SOL --rpc https://api.mainnet-beta.solana.com --validator he1iusunGwqrNtafDtLdhsUQDFvo13z9sUa36PauBtk
```

### Download a transaction

Transactions are represented in a universal format across different chains.

```bash
xc tx-info --chain BTC b5734126a7b9f5a3a94491c7297959b74099c5c88d2f5f34ea3cb432abdf9c5e
```

Download another transaction from a difference chain.

```bash
xc tx-info --chain SOL 2NNSwe5ZCHx1SuYfgqy1pyWxDCfEcge3H4Eak1KyGCctjJictYtkQ4FFRH7CMJHM1W55FnyBmtKrxdZzkkThkjVL
```

### Lookup a balance

Get ether balance (in wei).

```bash
xc balance 0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5 --chain ETH
```

Add `--contract` to see a token balance.

```bash
xc balance 0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5 --chain ETH --contract 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48
```

### Lookup transaction input

This looks up all of the necessary inputs needed to serialize a transaction on a given blockchain. The values for the "input"
depends on each chain. Normally this is providing account sequences, gas fees, or unspent outputs.

```bash
xc tx-input <from-address> --chain ETH
```

```bash
xc tx-input 0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5 --chain ETH
```

## Features

### Blockchains

- [x] Aptos
- [x] Bitcoin
- [x] Bitcoin derived: Bitcoin Cash, Dogecoin
- [x] Bittensor
- [x] Cosmos
- [x] Cosmos derived: Terra, Injective, XPLA, ...
- [x] Ethereum
- [x] EVMs: Polygon, Binance Smart Chain, ...
- [ ] Filecoin
- [x] Polkadot
- [x] Solana
- [x] Sui
- [x] TON
- [x] Tron
- [ ] XRP

### Assets

- [x] Native assets
- [x] Tokens
- [x] Staked assets
- [ ] NFTs
- [ ] Liquidity pools

### Operations

- [x] Balances (native asset, tokens)
- [x] Transfers (native transfers, token transfers)
- [x] Transaction reporting
- [ ] Wraps/unwraps: ETH, SOL (partial support)
- [x] Staking/unstaking

### Devnet nodes

You can spin up your own devnet nodes + universal faucet API for testing.

Example on EVM:

```
# build and run container
cd chain/evm/node && docker build -t devnet-evm .
devnet-evm
docker run --name devnet-evm -p 10000:10000 -p 10001:10001 devnet-evm
```