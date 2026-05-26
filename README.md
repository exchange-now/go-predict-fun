# go-predict-fun

Golang SDK for [predict.fun](https://predict.fun/) — mirrors the official [@predictdotfun/sdk](https://github.com/PredictDotFun/sdk) TypeScript SDK.

Official SDKs today are TypeScript and Python only; this project gives Go developers the same order-building, EIP-712 signing, and on-chain helpers.

## Install

```bash
go get github.com/exchange-now/go-predict-fun/x
```

## Quick start — limit order (EOA)

```go
package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/exchange-now/go-predict-fun/predictfun"
)

func main() {
	ctx := context.Background()
	privateKey := os.Getenv("WALLET_PRIVATE_KEY")

	ob, err := predictfun.NewOrderBuilderWithSigner(ctx, predictfun.ChainIDBnbMainnet, privateKey, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ob.Close()

	// 1. Set approvals (once per wallet)
	result, err := ob.SetApprovals(ctx)
	if err != nil || !result.Success {
		log.Fatal("approvals failed")
	}

	// 2. Compute amounts (price 0.5 USDT/share, 10 shares)
	amounts, err := ob.GetLimitOrderAmounts(predictfun.LimitHelperInput{
		Side:             predictfun.SideBuy,
		PricePerShareWei: predictfun.ParseWeiMust("500000000000000000"), // 0.5e18
		QuantityWei:      predictfun.ParseWeiMust("10000000000000000000"),
	})
	if err != nil {
		log.Fatal(err)
	}

	// 3. Build order (fetch feeRateBps from GET /markets)
	order, err := ob.BuildOrder(predictfun.OrderStrategyLimit, predictfun.BuildOrderInput{
		Side:        predictfun.SideBuy,
		TokenID:     "YOUR_OUTCOME_TOKEN_ID",
		MakerAmount: amounts.MakerAmount,
		TakerAmount: amounts.TakerAmount,
		FeeRateBps:  big.NewInt(0),
	})
	if err != nil {
		log.Fatal(err)
	}

	// 4. EIP-712 sign (set isNegRisk / isYieldBearing from market metadata)
	typed := ob.BuildTypedData(order, false, false)
	signed, err := ob.SignTypedDataOrder(typed)
	if err != nil {
		log.Fatal(err)
	}

	hash, _ := ob.BuildTypedDataHash(typed)
	fmt.Println("order hash:", hash.Hex())
	fmt.Println("signature:", signed.Signature)
}
```

## Market order from order book

Fetch the book from `GET /markets/{marketId}/orderbook`, then:

```go
book := predictfun.Book{
	Asks: []predictfun.DepthLevel{{0.5, 3}, {0.88, 4}},
	Bids: []predictfun.DepthLevel{{0.9, 2}, {0.5, 3}},
}

amounts, err := ob.GetMarketOrderAmounts(predictfun.MarketHelperInput{
	Side:        predictfun.SideBuy,
	QuantityWei: predictfun.ParseWeiMust("5000000000000000000"),
}, book)
```

For BUY-by-USDT-budget, use `MarketHelperValueInput` with `ValueWei`.

## Predict Account (smart wallet)

Pass `PredictAccount` (deposit address from [account settings](https://predict.fun/account/settings)) and sign with your Privy exported key:

```go
ob, err := predictfun.NewOrderBuilderWithSigner(ctx, predictfun.ChainIDBnbMainnet, privyKey, &predictfun.OrderBuilderOptions{
	PredictAccount: "0xYourPredictAccount",
})
```

`BuildOrder` sets `maker` / `signer` to the Predict account; `SignTypedDataOrder` uses Kernel EIP-712 wrapping.

## API parity with TypeScript SDK

| TypeScript | Go |
|------------|-----|
| `OrderBuilder.make(chainId, signer?)` | `NewOrderBuilder` / `NewOrderBuilderWithSigner` |
| `getLimitOrderAmounts` | `GetLimitOrderAmounts` |
| `getMarketOrderAmounts` | `GetMarketOrderAmounts` |
| `buildOrder` | `BuildOrder` |
| `buildTypedData` | `BuildTypedData` |
| `signTypedDataOrder` | `SignTypedDataOrder` |
| `buildTypedDataHash` | `BuildTypedDataHash` |
| `setApprovals` | `SetApprovals` |
| `cancelOrders` | `CancelOrders` |
| `redeemPositions` / `mergePositions` / `splitPositions` | same names |
| `balanceOf` | `BalanceOf` |
| `ChainId`, `Side`, `SignatureType` | `ChainID`, `Side`, `SignatureType` |

Contract addresses and EIP-712 constants match [`Constants.ts`](https://github.com/PredictDotFun/sdk/blob/main/src/Constants.ts) on BNB mainnet (56) and testnet (97).

## REST API — Get markets

Implements [`GET /v1/markets`](https://dev.predict.fun/get-markets-25326905e0).

```go
client := predictfun.NewAPIClient(predictfun.APIClientOptions{
	BaseURL: predictfun.APIBaseURLTestnet, // testnet: no API key required
	// BaseURL: x.APIBaseURLProd,
	// APIKey:  os.Getenv("PREDICT_API_KEY"), // required on mainnet
})

first := 10
status := predictfun.MarketStatusOpen
resp, err := client.GetMarkets(ctx, predictfun.GetMarketsParams{
	First:  &first,
	Status: &status,
})
if err != nil {
	log.Fatal(err)
}
for _, m := range resp.Data {
	fmt.Println(m.ID, m.Title, m.FeeRateBps, m.IsNegRisk)
	for _, o := range m.Outcomes {
		fmt.Println(" ", o.Name, "tokenId:", o.OnChainID)
	}
}
// Paginate with resp.Cursor:
// params.After = resp.Cursor
```

Mainnet base URL: `https://api.predict.fun` (API key via `x-api-key` header).  
Testnet: `https://api-testnet.predict.fun` (no key required per [docs](https://dev.predict.fun/)).

## Development

```bash
go test ./...
```

## References

- [Predict TypeScript SDK](https://github.com/PredictDotFun/sdk)
- [Predict Python SDK](https://github.com/PredictDotFun/sdk-python)
- [Developer docs](https://docs.predict.fun/)
