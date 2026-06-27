package x

import (
	"encoding/json"
	"fmt"
)

// API base URLs from https://dev.predict.fun/
const (
	APIBaseURLProd    = "https://api.predict.fun"
	APIBaseURLTestnet = "https://api-testnet.predict.fun"
)

// MarketStatus filters markets by lifecycle status.
type MarketStatus string

const (
	MarketStatusOpen     MarketStatus = "OPEN"
	MarketStatusResolved MarketStatus = "RESOLVED"
)

// MarketVariant filters markets by variant type.
type MarketVariant string

const (
	MarketVariantDefault         MarketVariant = "DEFAULT"
	MarketVariantSportsMatch     MarketVariant = "SPORTS_MATCH"
	MarketVariantCryptoUpDown    MarketVariant = "CRYPTO_UP_DOWN"
	MarketVariantTweetCount      MarketVariant = "TWEET_COUNT"
	MarketVariantSportsTeamMatch MarketVariant = "SPORTS_TEAM_MATCH"
)

// MarketSort controls list ordering for GET /v1/markets.
type MarketSort string

const (
	MarketSortChance24hChangeAsc  MarketSort = "CHANCE_24H_CHANGE_ASC"
	MarketSortChance24hChangeDesc MarketSort = "CHANCE_24H_CHANGE_DESC"
	MarketSortVolume24hAsc        MarketSort = "VOLUME_24H_ASC"
	MarketSortVolume24hDesc       MarketSort = "VOLUME_24H_DESC"
	MarketSortVolume24hChangeAsc  MarketSort = "VOLUME_24H_CHANGE_ASC"
	MarketSortVolume24hChangeDesc MarketSort = "VOLUME_24H_CHANGE_DESC"
	MarketSortVolumeTotalAsc      MarketSort = "VOLUME_TOTAL_ASC"
	MarketSortVolumeTotalDesc     MarketSort = "VOLUME_TOTAL_DESC"
	MarketSortRewardRateAsc       MarketSort = "REWARD_RATE_ASC"
	MarketSortRewardRateDesc      MarketSort = "REWARD_RATE_DESC"
)

// GetMarketsParams are query parameters for GET /v1/markets.
// See https://dev.predict.fun/get-markets-25326905e0
type GetMarketsParams struct {
	First            *string // a string to be decoded into a number
	After            *string
	Status           *MarketStatus
	TagIDs           []string
	MarketVariant    *MarketVariant
	Sort             *MarketSort
	HasActiveRewards *bool
}

// GetMarketsResponse is the paginated response from GET /v1/markets.
type GetMarketsResponse struct {
	Success bool     `json:"success"`
	Cursor  *string  `json:"cursor"`
	Data    []Market `json:"data"`
}

// Market is full market data from the REST API.
type Market struct {
	ID                     int64         `json:"id"`
	ImageURL               string        `json:"imageUrl"`
	Title                  string        `json:"title"` // predict.fun 的 title 是子市场短标签（如 $150M、Discord、OpenAI、England)
	Question               string        `json:"question"`
	Description            string        `json:"description"`
	TradingStatus          string        `json:"tradingStatus"`
	Status                 string        `json:"status"`
	IsVisible              bool          `json:"isVisible"`
	IsNegRisk              bool          `json:"isNegRisk"`
	IsYieldBearing         bool          `json:"isYieldBearing"`
	FeeRateBps             int           `json:"feeRateBps"`
	Resolution             any           `json:"resolution,omitempty"`
	OracleQuestionID       string        `json:"oracleQuestionId"`
	ConditionID            string        `json:"conditionId"`
	ResolverAddress        string        `json:"resolverAddress"`
	Outcomes               []Outcome     `json:"outcomes"`
	QuestionIndex          *int          `json:"questionIndex,omitempty"`
	SpreadThreshold        json.Number   `json:"spreadThreshold"`
	ShareThreshold         int           `json:"shareThreshold"`
	IsBoosted              bool          `json:"isBoosted"`
	BoostStartsAt          *string       `json:"boostStartsAt,omitempty"`
	BoostEndsAt            *string       `json:"boostEndsAt,omitempty"`
	PolymarketConditionIDs []string      `json:"polymarketConditionIds"`
	KalshiMarketTicker     *string       `json:"kalshiMarketTicker,omitempty"`
	CategorySlug           string        `json:"categorySlug"`
	CreatedAt              string        `json:"createdAt"`
	DecimalPrecision       int           `json:"decimalPrecision"`
	MarketVariant          string        `json:"marketVariant"`
	VariantData            any           `json:"variantData,omitempty"`
	Team                   any           `json:"team,omitempty"`
	Rewards                MarketRewards `json:"rewards"`
	Stats                  any           `json:"stats,omitempty"`
}

// Outcome is a single market outcome from GET /v1/markets.
type Outcome struct {
	IndexSet    int         `json:"indexSet"`
	Name        string      `json:"name"`
	OnChainID   string      `json:"onChainId"`
	BestAsk     *PriceLevel `json:"bestAsk,omitempty"`
	BestBid     *PriceLevel `json:"bestBid,omitempty"`
	Status      any         `json:"status,omitempty"`
	Team        any         `json:"team,omitempty"`
	VariantData any         `json:"variantData,omitempty"`
}

// PriceLevel is a best bid/ask quote on an outcome. Price (in [0,1]) and Size
// are kept as json.Number to preserve the exact API representation.
type PriceLevel struct {
	Price json.Number `json:"price"`
	Size  json.Number `json:"size"`
}

// MarketRewards is the reward schedule for a market.
type MarketRewards struct {
	Current  *RewardPeriod  `json:"current"`
	Schedule []RewardPeriod `json:"schedule"`
}

// RewardPeriod is a single reward accrual window.
type RewardPeriod struct {
	HourlyRate int    `json:"hourlyRate"`
	StartsAt   string `json:"startsAt"`
	EndsAt     string `json:"endsAt"`
}

// APIError is returned when the REST API responds with success=false.
type APIError struct {
	Code      int    `json:"code"`
	ErrorCode string `json:"error"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Trace     string `json:"trace"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("predict.fun api error %d (%s): %s", e.Code, e.ErrorCode, e.Message)
	}
	return fmt.Sprintf("predict.fun api error %d (%s)", e.Code, e.ErrorCode)
}
