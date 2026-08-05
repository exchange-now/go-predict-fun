package x

type GetMarketOrderbookResponse struct {
	Success bool `json:"success"`
	Data    Book `json:"data"`
}

type GetAuthMessageResponse struct {
	Success bool            `json:"success"`
	Data    AuthMessageData `json:"data"`
}

type AuthMessageData struct {
	Message string `json:"message"`
}

type PostAuthRequest struct {
	Signer    string `json:"signer"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
}

type PostAuthResponse struct {
	Success bool           `json:"success"`
	Data    PostAuthResult `json:"data"`
}

type PostAuthResult struct {
	Token string `json:"token"`
}

type CreateOrderRequest struct {
	Data CreateOrderData `json:"data"`
}

// CreateOrderData 对应 POST /v1/orders 的 data 字段。
// 文档: https://dev.predict.fun/createorderrequest-14037466d0
// 除 pricePerShare/strategy/order 外均为可选字段，未设置时由服务端取默认值。
type CreateOrderData struct {
	PricePerShare string `json:"pricePerShare"`
	Strategy      string `json:"strategy"`
	// SlippageBps 仅对市价单有意义。
	SlippageBps           string                `json:"slippageBps,omitempty"`
	IsFillOrKill          *bool                 `json:"isFillOrKill,omitempty"`
	IsPostOnly            *bool                 `json:"isPostOnly,omitempty"`
	ReservedBalancePolicy ReservedBalancePolicy `json:"reservedBalancePolicy,omitempty"`
	// IsMinAmountOut 表示 takerAmount 是签名的下限而非实际成交量，
	// 服务端按 makerAmount * 1e18 / priceInCurrency 推导实际份数。仅用于市价单。
	IsMinAmountOut      *bool               `json:"isMinAmountOut,omitempty"`
	SelfTradePrevention SelfTradePrevention `json:"selfTradePrevention,omitempty"`
	Order               SubmitSignedOrder   `json:"order"`
}

// SubmitSignedOrder 是 POST /v1/orders 所需的已签名订单体。
type SubmitSignedOrder struct {
	Order
	Hash      string `json:"hash"`
	Signature string `json:"signature"`
}

type CreateOrderResponse struct {
	Success bool                  `json:"success"`
	Data    CreateOrderResultData `json:"data"`
}

type CreateOrderResultData struct {
	Code      string `json:"code"`
	OrderID   string `json:"orderId"`
	OrderHash string `json:"orderHash"`
}

// LimitOrderParams 是高层限价买单入参。
// IsPostOnly/IsFillOrKill/ReservedBalancePolicy/SelfTradePrevention 为可选，
// 零值表示不下发该字段，由服务端取默认值。
type LimitOrderParams struct {
	TokenID        string
	PricePerShare  string // 人类可读价格，如 "0.850"
	QuantityShares string // 份数，如 "100"
	FeeRateBps     int
	IsNegRisk      bool
	IsYieldBearing bool
	// IsPostOnly 为 true 时只做 maker，若会立即成交则整单被拒。
	IsPostOnly *bool
	// IsFillOrKill 为 true 时要求立即全部成交，否则整单取消。与 IsPostOnly 互斥。
	IsFillOrKill          *bool
	ReservedBalancePolicy ReservedBalancePolicy
	SelfTradePrevention   SelfTradePrevention
}

// OrderListStatus 是 GET /v1/orders 列表 filter（官方仅 OPEN|FILLED）。
type OrderListStatus string

const (
	OrderListStatusOpen   OrderListStatus = "OPEN"
	OrderListStatusFilled OrderListStatus = "FILLED"
)

// ListOrdersParams 是 GET /v1/orders 查询参数。
type ListOrdersParams struct {
	Status OrderListStatus
	First  string  // 分页大小，如 "100"
	After  *string // cursor
}

// OrderRecord 是订单列表/单查返回的订单摘要。
type OrderRecord struct {
	ID             string `json:"id"`
	MarketID       int64  `json:"marketId"`
	Amount         string `json:"amount"`
	AmountFilled   string `json:"amountFilled"`
	Status         string `json:"status"`
	IsNegRisk      bool   `json:"isNegRisk"`
	IsYieldBearing bool   `json:"isYieldBearing"`
	Order          struct {
		Hash string `json:"hash"`
	} `json:"order"`
}

// ListOrdersResponse 是 GET /v1/orders 分页响应。
type ListOrdersResponse struct {
	Success bool          `json:"success"`
	Cursor  *string       `json:"cursor"`
	Data    []OrderRecord `json:"data"`
}

// GetOrderResponse 是 GET /v1/orders/{hash|id} 响应。
type GetOrderResponse struct {
	Success bool        `json:"success"`
	Data    OrderRecord `json:"data"`
}

// RemoveOrdersRequest 是 POST /v1/orders/remove 请求体。
type RemoveOrdersRequest struct {
	Data RemoveOrdersData `json:"data"`
}

type RemoveOrdersData struct {
	IDs []string `json:"ids"`
}

// RemoveOrdersResponse 是 POST /v1/orders/remove 响应。
type RemoveOrdersResponse struct {
	Success bool     `json:"success"`
	Removed []string `json:"removed"`
	Noop    []string `json:"noop"`
}

// ListPositionsParams 是 GET /v1/positions 查询参数。
type ListPositionsParams struct {
	First string
	After *string
}

// PositionRecord 是持仓列表条目。
type PositionRecord struct {
	Amount  string `json:"amount"`
	Outcome struct {
		OnChainID string `json:"onChainId"`
	} `json:"outcome"`
}

// ListPositionsResponse 是 GET /v1/positions 分页响应。
type ListPositionsResponse struct {
	Success bool             `json:"success"`
	Cursor  *string          `json:"cursor"`
	Data    []PositionRecord `json:"data"`
}
