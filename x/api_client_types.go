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

type CreateOrderData struct {
	PricePerShare string            `json:"pricePerShare"`
	Strategy      string            `json:"strategy"`
	Order         SubmitSignedOrder `json:"order"`
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
type LimitOrderParams struct {
	TokenID        string
	PricePerShare  string // 人类可读价格，如 "0.850"
	QuantityShares string // 份数，如 "100"
	FeeRateBps     int
	IsNegRisk      bool
	IsYieldBearing bool
}
