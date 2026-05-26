package x

type GetMarketOrderbookResponse struct {
	Success bool `json:"success"`
	Data    Book `json:"data"`
}
