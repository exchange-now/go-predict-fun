package x

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func parseEther(t *testing.T, v float64) *big.Int {
	t.Helper()
	wei := new(big.Int)
	// match ethers parseEther for test literals
	f := new(big.Float).SetFloat64(v)
	f.Mul(f, new(big.Float).SetInt64(1e18))
	f.Int(wei)
	return wei
}

func TestGetLimitOrderAmounts(t *testing.T) {
	ob := NewOrderBuilder(ChainIDBnbMainnet, &OrderBuilderOptions{GenerateSalt: func() string { return "1234" }})

	t.Run("BUY", func(t *testing.T) {
		result, err := ob.GetLimitOrderAmounts(LimitHelperInput{
			Side:             SideBuy,
			PricePerShareWei: parseEther(t, 2),
			QuantityWei:      parseEther(t, 5),
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.PricePerShare.Cmp(parseEther(t, 2)) != 0 {
			t.Fatalf("pricePerShare = %s", result.PricePerShare)
		}
		if result.MakerAmount.Cmp(parseEther(t, 10)) != 0 {
			t.Fatalf("makerAmount = %s", result.MakerAmount)
		}
		if result.TakerAmount.Cmp(parseEther(t, 5)) != 0 {
			t.Fatalf("takerAmount = %s", result.TakerAmount)
		}
	})

	t.Run("SELL", func(t *testing.T) {
		result, err := ob.GetLimitOrderAmounts(LimitHelperInput{
			Side:             SideSell,
			PricePerShareWei: parseEther(t, 2),
			QuantityWei:      parseEther(t, 5),
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.MakerAmount.Cmp(parseEther(t, 5)) != 0 {
			t.Fatalf("makerAmount = %s", result.MakerAmount)
		}
		if result.TakerAmount.Cmp(parseEther(t, 10)) != 0 {
			t.Fatalf("takerAmount = %s", result.TakerAmount)
		}
	})

	t.Run("small quantity", func(t *testing.T) {
		_, err := ob.GetLimitOrderAmounts(LimitHelperInput{
			Side:             SideBuy,
			PricePerShareWei: parseEther(t, 2),
			QuantityWei:      parseEther(t, 0.001),
		})
		if err != ErrInvalidQuantity {
			t.Fatalf("expected ErrInvalidQuantity, got %v", err)
		}
	})
}

func TestGetMarketOrderAmounts(t *testing.T) {
	ob := NewOrderBuilder(ChainIDBnbMainnet, nil)
	book := Book{
		UpdateTimestampMs: 0,
		Asks:              []DepthLevel{{"0.5", "3"}, {"0.88", "4"}},
		Bids:              []DepthLevel{{"0.9", "2"}, {"0.5", "3"}},
	}

	t.Run("BUY by value", func(t *testing.T) {
		result, err := ob.GetMarketOrderAmounts(MarketHelperValueInput{Side: SideBuy, ValueWei: parseEther(t, 1)}, book)
		if err != nil {
			t.Fatal(err)
		}
		if result.MakerAmount.Cmp(parseEther(t, 1)) != 0 {
			t.Fatalf("makerAmount = %s", result.MakerAmount)
		}
		if result.TakerAmount.Cmp(parseEther(t, 2)) != 0 {
			t.Fatalf("takerAmount = %s", result.TakerAmount)
		}
	})

	t.Run("BUY by quantity", func(t *testing.T) {
		result, err := ob.GetMarketOrderAmounts(MarketHelperInput{Side: SideBuy, QuantityWei: parseEther(t, 5)}, book)
		if err != nil {
			t.Fatal(err)
		}
		lastPrice := parseEther(t, 0.88)
		qty := parseEther(t, 5)
		expectedMaker := new(big.Int).Div(new(big.Int).Mul(lastPrice, qty), big.NewInt(1e18))
		if result.MakerAmount.Cmp(expectedMaker) != 0 {
			t.Fatalf("makerAmount = %s, want %s", result.MakerAmount, expectedMaker)
		}
		if result.TakerAmount.Cmp(parseEther(t, 5)) != 0 {
			t.Fatalf("takerAmount = %s", result.TakerAmount)
		}
	})

	t.Run("SELL", func(t *testing.T) {
		result, err := ob.GetMarketOrderAmounts(MarketHelperInput{Side: SideSell, QuantityWei: parseEther(t, 5)}, book)
		if err != nil {
			t.Fatal(err)
		}
		lastPrice := parseEther(t, 0.5)
		qty := parseEther(t, 5)
		expectedTaker := new(big.Int).Div(new(big.Int).Mul(lastPrice, qty), big.NewInt(1e18))
		if result.TakerAmount.Cmp(expectedTaker) != 0 {
			t.Fatalf("takerAmount = %s, want %s", result.TakerAmount, expectedTaker)
		}
	})
}

func TestBuildOrderAndTypedData(t *testing.T) {
	key, _ := hexutil.Decode("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	ob := NewOrderBuilder(ChainIDBnbMainnet, &OrderBuilderOptions{GenerateSalt: func() string { return "99" }})
	_ = key

	// build without signer for structure test
	order, err := ob.BuildOrder(OrderStrategyLimit, BuildOrderInput{
		Side:        SideBuy,
		TokenID:     "1",
		MakerAmount: big.NewInt(1000),
		TakerAmount: big.NewInt(500),
		FeeRateBps:  big.NewInt(0),
		Signer:      "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.Salt != "99" {
		t.Fatalf("salt = %s", order.Salt)
	}

	typed := ob.BuildTypedData(order, false, false)
	if typed.PrimaryType != "Order" {
		t.Fatalf("primaryType = %s", typed.PrimaryType)
	}
}

func TestBuildTypedDataHashMatchesTSSDK(t *testing.T) {
	ob := NewOrderBuilder(ChainIDBnbMainnet, nil)

	typed := EIP712TypedData{
		PrimaryType: "Order",
		Types: EIP712Types{
			"EIP712Domain": EIP712Domain,
			"Order":        OrderStructure,
		},
		Domain: EIP712Object{
			"name":              ProtocolName,
			"version":           ProtocolVersion,
			"chainId":           int64(ChainIDBnbMainnet),
			"verifyingContract": AddressesByChainID[ChainIDBnbMainnet].CTFExchange,
		},
		Message: EIP712Object{
			"salt":          "123456789",
			"maker":         "0x1234567890123456789012345678901234567890",
			"signer":        "0x1234567890123456789012345678901234567890",
			"taker":         ZeroAddress,
			"tokenId":       "12345",
			"makerAmount":   "1000000000000000000",
			"takerAmount":   "2000000000000000000",
			"expiration":    "4102444800",
			"nonce":         "0",
			"feeRateBps":    "100",
			"side":          "0",
			"signatureType": "0",
		},
	}

	hash, err := ob.BuildTypedDataHash(typed)
	if err != nil {
		t.Fatal(err)
	}

	const expected = "0x814000c89efa61ae42a2bcc4c98e06e90c11480b95a12edea00e3411ec76821d"
	if got := hash.Hex(); got != expected {
		t.Fatalf("hash = %s, want %s", got, expected)
	}
}
