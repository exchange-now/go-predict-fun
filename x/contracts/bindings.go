package contracts

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// BoundContract holds a contract address and its ABI for encoding calls.
type BoundContract struct {
	Address common.Address
	ABI     abi.ABI
}

// NewBoundContract creates a contract binding.
func NewBoundContract(address common.Address, contractABI abi.ABI) *BoundContract {
	return &BoundContract{Address: address, ABI: contractABI}
}

// PackMethod ABI-encodes a contract method call.
func (c *BoundContract) PackMethod(method string, args ...any) ([]byte, error) {
	return c.ABI.Pack(method, args...)
}

// OrderTuple matches the on-chain Order struct for cancelOrders.
type OrderTuple struct {
	Salt          *big.Int
	Maker         common.Address
	Signer        common.Address
	Taker         common.Address
	TokenID       *big.Int
	MakerAmount   *big.Int
	TakerAmount   *big.Int
	Expiration    *big.Int
	Nonce         *big.Int
	FeeRateBps    *big.Int
	Side          uint8
	SignatureType uint8
}
