package x

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// LogLevel controls SDK logging verbosity.
type LogLevel string

const (
	LogLevelError LogLevel = "ERROR"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelDebug LogLevel = "DEBUG"
)

// Currency is the collateral token symbol supported by BalanceOf.
type Currency string

const CurrencyUSDT Currency = "USDT"

// OrderStrategy is LIMIT or MARKET.
type OrderStrategy string

const (
	OrderStrategyMarket OrderStrategy = "MARKET"
	OrderStrategyLimit  OrderStrategy = "LIMIT"
)

// SelfTradePrevention decides which side is cancelled when an order would match
// against another order from the same account.
type SelfTradePrevention string

const (
	SelfTradePreventionCancelMaker SelfTradePrevention = "CANCEL_MAKER"
	SelfTradePreventionCancelTaker SelfTradePrevention = "CANCEL_TAKER"
	SelfTradePreventionCancelBoth  SelfTradePrevention = "CANCEL_BOTH"
)

// ReservedBalancePolicy controls how balance already reserved by resting orders
// is treated when the new order is checked.
type ReservedBalancePolicy string

const (
	ReservedBalancePolicyRejectMarketOrder        ReservedBalancePolicy = "REJECT_MARKET_ORDER"
	ReservedBalancePolicySkipReservedBalanceCheck ReservedBalancePolicy = "SKIP_RESERVED_BALANCE_CHECKS"
)

// DepthLevel is [price, quantity] from the order book API. Values are kept as
// json.Number so the exact wire representation is preserved and converted to
// wei without ever passing through a float64.
type DepthLevel [2]json.Number

// Book is the order book returned by GET /markets/{id}/orderbook.
type Book struct {
	MarketID          int          `json:"marketId"`
	UpdateTimestampMs int64        `json:"updateTimestampMs"`
	Asks              []DepthLevel `json:"asks"`
	Bids              []DepthLevel `json:"bids"`
}

// MarketHelperInput calculates market order amounts by share quantity.
type MarketHelperInput struct {
	Side           Side
	QuantityWei    *big.Int
	SlippageBps    *big.Int
	IsMinAmountOut bool
}

// MarketHelperValueInput calculates market BUY amounts by USDT value.
type MarketHelperValueInput struct {
	Side           Side
	ValueWei       *big.Int
	SlippageBps    *big.Int
	IsMinAmountOut bool
}

// LimitHelperInput calculates limit order amounts.
type LimitHelperInput struct {
	Side             Side
	PricePerShareWei *big.Int
	QuantityWei      *big.Int
}

// OrderAmounts holds computed maker/taker amounts for an order.
type OrderAmounts struct {
	LastPrice      *big.Int
	PricePerShare  *big.Int
	MakerAmount    *big.Int
	TakerAmount    *big.Int
	Amount         *big.Int
	SlippageBps    *big.Int
	IsMinAmountOut bool
}

// ProcessedBookAmounts is an internal order-book aggregation result.
type ProcessedBookAmounts struct {
	QuantityWei  *big.Int
	PriceWei     *big.Int
	LastPriceWei *big.Int
}

// BuildOrderInput is the input for BuildOrder.
type BuildOrderInput struct {
	Side          Side
	TokenID       string
	MakerAmount   *big.Int
	TakerAmount   *big.Int
	FeeRateBps    *big.Int
	Signer        string
	Nonce         *big.Int
	Salt          *big.Int
	Maker         string
	Taker         string
	SignatureType SignatureType
	ExpiresAt     *time.Time
}

// Order is the on-chain / API order struct.
type Order struct {
	Salt          string        `json:"salt"`
	Maker         string        `json:"maker"`
	Signer        string        `json:"signer"`
	Taker         string        `json:"taker"`
	TokenID       string        `json:"tokenId"`
	MakerAmount   string        `json:"makerAmount"`
	TakerAmount   string        `json:"takerAmount"`
	Expiration    string        `json:"expiration"`
	Nonce         string        `json:"nonce"`
	FeeRateBps    string        `json:"feeRateBps"`
	Side          Side          `json:"side"`
	SignatureType SignatureType `json:"signatureType"`
}

// OrderWithHash includes the EIP-712 order hash.
type OrderWithHash struct {
	Order
	Hash string `json:"hash"`
}

// SignedOrder is an order with an EIP-712 signature.
type SignedOrder struct {
	Order
	Hash      string `json:"hash,omitempty"`
	Signature string `json:"signature"`
}

// EIP712Parameter is a single field in an EIP-712 type definition.
type EIP712Parameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// EIP712Types maps type names to field lists.
type EIP712Types map[string][]EIP712Parameter

// EIP712Object is a generic EIP-712 struct value.
type EIP712Object map[string]any

// EIP712TypedData is the full typed-data payload for signing.
type EIP712TypedData struct {
	Types       EIP712Types  `json:"types"`
	Domain      EIP712Object `json:"domain"`
	Message     EIP712Object `json:"message"`
	PrimaryType string       `json:"primaryType"`
}

// OrderBuilderOptions configures OrderBuilder construction.
type OrderBuilderOptions struct {
	Addresses      *Addresses
	Precision      int
	PredictAccount string
	GenerateSalt   func() string
	LogLevel       LogLevel
	RPCURL         string
}

// CancelOrdersOptions configures CancelOrders.
type CancelOrdersOptions struct {
	IsYieldBearing bool
	IsNegRisk      bool
	WithValidation *bool
}

// RedeemPositionsOptions configures RedeemPositions.
type RedeemPositionsOptions struct {
	ConditionID    string
	IndexSet       int
	Amount         *big.Int
	IsNegRisk      bool
	IsYieldBearing bool
}

// MergePositionsOptions configures MergePositions.
type MergePositionsOptions struct {
	ConditionID    string
	Amount         *big.Int
	IsNegRisk      bool
	IsYieldBearing bool
}

// SplitPositionsOptions configures SplitPositions.
type SplitPositionsOptions struct {
	ConditionID    string
	Amount         *big.Int
	IsNegRisk      bool
	IsYieldBearing bool
}

// SetApprovalsResult is returned by SetApprovals.
type SetApprovalsResult struct {
	Success      bool
	Transactions []TransactionResult
}

// TransactionSuccess indicates a successful on-chain transaction.
type TransactionSuccess struct {
	Success bool
	TxHash  common.Hash
}

// TransactionFail indicates a failed on-chain transaction.
type TransactionFail struct {
	Success bool
	Cause   error
	TxHash  *common.Hash
}

// TransactionResult is the outcome of a contract transaction.
type TransactionResult interface {
	isTransactionResult()
	Ok() bool
}

func (TransactionSuccess) isTransactionResult() {}
func (TransactionFail) isTransactionResult()    {}

// Ok reports whether the transaction succeeded.
func (r TransactionSuccess) Ok() bool { return r.Success }
func (r TransactionFail) Ok() bool    { return r.Success }

// ExchangeContract identifies which CTF exchange contract to use.
type ExchangeContract int

const (
	ExchangeCTF ExchangeContract = iota
	ExchangeNegRiskCTF
	ExchangeYieldBearingCTF
	ExchangeYieldBearingNegRiskCTF
)

// CtfContract identifies which conditional-tokens contract to use.
type CtfContract int

const (
	CtfStandard CtfContract = iota
	CtfNegRisk
	CtfYieldBearing
	CtfYieldBearingNegRisk
)
