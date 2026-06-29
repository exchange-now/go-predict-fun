package x

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// predict.fun 下单联调（真实下单，默认低价挂单，通常不会立即成交）。
//
// 运行前在项目根目录创建 .env（可参考 .env.example），或导出同名环境变量。
// 已设置的环境变量优先于 .env 中的值。
//
//	go test ./x -run TestIntegrationPF -v

type pfIntegrationEnv struct {
	baseURL        string
	apiKey         string
	privateKey     string
	predictAccount string
	pfRPCURL       string
	pfProxyIP      string
	pfProxyUser    string
	pfProxyPass    string
}

var loadTestDotEnvOnce sync.Once

func loadTestDotEnv() {
	loadTestDotEnvOnce.Do(func() {
		// go test ./x 时 cwd 为 x/，仓库根目录的 .env 在上一级。
		_ = godotenv.Load("../.env", ".env")
	})
}

func loadPFIntegrationEnv(t *testing.T) pfIntegrationEnv {
	t.Helper()
	loadTestDotEnv()

	if os.Getenv("PF_INTEGRATION") != "1" {
		t.Skip("set PF_INTEGRATION=1 in .env or environment to run predict.fun live integration test")
	}

	apiKey := os.Getenv("PREDICT_API_KEY")
	privateKey := os.Getenv("PF_PRIVATE_KEY")
	predictAccount := os.Getenv("PF_PREDICT_ACCOUNT")
	if apiKey == "" || privateKey == "" || predictAccount == "" {
		t.Skip("set PREDICT_API_KEY, PF_PRIVATE_KEY, PF_PREDICT_ACCOUNT in .env or environment to run live integration test")
	}

	//chainID := ChainIDBnbTestnet
	chainID := ChainIDBnbMainnet
	rpcURL := RPCURLByChainID[chainID]

	return pfIntegrationEnv{
		baseURL:        APIBaseURLProd,
		apiKey:         apiKey,
		privateKey:     privateKey,
		predictAccount: predictAccount,
		pfRPCURL:       rpcURL, // 留空则用 SDK 默认 BNB RPC
		pfProxyIP:      os.Getenv("PF_PROXY_IP"),
		pfProxyUser:    os.Getenv("PF_PROXY_IP_AUTH_USER"),
		pfProxyPass:    os.Getenv("PF_PROXY_IP_AUTH_PASS"),
	}
}

func loadPFAPIEnv(t *testing.T) pfIntegrationEnv {
	t.Helper()
	loadTestDotEnv()

	if os.Getenv("PF_INTEGRATION") != "1" {
		t.Skip("set PF_INTEGRATION=1 in .env or environment to run predict.fun live integration test")
	}
	apiKey := os.Getenv("PREDICT_API_KEY")
	if apiKey == "" {
		t.Skip("set PREDICT_API_KEY in .env or environment to run live API test")
	}

	return pfIntegrationEnv{
		baseURL:     APIBaseURLProd,
		apiKey:      apiKey,
		pfProxyIP:   os.Getenv("PF_PROXY_IP"),
		pfProxyUser: os.Getenv("PF_PROXY_IP_AUTH_USER"),
		pfProxyPass: os.Getenv("PF_PROXY_IP_AUTH_PASS"),
	}
}

func newPFAPIClient(t *testing.T, env pfIntegrationEnv) *APIClient {
	t.Helper()
	client, err := NewAPIClient(APIClientOptions{
		BaseURL:   env.baseURL,
		APIKey:    env.apiKey,
		ProxyAddr: env.pfProxyIP,
		ProxyUser: env.pfProxyUser,
		ProxyPass: env.pfProxyPass,
	})
	require.NoError(t, err)
	return client
}

func newPFIntegrationClients(t *testing.T, env pfIntegrationEnv) (*APIClient, *OrderBuilder) {
	t.Helper()
	ctx := context.Background()

	chainID := ChainIDBnbMainnet
	if strings.Contains(env.baseURL, "testnet") {
		chainID = ChainIDBnbTestnet
	}

	opts := &OrderBuilderOptions{PredictAccount: env.predictAccount}
	if env.pfRPCURL != "" {
		opts.RPCURL = env.pfRPCURL
	}
	ob, err := NewOrderBuilderWithSigner(ctx, chainID, env.privateKey, opts)
	require.NoError(t, err)
	t.Cleanup(ob.Close)

	return newPFAPIClient(t, env), ob
}

func TestIntegrationPFAuthenticate(t *testing.T) {
	env := loadPFIntegrationEnv(t)
	client, ob := newPFIntegrationClients(t, env)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := client.Authenticate(ctx, ob)
	require.NoError(t, err)
	t.Log("predict.fun JWT auth ok")
}

func TestIntegrationPFCreateLimitOrder(t *testing.T) {
	env := loadPFIntegrationEnv(t)

	var marketID int64 = 473
	outcome := "Yes"
	price := "0.001" // 故意低价避免成交
	size := "5"

	client, ob := newPFIntegrationClients(t, env)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	require.NoError(t, client.Authenticate(ctx, ob))

	market, err := fetchMarketByID(ctx, client, marketID)
	require.NoError(t, err)
	tokenID, err := outcomeToken(*market, outcome)
	require.NoError(t, err)

	t.Logf("market id=%d question=%q outcome=%s tokenId=%s price=%s size=%s",
		market.ID, market.Question, outcome, tokenID, price, size)

	resp, err := client.CreateLimitOrder(ctx, ob, LimitOrderParams{
		TokenID:        tokenID,
		PricePerShare:  price,
		QuantityShares: size,
		FeeRateBps:     market.FeeRateBps,
		IsNegRisk:      market.IsNegRisk,
		IsYieldBearing: market.IsYieldBearing,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.OrderID)
	t.Logf("order placed: orderId=%s orderHash=%s code=%s", resp.OrderID, resp.OrderHash, resp.Code)
}

// 分页拉取 markets，按 id 查找目标市场。
func fetchMarketByID(ctx context.Context, client *APIClient, marketID int64) (*Market, error) {
	var after *string
	openStatus := MarketStatusOpen
	first := "150"
	for {
		params := GetMarketsParams{
			First:  &first,
			After:  after,
			Status: &openStatus,
		}
		resp, err := client.GetMarkets(ctx, params)
		if err != nil {
			return nil, err
		}
		for i := range resp.Data {
			if resp.Data[i].ID == marketID {
				return &resp.Data[i], nil
			}
		}
		if resp.Cursor == nil || *resp.Cursor == "" {
			break
		}
		after = resp.Cursor
	}
	return nil, fmt.Errorf("market not found: id=%d", marketID)
}

// 从 Market.Outcomes 按 outcome 名（Yes/No）取 onChainId。
func outcomeToken(market Market, outcome string) (string, error) {
	for _, o := range market.Outcomes {
		if o.Name == outcome {
			if o.OnChainID == "" {
				return "", fmt.Errorf("outcome %q onChainId empty", outcome)
			}
			return o.OnChainID, nil
		}
	}
	return "", fmt.Errorf("outcome %q not found", outcome)
}
