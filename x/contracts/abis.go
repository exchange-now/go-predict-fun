package contracts

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abis/*.json
var abiFS embed.FS

var (
	abiOnce sync.Once
	abis    struct {
		CTFExchange                   abi.ABI
		NegRiskCTFExchange            abi.ABI
		NegRiskAdapter                abi.ABI
		ConditionalTokens             abi.ABI
		YieldBearingConditionalTokens abi.ABI
		ERC20                         abi.ABI
		Kernel                        abi.ABI
		ECDSAValidator                abi.ABI
	}
	abiErr error
)

func loadABIs() {
	files := map[string]*abi.ABI{
		"CTFExchange.json":                   &abis.CTFExchange,
		"NegRiskCtfExchange.json":            &abis.NegRiskCTFExchange,
		"NegRiskAdapter.json":                &abis.NegRiskAdapter,
		"ConditionalTokens.json":             &abis.ConditionalTokens,
		"YieldBearingConditionalTokens.json": &abis.YieldBearingConditionalTokens,
		"ERC20.json":                         &abis.ERC20,
		"Kernel.json":                        &abis.Kernel,
		"ECDSAValidator.json":                &abis.ECDSAValidator,
	}
	for name, target := range files {
		raw, err := abiFS.ReadFile("abis/" + name)
		if err != nil {
			abiErr = fmt.Errorf("read abi %s: %w", name, err)
			return
		}
		var parsed any
		if err := json.Unmarshal(raw, &parsed); err != nil {
			abiErr = fmt.Errorf("parse abi json %s: %w", name, err)
			return
		}
		contractABI, err := abi.JSON(strings.NewReader(string(raw)))
		if err != nil {
			abiErr = fmt.Errorf("decode abi %s: %w", name, err)
			return
		}
		*target = contractABI
	}
}

func mustABI() {
	abiOnce.Do(loadABIs)
	if abiErr != nil {
		panic(abiErr)
	}
}

func CTFExchangeABI() abi.ABI        { mustABI(); return abis.CTFExchange }
func NegRiskCTFExchangeABI() abi.ABI { mustABI(); return abis.NegRiskCTFExchange }
func NegRiskAdapterABI() abi.ABI     { mustABI(); return abis.NegRiskAdapter }
func ConditionalTokensABI() abi.ABI  { mustABI(); return abis.ConditionalTokens }
func YieldBearingConditionalTokensABI() abi.ABI {
	mustABI()
	return abis.YieldBearingConditionalTokens
}
func ERC20ABI() abi.ABI          { mustABI(); return abis.ERC20 }
func KernelABI() abi.ABI         { mustABI(); return abis.Kernel }
func ECDSAValidatorABI() abi.ABI { mustABI(); return abis.ECDSAValidator }
