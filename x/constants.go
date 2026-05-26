package x

// MaxSalt is the upper bound for random order salts.
const MaxSalt = 2_147_483_648

// FiveMinutesSeconds is the default market-order expiration window.
const FiveMinutesSeconds = 60 * 5

// ProtocolName is the EIP-712 domain name for CTF Exchange orders.
const ProtocolName = "predict.fun CTF Exchange"

// ProtocolVersion is the EIP-712 domain version.
const ProtocolVersion = "1"

// ZeroAddress is the public-order taker address.
const ZeroAddress = "0x0000000000000000000000000000000000000000"

// ZeroHash is the Kernel execution mode used for Predict account transactions.
const ZeroHash = "0x0000000000000000000000000000000000000000000000000000000000000000"

// ChainID identifies a supported BNB chain.
type ChainID int

const (
	ChainIDBnbMainnet ChainID = 56
	ChainIDBnbTestnet ChainID = 97
)

// SignatureType is the on-chain order signature type.
type SignatureType uint8

const (
	SignatureTypeEOA            SignatureType = 0
	SignatureTypePolyProxy      SignatureType = 1
	SignatureTypePolyGnosisSafe SignatureType = 2
)

// Side is the order side (BUY = bid, SELL = ask).
type Side uint8

const (
	SideBuy  Side = 0
	SideSell Side = 1
)

// EIP712Domain defines the EIP-712 domain type fields.
var EIP712Domain = []EIP712Parameter{
	{Name: "name", Type: "string"},
	{Name: "version", Type: "string"},
	{Name: "chainId", Type: "uint256"},
	{Name: "verifyingContract", Type: "address"},
}

// OrderStructure defines the EIP-712 Order type fields.
var OrderStructure = []EIP712Parameter{
	{Name: "salt", Type: "uint256"},
	{Name: "maker", Type: "address"},
	{Name: "signer", Type: "address"},
	{Name: "taker", Type: "address"},
	{Name: "tokenId", Type: "uint256"},
	{Name: "makerAmount", Type: "uint256"},
	{Name: "takerAmount", Type: "uint256"},
	{Name: "expiration", Type: "uint256"},
	{Name: "nonce", Type: "uint256"},
	{Name: "feeRateBps", Type: "uint256"},
	{Name: "side", Type: "uint8"},
	{Name: "signatureType", Type: "uint8"},
}

// KernelDomain holds EIP-712 domain fields for Kernel smart-wallet signing.
type KernelDomain struct {
	Name    string
	Version string
	ChainID ChainID
}

// KernelDomainByChainID maps chain IDs to Kernel EIP-712 domains.
var KernelDomainByChainID = map[ChainID]KernelDomain{
	ChainIDBnbMainnet: {Name: "Kernel", Version: "0.3.1", ChainID: ChainIDBnbMainnet},
	ChainIDBnbTestnet: {Name: "Kernel", Version: "0.3.1", ChainID: ChainIDBnbTestnet},
}

// Addresses holds deployed contract addresses for a chain.
type Addresses struct {
	YieldBearingCTFExchange              string
	YieldBearingNegRiskCTFExchange       string
	YieldBearingNegRiskAdapter           string
	YieldBearingConditionalTokens        string
	YieldBearingNegRiskConditionalTokens string

	CTFExchange              string
	NegRiskCTFExchange       string
	NegRiskAdapter           string
	ConditionalTokens        string
	NegRiskConditionalTokens string

	USDT           string
	Kernel         string
	ECDSAValidator string
}

// AddressesByChainID maps chain IDs to default contract addresses.
var AddressesByChainID = map[ChainID]Addresses{
	ChainIDBnbMainnet: {
		YieldBearingCTFExchange:              "0x6bEb5a40C032AFc305961162d8204CDA16DECFa5",
		YieldBearingNegRiskCTFExchange:       "0x8A289d458f5a134bA40015085A8F50Ffb681B41d",
		YieldBearingNegRiskAdapter:           "0x41dCe1A4B8FB5e6327701750aF6231B7CD0B2A40",
		YieldBearingConditionalTokens:        "0x9400F8Ad57e9e0F352345935d6D3175975eb1d9F",
		YieldBearingNegRiskConditionalTokens: "0xF64b0b318AAf83BD9071110af24D24445719A07F",

		CTFExchange:              "0x8BC070BEdAB741406F4B1Eb65A72bee27894B689",
		NegRiskCTFExchange:       "0x365fb81bd4A24D6303cd2F19c349dE6894D8d58A",
		NegRiskAdapter:           "0xc3Cf7c252f65E0d8D88537dF96569AE94a7F1A6E",
		ConditionalTokens:        "0x22DA1810B194ca018378464a58f6Ac2B10C9d244",
		NegRiskConditionalTokens: "0x22DA1810B194ca018378464a58f6Ac2B10C9d244",

		USDT:           "0x55d398326f99059fF775485246999027B3197955",
		Kernel:         "0xBAC849bB641841b44E965fB01A4Bf5F074f84b4D",
		ECDSAValidator: "0x845ADb2C711129d4f3966735eD98a9F09fC4cE57",
	},
	ChainIDBnbTestnet: {
		YieldBearingCTFExchange:              "0x8a6B4Fa700A1e310b106E7a48bAFa29111f66e89",
		YieldBearingNegRiskCTFExchange:       "0x95D5113bc50eD201e319101bbca3e0E250662fCC",
		YieldBearingNegRiskAdapter:           "0xb74aea04bdeBE912Aa425bC9173F9668e6f11F99",
		YieldBearingConditionalTokens:        "0x38BF1cbD66d174bb5F3037d7068E708861D68D7f",
		YieldBearingNegRiskConditionalTokens: "0x26e865CbaAe99b62fbF9D18B55c25B5E079A93D5",

		CTFExchange:              "0x2A6413639BD3d73a20ed8C95F634Ce198ABbd2d7",
		NegRiskCTFExchange:       "0xd690b2bd441bE36431F6F6639D7Ad351e7B29680",
		NegRiskAdapter:           "0x285c1B939380B130D7EBd09467b93faD4BA623Ed",
		ConditionalTokens:        "0x2827AAef52D71910E8FBad2FfeBC1B6C2DA37743",
		NegRiskConditionalTokens: "0x2827AAef52D71910E8FBad2FfeBC1B6C2DA37743",

		USDT:           "0xB32171ecD878607FFc4F8FC0bCcE6852BB3149E0",
		Kernel:         "0xBAC849bB641841b44E965fB01A4Bf5F074f84b4D",
		ECDSAValidator: "0x845ADb2C711129d4f3966735eD98a9F09fC4cE57",
	},
}

// RPCURLByChainID maps chain IDs to default BNB RPC endpoints.
var RPCURLByChainID = map[ChainID]string{
	ChainIDBnbMainnet: "https://bsc-dataseed.bnbchain.org/",
	ChainIDBnbTestnet: "https://bsc-testnet-dataseed.bnbchain.org/",
}
