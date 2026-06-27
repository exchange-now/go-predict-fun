package x

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/exchange-now/go-predict-fun/x/contracts"
	"github.com/exchange-now/go-predict-fun/x/internal"
)

// GenerateOrderSalt returns a random numeric salt string.
func GenerateOrderSalt() string {
	return fmt.Sprintf("%d", rand.Intn(MaxSalt))
}

// OrderBuilder helps build, sign, and submit Predict.fun orders.
type OrderBuilder struct {
	chainID        ChainID
	precision      *big.Int
	addresses      Addresses
	generateSalt   func() string
	logger         *Logger
	privateKey     *ecdsa.PrivateKey
	signerAddress  common.Address
	predictAccount *common.Address
	contracts      *contractSet
	client         *ethclient.Client
}

type contractSet struct {
	yieldBearingCTFExchange              *contracts.BoundContract
	yieldBearingNegRiskCTFExchange       *contracts.BoundContract
	yieldBearingNegRiskAdapter           *contracts.BoundContract
	yieldBearingConditionalTokens        *contracts.BoundContract
	yieldBearingNegRiskConditionalTokens *contracts.BoundContract

	ctfExchange              *contracts.BoundContract
	negRiskCTFExchange       *contracts.BoundContract
	negRiskAdapter           *contracts.BoundContract
	conditionalTokens        *contracts.BoundContract
	negRiskConditionalTokens *contracts.BoundContract

	usdt           *contracts.BoundContract
	kernel         *contracts.BoundContract
	ecdsaValidator *contracts.BoundContract
}

// NewOrderBuilder creates an OrderBuilder without on-chain capabilities (order math and EIP-712 only).
func NewOrderBuilder(chainID ChainID, opts *OrderBuilderOptions) *OrderBuilder {
	return newOrderBuilder(chainID, nil, opts)
}

// NewOrderBuilderWithSigner creates an OrderBuilder with a private key for signing and contract calls.
func NewOrderBuilderWithSigner(ctx context.Context, chainID ChainID, privateKeyHex string, opts *OrderBuilderOptions) (*OrderBuilder, error) {
	key, err := crypto.HexToECDSA(trimHexPrefix(privateKeyHex))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	ob := newOrderBuilder(chainID, key, opts)

	rpcURL := RPCURLByChainID[chainID]
	if opts != nil && opts.RPCURL != "" {
		rpcURL = opts.RPCURL
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("connect rpc: %w", err)
	}
	ob.client = client
	ob.contracts = ob.buildContracts()

	if ob.predictAccount != nil {
		owner, err := ob.readECDSAValidatorOwner(ctx, *ob.predictAccount)
		if err != nil {
			return nil, err
		}
		if owner != ob.signerAddress {
			return nil, ErrInvalidSigner
		}
	}
	return ob, nil
}

func newOrderBuilder(chainID ChainID, privateKey *ecdsa.PrivateKey, opts *OrderBuilderOptions) *OrderBuilder {
	addresses := AddressesByChainID[chainID]
	precision := big.NewInt(1e18)
	generateSalt := GenerateOrderSalt
	logLevel := LogLevelWarn

	if opts != nil {
		if opts.Addresses != nil {
			addresses = *opts.Addresses
		}
		if opts.Precision > 0 {
			precision = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(opts.Precision)), nil)
		}
		if opts.GenerateSalt != nil {
			generateSalt = opts.GenerateSalt
		}
		if opts.LogLevel != "" {
			logLevel = opts.LogLevel
		}
	}

	ob := &OrderBuilder{
		chainID:      chainID,
		precision:    precision,
		addresses:    addresses,
		generateSalt: generateSalt,
		logger:       newLogger(logLevel),
		privateKey:   privateKey,
	}
	if privateKey != nil {
		ob.signerAddress = crypto.PubkeyToAddress(privateKey.PublicKey)
	}
	if opts != nil && opts.PredictAccount != "" {
		addr := common.HexToAddress(opts.PredictAccount)
		ob.predictAccount = &addr
	}
	return ob
}

func (ob *OrderBuilder) buildContracts() *contractSet {
	addr := func(s string) common.Address { return common.HexToAddress(s) }
	return &contractSet{
		yieldBearingCTFExchange:              contracts.NewBoundContract(addr(ob.addresses.YieldBearingCTFExchange), contracts.CTFExchangeABI()),
		yieldBearingNegRiskCTFExchange:       contracts.NewBoundContract(addr(ob.addresses.YieldBearingNegRiskCTFExchange), contracts.NegRiskCTFExchangeABI()),
		yieldBearingNegRiskAdapter:           contracts.NewBoundContract(addr(ob.addresses.YieldBearingNegRiskAdapter), contracts.NegRiskAdapterABI()),
		yieldBearingConditionalTokens:        contracts.NewBoundContract(addr(ob.addresses.YieldBearingConditionalTokens), contracts.YieldBearingConditionalTokensABI()),
		yieldBearingNegRiskConditionalTokens: contracts.NewBoundContract(addr(ob.addresses.YieldBearingNegRiskConditionalTokens), contracts.ConditionalTokensABI()),
		ctfExchange:                          contracts.NewBoundContract(addr(ob.addresses.CTFExchange), contracts.CTFExchangeABI()),
		negRiskCTFExchange:                   contracts.NewBoundContract(addr(ob.addresses.NegRiskCTFExchange), contracts.NegRiskCTFExchangeABI()),
		negRiskAdapter:                       contracts.NewBoundContract(addr(ob.addresses.NegRiskAdapter), contracts.NegRiskAdapterABI()),
		conditionalTokens:                    contracts.NewBoundContract(addr(ob.addresses.ConditionalTokens), contracts.ConditionalTokensABI()),
		negRiskConditionalTokens:             contracts.NewBoundContract(addr(ob.addresses.NegRiskConditionalTokens), contracts.ConditionalTokensABI()),
		usdt:                                 contracts.NewBoundContract(addr(ob.addresses.USDT), contracts.ERC20ABI()),
		kernel:                               contracts.NewBoundContract(addr(ob.predictAccountOrKernel()), contracts.KernelABI()),
		ecdsaValidator:                       contracts.NewBoundContract(addr(ob.addresses.ECDSAValidator), contracts.ECDSAValidatorABI()),
	}
}

func (ob *OrderBuilder) predictAccountOrKernel() string {
	if ob.predictAccount != nil {
		return ob.predictAccount.Hex()
	}
	return ob.addresses.Kernel
}

func (ob *OrderBuilder) min(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}

func (ob *OrderBuilder) max(a, b *big.Int) *big.Int {
	if a.Cmp(b) > 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}

func (ob *OrderBuilder) processBook(depths []DepthLevel, quantityWei *big.Int) ProcessedBookAmounts {
	acc := ProcessedBookAmounts{
		QuantityWei:  big.NewInt(0),
		PriceWei:     big.NewInt(0),
		LastPriceWei: big.NewInt(0),
	}
	remaining := new(big.Int).Set(quantityWei)

	for _, level := range depths {
		if remaining.Sign() <= 0 {
			break
		}
		priceWei := internal.ParseEther(level[0])
		qtyWei := internal.ParseEther(level[1])

		if remaining.Cmp(qtyWei) < 0 {
			acc.QuantityWei.Add(acc.QuantityWei, remaining)
			part := new(big.Int).Mul(priceWei, remaining)
			acc.PriceWei.Add(acc.PriceWei, part)
			acc.LastPriceWei = priceWei
			break
		}
		acc.QuantityWei.Add(acc.QuantityWei, qtyWei)
		part := new(big.Int).Mul(priceWei, qtyWei)
		acc.PriceWei.Add(acc.PriceWei, part)
		acc.LastPriceWei = priceWei
		remaining.Sub(remaining, qtyWei)
	}
	return acc
}

// GetLimitOrderAmounts calculates maker/taker amounts for a limit order.
func (ob *OrderBuilder) GetLimitOrderAmounts(data LimitHelperInput) (OrderAmounts, error) {
	if data.QuantityWei.Cmp(big.NewInt(1e16)) < 0 {
		return OrderAmounts{}, ErrInvalidQuantity
	}

	price := internal.RetainSignificantDigits(data.PricePerShareWei, 3)
	qty := internal.RetainSignificantDigits(data.QuantityWei, 5)

	switch data.Side {
	case SideBuy:
		maker := new(big.Int).Div(new(big.Int).Mul(price, qty), ob.precision)
		return OrderAmounts{
			PricePerShare:  price,
			MakerAmount:    maker,
			TakerAmount:    qty,
			Amount:         qty,
			LastPrice:      price,
			SlippageBps:    big.NewInt(0),
			IsMinAmountOut: false,
		}, nil
	case SideSell:
		taker := new(big.Int).Div(new(big.Int).Mul(price, qty), ob.precision)
		return OrderAmounts{
			PricePerShare:  price,
			MakerAmount:    qty,
			TakerAmount:    taker,
			Amount:         qty,
			LastPrice:      price,
			SlippageBps:    big.NewInt(0),
			IsMinAmountOut: false,
		}, nil
	default:
		return OrderAmounts{}, fmt.Errorf("invalid side")
	}
}

// GetMarketOrderAmounts calculates amounts for a market order using an order book.
func (ob *OrderBuilder) GetMarketOrderAmounts(data any, book Book) (OrderAmounts, error) {
	switch d := data.(type) {
	case MarketHelperValueInput:
		if d.Side == SideBuy {
			return ob.getMarketOrderAmountsByValue(d, book)
		}
		return ob.getMarketOrderAmountsByQuantity(MarketHelperInput{
			Side: d.Side, QuantityWei: d.ValueWei, SlippageBps: d.SlippageBps, IsMinAmountOut: d.IsMinAmountOut,
		}, book)
	case MarketHelperInput:
		return ob.getMarketOrderAmountsByQuantity(d, book)
	default:
		return OrderAmounts{}, fmt.Errorf("unsupported market helper input type")
	}
}

func (ob *OrderBuilder) getMarketOrderAmountsByQuantity(data MarketHelperInput, book Book) (OrderAmounts, error) {
	qty := internal.RetainSignificantDigits(data.QuantityWei, 5)
	if qty.Cmp(big.NewInt(1e16)) < 0 {
		return OrderAmounts{}, ErrInvalidQuantity
	}

	slippageBps := big.NewInt(0)
	if data.SlippageBps != nil {
		slippageBps = data.SlippageBps
	}
	isMinAmountOut := data.IsMinAmountOut

	switch data.Side {
	case SideBuy:
		processed := ob.processBook(book.Asks, qty)
		priceWei := processed.PriceWei
		quantityWei := processed.QuantityWei
		lastPriceWei := processed.LastPriceWei

		if isMinAmountOut {
			makerAmount := new(big.Int).Div(priceWei, ob.precision)
			var signedShares *big.Int
			if lastPriceWei.Sign() > 0 {
				signedShares = new(big.Int).Div(priceWei, lastPriceWei)
			} else {
				signedShares = big.NewInt(0)
			}
			takerAmount := new(big.Int).Set(signedShares)
			if slippageBps.Sign() > 0 {
				takerAmount = ob.max(new(big.Int).Div(new(big.Int).Mul(signedShares, new(big.Int).Sub(big.NewInt(10000), slippageBps)), big.NewInt(10000)), big.NewInt(0))
			}
			var pricePerShare *big.Int
			if quantityWei.Sign() > 0 {
				pricePerShare = new(big.Int).Div(priceWei, quantityWei)
			} else {
				pricePerShare = big.NewInt(0)
			}
			return OrderAmounts{
				LastPrice: lastPriceWei, PricePerShare: pricePerShare,
				MakerAmount: makerAmount, TakerAmount: takerAmount, Amount: quantityWei,
				SlippageBps: slippageBps, IsMinAmountOut: true,
			}, nil
		}

		baseMakerAmount := new(big.Int).Div(new(big.Int).Mul(lastPriceWei, quantityWei), ob.precision)
		makerAmount := new(big.Int).Set(baseMakerAmount)
		if slippageBps.Sign() > 0 {
			inflated := new(big.Int).Div(new(big.Int).Mul(baseMakerAmount, new(big.Int).Add(big.NewInt(10000), slippageBps)), big.NewInt(10000))
			makerAmount = ob.min(inflated, quantityWei)
		}
		var pricePerShare *big.Int
		if quantityWei.Sign() > 0 {
			pricePerShare = new(big.Int).Div(priceWei, quantityWei)
		} else {
			pricePerShare = big.NewInt(0)
		}
		return OrderAmounts{
			LastPrice: lastPriceWei, PricePerShare: pricePerShare,
			MakerAmount: makerAmount, TakerAmount: quantityWei, Amount: quantityWei,
			SlippageBps: slippageBps, IsMinAmountOut: false,
		}, nil

	case SideSell:
		processed := ob.processBook(book.Bids, qty)
		priceWei := processed.PriceWei
		quantityWei := processed.QuantityWei
		lastPriceWei := processed.LastPriceWei

		baseTakerAmount := new(big.Int).Div(new(big.Int).Mul(lastPriceWei, quantityWei), ob.precision)
		takerAmount := new(big.Int).Set(baseTakerAmount)
		if slippageBps.Sign() > 0 {
			takerAmount = ob.max(new(big.Int).Div(new(big.Int).Mul(baseTakerAmount, new(big.Int).Sub(big.NewInt(10000), slippageBps)), big.NewInt(10000)), big.NewInt(0))
		}
		var pricePerShare *big.Int
		if quantityWei.Sign() > 0 {
			pricePerShare = new(big.Int).Div(priceWei, quantityWei)
		} else {
			pricePerShare = big.NewInt(0)
		}
		return OrderAmounts{
			LastPrice: lastPriceWei, PricePerShare: pricePerShare,
			MakerAmount: quantityWei, TakerAmount: takerAmount, Amount: quantityWei,
			SlippageBps: slippageBps, IsMinAmountOut: false,
		}, nil
	default:
		return OrderAmounts{}, fmt.Errorf("invalid side")
	}
}

func (ob *OrderBuilder) getMarketOrderAmountsByValue(data MarketHelperValueInput, book Book) (OrderAmounts, error) {
	if data.ValueWei.Cmp(big.NewInt(1e18)) < 0 {
		return OrderAmounts{}, ErrInvalidQuantity
	}

	currencyAmountWei := data.ValueWei
	numberOfShares := big.NewInt(0)
	totalPrice := big.NewInt(0)

	for _, level := range book.Asks {
		priceWei := internal.ParseEther(level[0])
		qtyWei := internal.ParseEther(level[1])
		remainingSpend := new(big.Int).Sub(currencyAmountWei, totalPrice)
		if remainingSpend.Sign() <= 0 {
			break
		}
		tierTotalPrice := new(big.Int).Div(new(big.Int).Mul(priceWei, qtyWei), ob.precision)
		if tierTotalPrice.Cmp(remainingSpend) <= 0 {
			numberOfShares.Add(numberOfShares, qtyWei)
			totalPrice.Add(totalPrice, new(big.Int).Div(new(big.Int).Mul(priceWei, qtyWei), ob.precision))
			continue
		}
		var fractionalShareAmount *big.Int
		if priceWei.Sign() > 0 {
			fractionalShareAmount = new(big.Int).Div(new(big.Int).Mul(remainingSpend, ob.precision), priceWei)
		} else {
			fractionalShareAmount = big.NewInt(0)
		}
		numberOfShares.Add(numberOfShares, fractionalShareAmount)
		totalPrice.Add(totalPrice, new(big.Int).Div(new(big.Int).Mul(priceWei, fractionalShareAmount), ob.precision))
		break
	}

	roundedShares := internal.RetainSignificantDigits(numberOfShares, 5)
	return ob.getMarketOrderAmountsByQuantity(MarketHelperInput{
		Side:           SideBuy,
		QuantityWei:    roundedShares,
		SlippageBps:    data.SlippageBps,
		IsMinAmountOut: data.IsMinAmountOut,
	}, book)
}

// BuildOrder constructs an order for the given strategy.
func (ob *OrderBuilder) BuildOrder(strategy OrderStrategy, data BuildOrderInput) (Order, error) {
	expiresAt := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	if data.ExpiresAt != nil {
		expiresAt = *data.ExpiresAt
	}

	if ob.predictAccount != nil && (data.Maker != "" || data.Signer != "") {
		ob.logger.Warn("[WARN]: When using a Predict account the maker and signer are ignored.")
	}
	if strategy == OrderStrategyMarket && data.ExpiresAt != nil {
		ob.logger.Warn("[WARN]: expiresAt for market orders is ignored.")
	}
	if strategy != OrderStrategyMarket && expiresAt.Before(time.Now()) {
		return Order{}, ErrInvalidExpiration
	}

	signer := ob.signerAddress.Hex()
	if data.Signer != "" {
		signer = data.Signer
	}
	if ob.privateKey == nil && data.Signer == "" {
		return Order{}, ErrMissingSigner
	}
	if data.Maker != "" && signer != data.Maker {
		return Order{}, ErrMakerSignerMismatch
	}

	limitExpiration := expiresAt.Unix()
	marketExpiration := time.Now().Unix() + FiveMinutesSeconds
	expiration := limitExpiration
	if strategy == OrderStrategyMarket {
		expiration = marketExpiration
	}

	maker := signer
	if ob.predictAccount != nil {
		maker = ob.predictAccount.Hex()
		signer = ob.predictAccount.Hex()
	} else if data.Maker != "" {
		maker = data.Maker
	}

	nonce := big.NewInt(0)
	if data.Nonce != nil {
		nonce = data.Nonce
	}
	salt := ob.generateSalt()
	if data.Salt != nil {
		salt = data.Salt.String()
	}

	sigType := data.SignatureType

	taker := ZeroAddress
	if data.Taker != "" {
		taker = data.Taker
	}

	return Order{
		Salt:          salt,
		Maker:         maker,
		Signer:        signer,
		Taker:         taker,
		TokenID:       data.TokenID,
		MakerAmount:   data.MakerAmount.String(),
		TakerAmount:   data.TakerAmount.String(),
		Expiration:    fmt.Sprintf("%d", expiration),
		Nonce:         nonce.String(),
		FeeRateBps:    data.FeeRateBps.String(),
		Side:          data.Side,
		SignatureType: sigType,
	}, nil
}

func trimHexPrefix(s string) string {
	if len(s) >= 2 && (s[0:2] == "0x" || s[0:2] == "0X") {
		return s[2:]
	}
	return s
}
