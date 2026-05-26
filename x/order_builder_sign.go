package x

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/exchange-now/go-predict-fun/x/internal"
)

// BuildTypedData builds EIP-712 typed data for an order.
func (ob *OrderBuilder) BuildTypedData(order Order, isNegRisk, isYieldBearing bool) EIP712TypedData {
	verifyingContract := ob.exchangeAddress(isNegRisk, isYieldBearing)
	return EIP712TypedData{
		PrimaryType: "Order",
		Types: EIP712Types{
			"EIP712Domain": EIP712Domain,
			"Order":        OrderStructure,
		},
		Domain: EIP712Object{
			"name":              ProtocolName,
			"version":           ProtocolVersion,
			"chainId":           int64(ob.chainID),
			"verifyingContract": verifyingContract,
		},
		Message: orderToMessage(order),
	}
}

// BuildTypedDataHash returns the EIP-712 hash of typed data.
func (ob *OrderBuilder) BuildTypedDataHash(typedData EIP712TypedData) (common.Hash, error) {
	td, err := toAPITypedData(typedData)
	if err != nil {
		return common.Hash{}, fmt.Errorf("%w: %v", ErrFailedTypedDataEncoder, err)
	}
	hash, _, err := apitypes.TypedDataAndHash(td)
	if err != nil {
		return common.Hash{}, fmt.Errorf("%w: %v", ErrFailedTypedDataEncoder, err)
	}
	return common.BytesToHash(hash), nil
}

// SignTypedDataOrder signs typed data and returns a signed order.
func (ob *OrderBuilder) SignTypedDataOrder(typedData EIP712TypedData) (SignedOrder, error) {
	if ob.privateKey == nil {
		return SignedOrder{}, ErrMissingSigner
	}

	order, err := messageToOrder(typedData.Message)
	if err != nil {
		return SignedOrder{}, err
	}

	if ob.predictAccount != nil {
		hash, err := ob.BuildTypedDataHash(typedData)
		if err != nil {
			return SignedOrder{}, err
		}
		sig, err := ob.signPredictAccountMessage(hash)
		if err != nil {
			return SignedOrder{}, fmt.Errorf("%w: %v", ErrFailedOrderSign, err)
		}
		return SignedOrder{Order: order, Signature: sig}, nil
	}

	td, err := toAPITypedData(typedData)
	if err != nil {
		return SignedOrder{}, fmt.Errorf("%w: %v", ErrFailedTypedDataEncoder, err)
	}
	hash, _, err := apitypes.TypedDataAndHash(td)
	if err != nil {
		return SignedOrder{}, fmt.Errorf("%w: %v", ErrFailedTypedDataEncoder, err)
	}

	sig, err := crypto.Sign(hash[:], ob.privateKey)
	if err != nil {
		return SignedOrder{}, fmt.Errorf("%w: %v", ErrFailedOrderSign, err)
	}
	sig[64] += 27
	return SignedOrder{Order: order, Signature: hexutil.Encode(sig)}, nil
}

func (ob *OrderBuilder) signPredictAccountMessage(messageHash common.Hash) (string, error) {
	if ob.privateKey == nil || ob.predictAccount == nil {
		return "", ErrMissingSigner
	}

	domain := KernelDomainByChainID[ob.chainID]
	digest := internal.EIP712WrapHash(messageHash, domain.Name, domain.Version, int64(domain.ChainID), *ob.predictAccount)

	sig, err := crypto.Sign(digest.Bytes(), ob.privateKey)
	if err != nil {
		return "", err
	}
	sig[64] += 27

	validator := common.HexToAddress(ob.addresses.ECDSAValidator)
	prefix := append([]byte{0x01}, validator.Bytes()...)
	combined := append(prefix, sig...)
	return hexutil.Encode(combined), nil
}

func (ob *OrderBuilder) exchangeAddress(isNegRisk, isYieldBearing bool) string {
	if isNegRisk {
		if isYieldBearing {
			return ob.addresses.YieldBearingNegRiskCTFExchange
		}
		return ob.addresses.NegRiskCTFExchange
	}
	if isYieldBearing {
		return ob.addresses.YieldBearingCTFExchange
	}
	return ob.addresses.CTFExchange
}

func orderToMessage(order Order) EIP712Object {
	return EIP712Object{
		"salt":          order.Salt,
		"maker":         order.Maker,
		"signer":        order.Signer,
		"taker":         order.Taker,
		"tokenId":       order.TokenID,
		"makerAmount":   order.MakerAmount,
		"takerAmount":   order.TakerAmount,
		"expiration":    order.Expiration,
		"nonce":         order.Nonce,
		"feeRateBps":    order.FeeRateBps,
		"side":          order.Side,
		"signatureType": order.SignatureType,
	}
}

func messageToOrder(msg EIP712Object) (Order, error) {
	return Order{
		Salt:          fmt.Sprint(msg["salt"]),
		Maker:         fmt.Sprint(msg["maker"]),
		Signer:        fmt.Sprint(msg["signer"]),
		Taker:         fmt.Sprint(msg["taker"]),
		TokenID:       fmt.Sprint(msg["tokenId"]),
		MakerAmount:   fmt.Sprint(msg["makerAmount"]),
		TakerAmount:   fmt.Sprint(msg["takerAmount"]),
		Expiration:    fmt.Sprint(msg["expiration"]),
		Nonce:         fmt.Sprint(msg["nonce"]),
		FeeRateBps:    fmt.Sprint(msg["feeRateBps"]),
		Side:          Side(toUint8(msg["side"])),
		SignatureType: SignatureType(toUint8(msg["signatureType"])),
	}, nil
}

func toUint8(v any) uint8 {
	switch n := v.(type) {
	case Side:
		return uint8(n)
	case SignatureType:
		return uint8(n)
	case float64:
		return uint8(n)
	case int:
		return uint8(n)
	case int64:
		return uint8(n)
	default:
		return 0
	}
}

func toAPITypedData(data EIP712TypedData) (apitypes.TypedData, error) {
	types := apitypes.Types{}
	for name, fields := range data.Types {
		var typedFields []apitypes.Type
		for _, f := range fields {
			typedFields = append(typedFields, apitypes.Type{Name: f.Name, Type: f.Type})
		}
		types[name] = typedFields
	}

	chainID := math.NewHexOrDecimal256(int64(toInt64(data.Domain["chainId"])))

	domain := apitypes.TypedDataDomain{
		Name:              fmt.Sprint(data.Domain["name"]),
		Version:           fmt.Sprint(data.Domain["version"]),
		ChainId:           chainID,
		VerifyingContract: fmt.Sprint(data.Domain["verifyingContract"]),
	}

	return apitypes.TypedData{
		Types:       types,
		PrimaryType: data.PrimaryType,
		Domain:      domain,
		Message:     data.Message,
	}, nil
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case ChainID:
		return int64(n)
	default:
		return 0
	}
}

// ParseWei parses a decimal wei string into *big.Int.
func ParseWei(s string) (*big.Int, error) {
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("invalid wei string: %s", s)
	}
	return n, nil
}

// ParseWeiMust parses wei or panics (for examples/tests).
func ParseWeiMust(s string) *big.Int {
	n, err := ParseWei(s)
	if err != nil {
		panic(err)
	}
	return n
}
