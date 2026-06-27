package internal

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// RetainSignificantDigits truncates an integer to the given number of significant digits.
func RetainSignificantDigits(num *big.Int, significantDigits int) *big.Int {
	if num == nil || num.Sign() == 0 {
		return big.NewInt(0)
	}

	isNegative := num.Sign() < 0
	absNum := new(big.Int).Abs(num)

	str := absNum.String()
	magnitude := len(str)
	excess := magnitude - significantDigits
	if excess <= 0 {
		return new(big.Int).Set(num)
	}

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(excess)), nil)
	result := new(big.Int).Div(absNum, divisor)
	result.Mul(result, divisor)

	if isNegative {
		result.Neg(result)
	}
	return result
}

// ParseEther converts a decimal string amount to wei (18 decimals).
//
// Parsing is done with integer arithmetic on the decimal string, so values
// coming straight off the wire (e.g. a json.Number like "0.585") never pass
// through a float64 and cannot lose precision. Extra fractional digits beyond
// 18 are truncated, matching the on-chain wei resolution.
func ParseEther(s json.Number) *big.Int {
	return parseUnits(s.String(), 18)
}

// parseUnits converts a decimal string into base units scaled by 10^decimals.
//
// It is fully integer-exact for plain decimal input. Scientific-notation input
// (containing 'e'/'E') falls back to a high-precision big.Float conversion.
func parseUnits(s string, decimals int) *big.Int {
	s = strings.TrimSpace(s)
	if s == "" {
		return big.NewInt(0)
	}
	if strings.ContainsAny(s, "eE") {
		return parseUnitsFloat(s, decimals)
	}

	neg := false
	switch {
	case strings.HasPrefix(s, "-"):
		neg, s = true, s[1:]
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	}

	intPart, fracPart := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, fracPart = s[:i], s[i+1:]
	}
	if intPart == "" {
		intPart = "0"
	}
	if len(fracPart) > decimals {
		fracPart = fracPart[:decimals]
	}
	for len(fracPart) < decimals {
		fracPart += "0"
	}

	out, ok := new(big.Int).SetString(intPart+fracPart, 10)
	if !ok {
		return big.NewInt(0)
	}
	if neg {
		out.Neg(out)
	}
	return out
}

func parseUnitsFloat(s string, decimals int) *big.Int {
	f, _, err := big.ParseFloat(s, 10, 256, big.ToNearestEven)
	if err != nil {
		return big.NewInt(0)
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	f.Mul(f, new(big.Float).SetPrec(256).SetInt(scale))
	out, _ := f.Int(nil)
	return out
}

// HashKernelMessage wraps a message hash for Kernel smart-wallet signing.
func HashKernelMessage(messageHash common.Hash) common.Hash {
	kernelTypeHash := crypto.Keccak256Hash([]byte("Kernel(bytes32 hash)"))
	encoded, err := abi.Arguments{
		{Type: mustType("bytes32")},
		{Type: mustType("bytes32")},
	}.Pack(kernelTypeHash, messageHash)
	if err != nil {
		panic(err)
	}
	return crypto.Keccak256Hash(encoded)
}

// EIP712WrapHash wraps a message hash with the Kernel EIP-712 domain separator.
func EIP712WrapHash(messageHash common.Hash, name, version string, chainID int64, verifyingContract common.Address) common.Hash {
	domainSeparator := hashEIP712Domain(name, version, chainID, verifyingContract)
	finalMessageHash := HashKernelMessage(messageHash)

	var data []byte
	data = append(data, 0x19, 0x01)
	data = append(data, domainSeparator.Bytes()...)
	data = append(data, finalMessageHash.Bytes()...)
	return crypto.Keccak256Hash(data)
}

func hashEIP712Domain(name, version string, chainID int64, verifyingContract common.Address) common.Hash {
	domainTypeHash := crypto.Keccak256Hash([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	nameHash := crypto.Keccak256Hash([]byte(name))
	versionHash := crypto.Keccak256Hash([]byte(version))

	encoded, err := abi.Arguments{
		{Type: mustType("bytes32")},
		{Type: mustType("bytes32")},
		{Type: mustType("bytes32")},
		{Type: mustType("uint256")},
		{Type: mustType("address")},
	}.Pack(
		domainTypeHash,
		nameHash,
		versionHash,
		big.NewInt(chainID),
		verifyingContract,
	)
	if err != nil {
		panic(err)
	}
	return crypto.Keccak256Hash(encoded)
}

func mustType(t string) abi.Type {
	typ, err := abi.NewType(t, "", nil)
	if err != nil {
		panic(err)
	}
	return typ
}

// EncodeExecutionCalldata encodes Kernel execute calldata (to, value, inner calldata).
func EncodeExecutionCalldata(to common.Address, innerCalldata []byte, value *big.Int) []byte {
	if value == nil {
		value = big.NewInt(0)
	}
	valueBytes := common.LeftPadBytes(value.Bytes(), 32)
	out := make([]byte, 0, 20+32+len(innerCalldata))
	out = append(out, to.Bytes()...)
	out = append(out, valueBytes...)
	out = append(out, innerCalldata...)
	return out
}

// ConcatHex concatenates hex byte slices into one 0x-prefixed hex string.
func ConcatHex(parts ...[]byte) string {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return hexutil.Encode(out)
}
