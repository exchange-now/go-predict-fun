package x

import (
	"context"
	"fmt"
	"net/url"
)

// ListOrders 拉取当前用户订单列表（GET /v1/orders）。
// status 官方仅支持 OPEN|FILLED；需已 Authenticate 或预先设置 BearerToken。
func (c *APIClient) ListOrders(ctx context.Context, params ListOrdersParams) (*ListOrdersResponse, error) {
	q := url.Values{}
	if params.First != "" {
		q.Set("first", params.First)
	}
	if params.Status != "" {
		q.Set("status", string(params.Status))
	}
	if params.After != nil && *params.After != "" {
		q.Set("after", *params.After)
	}
	var resp ListOrdersResponse
	if err := c.get(ctx, "/v1/orders", q, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetOrder 按 order hash 或数字 id 单查订单（GET /v1/orders/{hash|id}）。
// 官方路径参数为 hash；数字 id 常 404。
func (c *APIClient) GetOrder(ctx context.Context, idOrHash string) (*OrderRecord, error) {
	if idOrHash == "" {
		return nil, fmt.Errorf("predict.fun get order: empty idOrHash")
	}
	path := "/v1/orders/" + url.PathEscape(idOrHash)
	var resp GetOrderResponse
	if err := c.get(ctx, path, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data.ID == "" {
		return nil, fmt.Errorf("predict.fun get order %s: empty data", idOrHash)
	}
	return &resp.Data, nil
}

// RemoveOrders 从订单簿移除指定挂单（POST /v1/orders/remove）。
func (c *APIClient) RemoveOrders(ctx context.Context, ids []string) (*RemoveOrdersResponse, error) {
	if len(ids) == 0 {
		return &RemoveOrdersResponse{Success: true}, nil
	}
	var resp RemoveOrdersResponse
	if err := c.post(ctx, "/v1/orders/remove", RemoveOrdersRequest{
		Data: RemoveOrdersData{IDs: ids},
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListPositions 拉取当前用户持仓（GET /v1/positions）。
func (c *APIClient) ListPositions(ctx context.Context, params ListPositionsParams) (*ListPositionsResponse, error) {
	q := url.Values{}
	if params.First != "" {
		q.Set("first", params.First)
	}
	if params.After != nil && *params.After != "" {
		q.Set("after", *params.After)
	}
	var resp ListPositionsResponse
	if err := c.get(ctx, "/v1/positions", q, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
