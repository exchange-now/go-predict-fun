package x

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/exchange-now/go-predict-fun/x/internal"
)

// CreateLimitOrder 构建、签名并提交 GTC 限价买单。
func (c *APIClient) CreateLimitOrder(ctx context.Context, ob *OrderBuilder, params LimitOrderParams) (*CreateOrderResultData, error) {
	priceWei := internal.ParseEther(json.Number(params.PricePerShare))
	qtyWei := internal.ParseEther(json.Number(params.QuantityShares))

	amounts, err := ob.GetLimitOrderAmounts(LimitHelperInput{
		Side:             SideBuy,
		PricePerShareWei: priceWei,
		QuantityWei:      qtyWei,
	})
	if err != nil {
		return nil, fmt.Errorf("get limit order amounts: %w", err)
	}

	order, err := ob.BuildOrder(OrderStrategyLimit, BuildOrderInput{
		Side:        SideBuy,
		TokenID:     params.TokenID,
		MakerAmount: amounts.MakerAmount,
		TakerAmount: amounts.TakerAmount,
		FeeRateBps:  big.NewInt(int64(params.FeeRateBps)),
	})
	if err != nil {
		return nil, fmt.Errorf("build order: %w", err)
	}

	typedData := ob.BuildTypedData(order, params.IsNegRisk, params.IsYieldBearing)
	signed, err := ob.SignTypedDataOrder(typedData)
	if err != nil {
		return nil, fmt.Errorf("sign order: %w", err)
	}
	hash, err := ob.BuildTypedDataHash(typedData)
	if err != nil {
		return nil, fmt.Errorf("hash order: %w", err)
	}

	req := CreateOrderRequest{
		Data: CreateOrderData{
			PricePerShare: amounts.PricePerShare.String(),
			Strategy:      string(OrderStrategyLimit),
			IsPostOnly:    params.IsPostOnly,
			Order: SubmitSignedOrder{
				Order:     signed.Order,
				Hash:      hash.Hex(),
				Signature: signed.Signature,
			},
		},
	}

	result, err := c.CreateOrder(ctx, req)
	if err != nil {
		// 401 时尝试重新认证后重试一次
		if apiErr, ok := err.(*APIError); ok && apiErr.Code == 401 {
			if authErr := c.Authenticate(ctx, ob); authErr != nil {
				return nil, fmt.Errorf("re-auth after 401: %w (original: %w)", authErr, err)
			}
			return c.CreateOrder(ctx, req)
		}
		return nil, err
	}
	return result, nil
}
