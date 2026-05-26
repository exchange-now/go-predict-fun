package x

import "errors"

var (
	// ErrMissingSigner is returned when a chain operation requires a signer.
	ErrMissingSigner = errors.New("a signer is required to sign the order")

	// ErrInvalidQuantity is returned when quantityWei is below 1e16.
	ErrInvalidQuantity = errors.New("invalid quantityWei: must be greater than 1e16")

	// ErrInvalidExpiration is returned when a limit order expires in the past.
	ErrInvalidExpiration = errors.New("invalid expiration: must be in the future")

	// ErrFailedOrderSign is returned when EIP-712 signing fails.
	ErrFailedOrderSign = errors.New("failed to EIP-712 sign the order")

	// ErrFailedTypedDataEncoder is returned when typed-data hashing fails.
	ErrFailedTypedDataEncoder = errors.New("failed to hash the order typed data")

	// ErrInvalidNegRiskConfig is returned when token IDs do not match the exchange.
	ErrInvalidNegRiskConfig = errors.New("token ID not registered in the selected contract; check isNegRisk")

	// ErrMakerSignerMismatch is returned when maker and signer differ.
	ErrMakerSignerMismatch = errors.New("the maker and signer must be the same address")

	// ErrInvalidSigner is returned when the Privy wallet does not own the Predict account.
	ErrInvalidSigner = errors.New("signer is not the owner of the Predict account; use the Privy wallet from account settings")
)
