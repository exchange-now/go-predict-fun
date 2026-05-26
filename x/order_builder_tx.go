package x

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/exchange-now/go-predict-fun/x/contracts"
	"github.com/exchange-now/go-predict-fun/x/internal"
)

func (ob *OrderBuilder) requireContracts() error {
	if ob.contracts == nil || ob.client == nil || ob.privateKey == nil {
		return ErrMissingSigner
	}
	return nil
}

func (ob *OrderBuilder) ownerAddress() common.Address {
	if ob.predictAccount != nil {
		return *ob.predictAccount
	}
	return ob.signerAddress
}

func (ob *OrderBuilder) readECDSAValidatorOwner(ctx context.Context, account common.Address) (common.Address, error) {
	data, err := ob.contracts.ecdsaValidator.PackMethod("ecdsaValidatorStorage", account)
	if err != nil {
		return common.Address{}, err
	}
	out, err := ob.client.CallContract(ctx, ethereum.CallMsg{To: &ob.contracts.ecdsaValidator.Address, Data: data}, nil)
	if err != nil {
		return common.Address{}, err
	}
	vals, err := ob.contracts.ecdsaValidator.ABI.Unpack("ecdsaValidatorStorage", out)
	if err != nil || len(vals) == 0 {
		return common.Address{}, fmt.Errorf("unpack ecdsaValidatorStorage: %w", err)
	}
	return vals[0].(common.Address), nil
}

// BalanceOf returns the USDT balance for the signer or the given address.
func (ob *OrderBuilder) BalanceOf(ctx context.Context, address *common.Address) (*big.Int, error) {
	if err := ob.requireContracts(); err != nil {
		return nil, err
	}
	who := ob.ownerAddress()
	if address != nil {
		who = *address
	}
	data, err := ob.contracts.usdt.PackMethod("balanceOf", who)
	if err != nil {
		return nil, err
	}
	out, err := ob.client.CallContract(ctx, ethereum.CallMsg{To: &ob.contracts.usdt.Address, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	vals, err := ob.contracts.usdt.ABI.Unpack("balanceOf", out)
	if err != nil {
		return nil, err
	}
	return vals[0].(*big.Int), nil
}

func (ob *OrderBuilder) exchangeContract(isNegRisk, isYieldBearing bool) *contracts.BoundContract {
	if isNegRisk {
		if isYieldBearing {
			return ob.contracts.yieldBearingNegRiskCTFExchange
		}
		return ob.contracts.negRiskCTFExchange
	}
	if isYieldBearing {
		return ob.contracts.yieldBearingCTFExchange
	}
	return ob.contracts.ctfExchange
}

func (ob *OrderBuilder) ctfContract(isNegRisk, isYieldBearing bool) *contracts.BoundContract {
	if isYieldBearing {
		if isNegRisk {
			return ob.contracts.yieldBearingNegRiskConditionalTokens
		}
		return ob.contracts.yieldBearingConditionalTokens
	}
	if isNegRisk {
		return ob.contracts.negRiskConditionalTokens
	}
	return ob.contracts.conditionalTokens
}

func (ob *OrderBuilder) negRiskAdapter(isYieldBearing bool) *contracts.BoundContract {
	if isYieldBearing {
		return ob.contracts.yieldBearingNegRiskAdapter
	}
	return ob.contracts.negRiskAdapter
}

// ValidateTokenIDs checks whether token IDs are registered on the selected exchange.
func (ob *OrderBuilder) ValidateTokenIDs(ctx context.Context, tokenIDs []*big.Int, isNegRisk, isYieldBearing bool) (bool, error) {
	if err := ob.requireContracts(); err != nil {
		return false, err
	}
	ex := ob.exchangeContract(isNegRisk, isYieldBearing)
	for _, id := range tokenIDs {
		data, err := ex.PackMethod("validateTokenId", id)
		if err != nil {
			return false, err
		}
		if _, err := ob.client.CallContract(ctx, ethereum.CallMsg{To: &ex.Address, Data: data}, nil); err != nil {
			return false, nil
		}
	}
	return true, nil
}

// CancelOrders cancels on-chain orders.
func (ob *OrderBuilder) CancelOrders(ctx context.Context, orders []Order, opts CancelOrdersOptions) (TransactionResult, error) {
	if err := ob.requireContracts(); err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return TransactionSuccess{Success: true}, nil
	}

	withValidation := true
	if opts.WithValidation != nil {
		withValidation = *opts.WithValidation
	}
	if withValidation {
		tokenIDs := make([]*big.Int, len(orders))
		for i, o := range orders {
			id, err := ParseWei(o.TokenID)
			if err != nil {
				return nil, err
			}
			tokenIDs[i] = id
		}
		ok, err := ob.ValidateTokenIDs(ctx, tokenIDs, opts.IsNegRisk, opts.IsYieldBearing)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrInvalidNegRiskConfig
		}
	}

	tuples := make([]contracts.OrderTuple, len(orders))
	for i, o := range orders {
		t, err := orderToTuple(o)
		if err != nil {
			return nil, err
		}
		tuples[i] = t
	}

	ex := ob.exchangeContract(opts.IsNegRisk, opts.IsYieldBearing)
	data, err := ex.PackMethod("cancelOrders", tuples)
	if err != nil {
		return nil, err
	}

	if ob.predictAccount != nil {
		encoded := internal.EncodeExecutionCalldata(ex.Address, data, big.NewInt(0))
		kData, err := ob.contracts.kernel.PackMethod("execute", common.HexToHash(ZeroHash), encoded)
		if err != nil {
			return nil, err
		}
		return ob.sendTx(ctx, ob.contracts.kernel.Address, kData)
	}
	return ob.sendTx(ctx, ex.Address, data)
}

// SetApprovals sets all protocol approvals needed for trading.
func (ob *OrderBuilder) SetApprovals(ctx context.Context) (SetApprovalsResult, error) {
	ops := []func(context.Context) (TransactionResult, error){
		func(c context.Context) (TransactionResult, error) {
			return ob.SetCTFExchangeApproval(c, false, false, true)
		},
		func(c context.Context) (TransactionResult, error) {
			return ob.SetCTFExchangeApproval(c, true, false, true)
		},
		func(c context.Context) (TransactionResult, error) {
			return ob.SetNegRiskAdapterApproval(c, false, true)
		},
		func(c context.Context) (TransactionResult, error) {
			return ob.SetCTFExchangeAllowance(c, false, false, nil, nil)
		},
		func(c context.Context) (TransactionResult, error) {
			return ob.SetCTFExchangeAllowance(c, true, false, nil, nil)
		},
		func(c context.Context) (TransactionResult, error) {
			return ob.SetCTFExchangeApproval(c, false, true, true)
		},
		func(c context.Context) (TransactionResult, error) {
			return ob.SetCTFExchangeApproval(c, true, true, true)
		},
		func(c context.Context) (TransactionResult, error) { return ob.SetNegRiskAdapterApproval(c, true, true) },
		func(c context.Context) (TransactionResult, error) {
			return ob.SetCTFExchangeAllowance(c, false, true, nil, nil)
		},
		func(c context.Context) (TransactionResult, error) {
			return ob.SetCTFExchangeAllowance(c, true, true, nil, nil)
		},
	}

	var results []TransactionResult
	for _, op := range ops {
		r, err := op(ctx)
		if err != nil {
			return SetApprovalsResult{}, err
		}
		results = append(results, r)
	}
	success := true
	for _, r := range results {
		if !r.Ok() {
			success = false
			break
		}
	}
	return SetApprovalsResult{Success: success, Transactions: results}, nil
}

// SetCTFExchangeApproval sets ERC-1155 approval for the CTF exchange.
func (ob *OrderBuilder) SetCTFExchangeApproval(ctx context.Context, isNegRisk, isYieldBearing, approved bool) (TransactionResult, error) {
	if err := ob.requireContracts(); err != nil {
		return nil, err
	}
	ex := ob.exchangeContract(isNegRisk, isYieldBearing)
	ctf := ob.ctfContract(isNegRisk, isYieldBearing)
	return ob.setERC1155Approval(ctx, ctf, ex.Address, approved)
}

// SetNegRiskAdapterApproval sets ERC-1155 approval for the neg-risk adapter.
func (ob *OrderBuilder) SetNegRiskAdapterApproval(ctx context.Context, isYieldBearing, approved bool) (TransactionResult, error) {
	if err := ob.requireContracts(); err != nil {
		return nil, err
	}
	adapter := ob.negRiskAdapter(isYieldBearing)
	ctf := ob.ctfContract(true, isYieldBearing)
	return ob.setERC1155Approval(ctx, ctf, adapter.Address, approved)
}

// SetCTFExchangeAllowance sets USDT allowance for the CTF exchange.
func (ob *OrderBuilder) SetCTFExchangeAllowance(ctx context.Context, isNegRisk, isYieldBearing bool, minAmount, maxAmount *big.Int) (TransactionResult, error) {
	if err := ob.requireContracts(); err != nil {
		return nil, err
	}
	if minAmount == nil {
		minAmount = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))
	}
	if maxAmount == nil {
		maxAmount = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	}
	ex := ob.exchangeContract(isNegRisk, isYieldBearing)
	current, err := ob.readERC20Allowance(ctx, ex.Address)
	if err != nil {
		return nil, err
	}
	if current.Cmp(minAmount) >= 0 {
		return TransactionSuccess{Success: true}, nil
	}
	return ob.setERC20Approval(ctx, ex.Address, maxAmount)
}

func (ob *OrderBuilder) setERC1155Approval(ctx context.Context, ctf *contracts.BoundContract, operator common.Address, approved bool) (TransactionResult, error) {
	owner := ob.ownerAddress()
	data, err := ctf.PackMethod("isApprovedForAll", owner, operator)
	if err != nil {
		return nil, err
	}
	out, err := ob.client.CallContract(ctx, ethereum.CallMsg{To: &ctf.Address, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	vals, err := ctf.ABI.Unpack("isApprovedForAll", out)
	if err != nil {
		return nil, err
	}
	if vals[0].(bool) == approved {
		return TransactionSuccess{Success: true}, nil
	}

	inner, err := ctf.PackMethod("setApprovalForAll", operator, approved)
	if err != nil {
		return nil, err
	}
	if ob.predictAccount != nil {
		encoded := internal.EncodeExecutionCalldata(ctf.Address, inner, big.NewInt(0))
		kData, err := ob.contracts.kernel.PackMethod("execute", common.HexToHash(ZeroHash), encoded)
		if err != nil {
			return nil, err
		}
		return ob.sendTx(ctx, ob.contracts.kernel.Address, kData)
	}
	return ob.sendTx(ctx, ctf.Address, inner)
}

func (ob *OrderBuilder) readERC20Allowance(ctx context.Context, spender common.Address) (*big.Int, error) {
	data, err := ob.contracts.usdt.PackMethod("allowance", ob.ownerAddress(), spender)
	if err != nil {
		return nil, err
	}
	out, err := ob.client.CallContract(ctx, ethereum.CallMsg{To: &ob.contracts.usdt.Address, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	vals, err := ob.contracts.usdt.ABI.Unpack("allowance", out)
	if err != nil {
		return nil, err
	}
	return vals[0].(*big.Int), nil
}

func (ob *OrderBuilder) setERC20Approval(ctx context.Context, spender common.Address, amount *big.Int) (TransactionResult, error) {
	inner, err := ob.contracts.usdt.PackMethod("approve", spender, amount)
	if err != nil {
		return nil, err
	}
	if ob.predictAccount != nil {
		encoded := internal.EncodeExecutionCalldata(ob.contracts.usdt.Address, inner, big.NewInt(0))
		kData, err := ob.contracts.kernel.PackMethod("execute", common.HexToHash(ZeroHash), encoded)
		if err != nil {
			return nil, err
		}
		return ob.sendTx(ctx, ob.contracts.kernel.Address, kData)
	}
	return ob.sendTx(ctx, ob.contracts.usdt.Address, inner)
}

// RedeemPositions redeems conditional token positions.
func (ob *OrderBuilder) RedeemPositions(ctx context.Context, opts RedeemPositionsOptions) (TransactionResult, error) {
	if err := ob.requireContracts(); err != nil {
		return nil, err
	}
	conditionID := common.HexToHash(opts.ConditionID)

	if opts.IsNegRisk {
		if opts.Amount == nil {
			return nil, fmt.Errorf("amount is required for NegRisk markets")
		}
		adapter := ob.negRiskAdapter(opts.IsYieldBearing)
		var amounts [2]*big.Int
		if opts.IndexSet == 1 {
			amounts = [2]*big.Int{opts.Amount, big.NewInt(0)}
		} else {
			amounts = [2]*big.Int{big.NewInt(0), opts.Amount}
		}
		inner, err := adapter.PackMethod("redeemPositions", conditionID, amounts)
		if err != nil {
			return nil, err
		}
		return ob.executeOrDirect(ctx, adapter, inner)
	}

	ctf := ob.ctfContract(opts.IsNegRisk, opts.IsYieldBearing)
	indexSet := []*big.Int{big.NewInt(int64(opts.IndexSet))}
	inner, err := ctf.PackMethod("redeemPositions", common.HexToAddress(ob.addresses.USDT), common.Hash{}, conditionID, indexSet)
	if err != nil {
		return nil, err
	}
	return ob.executeOrDirect(ctx, ctf, inner)
}

// MergePositions merges outcome tokens back into USDT.
func (ob *OrderBuilder) MergePositions(ctx context.Context, opts MergePositionsOptions) (TransactionResult, error) {
	if err := ob.requireContracts(); err != nil {
		return nil, err
	}
	conditionID := common.HexToHash(opts.ConditionID)

	if opts.IsNegRisk {
		adapter := ob.negRiskAdapter(opts.IsYieldBearing)
		inner, err := adapter.PackMethod("mergePositions(bytes32,uint256)", conditionID, opts.Amount)
		if err != nil {
			return nil, err
		}
		return ob.executeOrDirect(ctx, adapter, inner)
	}

	ctf := ob.ctfContract(opts.IsNegRisk, opts.IsYieldBearing)
	partition := []*big.Int{big.NewInt(1), big.NewInt(2)}
	inner, err := ctf.PackMethod("mergePositions", common.HexToAddress(ob.addresses.USDT), common.Hash{}, conditionID, partition, opts.Amount)
	if err != nil {
		return nil, err
	}
	return ob.executeOrDirect(ctx, ctf, inner)
}

// SplitPositions splits USDT into outcome tokens.
func (ob *OrderBuilder) SplitPositions(ctx context.Context, opts SplitPositionsOptions) (TransactionResult, error) {
	if err := ob.requireContracts(); err != nil {
		return nil, err
	}
	conditionID := common.HexToHash(opts.ConditionID)

	if opts.IsNegRisk {
		adapter := ob.negRiskAdapter(opts.IsYieldBearing)
		inner, err := adapter.PackMethod("splitPosition(bytes32,uint256)", conditionID, opts.Amount)
		if err != nil {
			return nil, err
		}
		return ob.executeOrDirect(ctx, adapter, inner)
	}

	ctf := ob.ctfContract(opts.IsNegRisk, opts.IsYieldBearing)
	partition := []*big.Int{big.NewInt(1), big.NewInt(2)}
	inner, err := ctf.PackMethod("splitPosition", common.HexToAddress(ob.addresses.USDT), common.Hash{}, conditionID, partition, opts.Amount)
	if err != nil {
		return nil, err
	}
	return ob.executeOrDirect(ctx, ctf, inner)
}

func (ob *OrderBuilder) executeOrDirect(ctx context.Context, target *contracts.BoundContract, inner []byte) (TransactionResult, error) {
	if ob.predictAccount != nil {
		encoded := internal.EncodeExecutionCalldata(target.Address, inner, big.NewInt(0))
		kData, err := ob.contracts.kernel.PackMethod("execute", common.HexToHash(ZeroHash), encoded)
		if err != nil {
			return nil, err
		}
		return ob.sendTx(ctx, ob.contracts.kernel.Address, kData)
	}
	return ob.sendTx(ctx, target.Address, inner)
}

func (ob *OrderBuilder) sendTx(ctx context.Context, to common.Address, data []byte) (TransactionResult, error) {
	chainID, err := ob.client.ChainID(ctx)
	if err != nil {
		return TransactionFail{Success: false, Cause: err}, nil
	}

	nonce, err := ob.client.PendingNonceAt(ctx, ob.signerAddress)
	if err != nil {
		return TransactionFail{Success: false, Cause: err}, nil
	}

	gasPrice, err := ob.client.SuggestGasPrice(ctx)
	if err != nil {
		return TransactionFail{Success: false, Cause: err}, nil
	}

	msg := ethereum.CallMsg{From: ob.signerAddress, To: &to, Data: data}
	gasLimit, err := ob.client.EstimateGas(ctx, msg)
	if err != nil {
		return TransactionFail{Success: false, Cause: err}, nil
	}
	gasLimit = gasLimit * 125 / 100

	tx := types.NewTransaction(nonce, to, big.NewInt(0), gasLimit, gasPrice, data)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), ob.privateKey)
	if err != nil {
		return TransactionFail{Success: false, Cause: err}, nil
	}
	if err := ob.client.SendTransaction(ctx, signed); err != nil {
		return TransactionFail{Success: false, Cause: err}, nil
	}

	receipt, err := bind.WaitMined(ctx, ob.client, signed)
	if err != nil {
		hash := signed.Hash()
		return TransactionFail{Success: false, Cause: err, TxHash: &hash}, nil
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		hash := signed.Hash()
		return TransactionFail{Success: false, TxHash: &hash}, nil
	}
	return TransactionSuccess{Success: true, TxHash: signed.Hash()}, nil
}

func orderToTuple(o Order) (contracts.OrderTuple, error) {
	salt, err := ParseWei(o.Salt)
	if err != nil {
		return contracts.OrderTuple{}, err
	}
	tokenID, err := ParseWei(o.TokenID)
	if err != nil {
		return contracts.OrderTuple{}, err
	}
	makerAmount, err := ParseWei(o.MakerAmount)
	if err != nil {
		return contracts.OrderTuple{}, err
	}
	takerAmount, err := ParseWei(o.TakerAmount)
	if err != nil {
		return contracts.OrderTuple{}, err
	}
	expiration, err := ParseWei(o.Expiration)
	if err != nil {
		return contracts.OrderTuple{}, err
	}
	nonce, err := ParseWei(o.Nonce)
	if err != nil {
		return contracts.OrderTuple{}, err
	}
	feeRateBps, err := ParseWei(o.FeeRateBps)
	if err != nil {
		return contracts.OrderTuple{}, err
	}
	return contracts.OrderTuple{
		Salt:          salt,
		Maker:         common.HexToAddress(o.Maker),
		Signer:        common.HexToAddress(o.Signer),
		Taker:         common.HexToAddress(o.Taker),
		TokenID:       tokenID,
		MakerAmount:   makerAmount,
		TakerAmount:   takerAmount,
		Expiration:    expiration,
		Nonce:         nonce,
		FeeRateBps:    feeRateBps,
		Side:          uint8(o.Side),
		SignatureType: uint8(o.SignatureType),
	}, nil
}

// Close closes the underlying RPC client.
func (ob *OrderBuilder) Close() {
	if ob.client != nil {
		ob.client.Close()
	}
}

// SignerAddress returns the EOA address derived from the private key.
func (ob *OrderBuilder) SignerAddress() common.Address {
	return ob.signerAddress
}

// ChainIDValue returns the configured chain ID.
func (ob *OrderBuilder) ChainIDValue() ChainID {
	return ob.chainID
}

// AddressesValue returns the contract addresses in use.
func (ob *OrderBuilder) AddressesValue() Addresses {
	return ob.addresses
}

// HashMessage computes the EIP-191 personal_sign hash of a message string.
func HashMessage(message string) common.Hash {
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	return crypto.Keccak256Hash([]byte(prefix))
}
