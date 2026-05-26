package internal

import (
	"math"
	"math/big"
	"strconv"

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

// ParseEtherFloat converts a floating-point ether amount to wei (18 decimals).
func ParseEtherFloat(v float64) *big.Int {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	f, _, err := new(big.Float).Parse(s, 10)
	if err != nil {
		return big.NewInt(0)
	}
	wei := new(big.Float).Mul(f, big.NewFloat(math.Pow10(18)))
	out, _ := wei.Int(nil)
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
