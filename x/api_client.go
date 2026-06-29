package x

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// APIClient calls the predict.fun REST API.
type APIClient struct {
	baseURL    string
	apiKey     string
	bearer     string
	httpClient *http.Client
}

// APIClientOptions configures a new APIClient.
type APIClientOptions struct {
	// BaseURL defaults to APIBaseURLProd. Use APIBaseURLTestnet for testnet.
	BaseURL string
	// APIKey is sent as the x-api-key header (required on mainnet).
	APIKey string
	// BearerToken is sent as Authorization: Bearer <token> when set.
	BearerToken string
	// ProxyAddr 代理地址 host:port，为空时使用环境变量或直连。
	ProxyAddr string
	// ProxyUser / ProxyPass 代理认证，地址已含凭证时可留空。
	ProxyUser  string
	ProxyPass  string
	HTTPClient *http.Client
}

// NewAPIClient creates a REST API client.
func NewAPIClient(opts APIClientOptions) (*APIClient, error) {
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = APIBaseURLProd
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		var err error
		httpClient, err = ConfigureHTTPClient(opts.ProxyAddr, opts.ProxyUser, opts.ProxyPass, 30*time.Second)
		if err != nil {
			return nil, err
		}
	}
	return &APIClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     opts.APIKey,
		bearer:     opts.BearerToken,
		httpClient: httpClient,
	}, nil
}

func (c *APIClient) GetMarkets(ctx context.Context, params GetMarketsParams) (*GetMarketsResponse, error) {
	// GET /v1/markets
	// Docs: https://dev.predict.fun/get-markets-25326905e0
	q := url.Values{}
	if params.First != nil {
		q.Set("first", *params.First)
	}
	if params.After != nil {
		q.Set("after", *params.After)
	}
	if params.Status != nil {
		q.Set("status", string(*params.Status))
	}
	for _, id := range params.TagIDs {
		q.Add("tagIds", id)
	}
	if params.MarketVariant != nil {
		q.Set("marketVariant", string(*params.MarketVariant))
	}
	if params.Sort != nil {
		q.Set("sort", string(*params.Sort))
	}
	if params.HasActiveRewards != nil {
		q.Set("hasActiveRewards", strconv.FormatBool(*params.HasActiveRewards))
	}

	var resp GetMarketsResponse
	if err := c.get(ctx, "/v1/markets", q, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *APIClient) GetMarketOrderbook(ctx context.Context, marketID int64) (*Book, error) {
	// GET /v1/markets/{id}/orderbook
	var resp GetMarketOrderbookResponse
	path := fmt.Sprintf("/v1/markets/%d/orderbook", marketID)
	if err := c.get(ctx, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetAuthMessage 拉取 JWT 认证待签消息。
func (c *APIClient) GetAuthMessage(ctx context.Context) (string, error) {
	var resp GetAuthMessageResponse
	if err := c.get(ctx, "/v1/auth/message", nil, &resp); err != nil {
		return "", err
	}
	if resp.Data.Message == "" {
		return "", fmt.Errorf("predict.fun auth: empty message")
	}
	return resp.Data.Message, nil
}

// PostAuth 提交签名换取 JWT，并缓存到客户端 bearer。
func (c *APIClient) PostAuth(ctx context.Context, req PostAuthRequest) error {
	var resp PostAuthResponse
	if err := c.post(ctx, "/v1/auth", req, &resp); err != nil {
		return err
	}
	if resp.Data.Token == "" {
		return fmt.Errorf("predict.fun auth: empty token")
	}
	c.bearer = resp.Data.Token
	return nil
}

// Authenticate 完成 JWT 认证流程：拉 message → 签名 → 换 token。
func (c *APIClient) Authenticate(ctx context.Context, ob *OrderBuilder) error {
	message, err := c.GetAuthMessage(ctx)
	if err != nil {
		return err
	}
	signature, err := ob.SignAuthMessage(message)
	if err != nil {
		return fmt.Errorf("sign auth message: %w", err)
	}
	return c.PostAuth(ctx, PostAuthRequest{
		Signer:    ob.AuthSigner(),
		Message:   message,
		Signature: signature,
	})
}

// CreateOrder 提交已签名订单到 POST /v1/orders。
func (c *APIClient) CreateOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResultData, error) {
	var resp CreateOrderResponse
	if err := c.post(ctx, "/v1/orders", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *APIClient) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := decodeAPIResponse(body, out); err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("predict.fun api http %d: %s", res.StatusCode, truncateBody(body))
	}
	return nil
}

func (c *APIClient) post(ctx context.Context, path string, body any, out any) error {
	u := c.baseURL + path
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := decodeAPIResponse(respBody, out); err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("predict.fun api http %d: %s", res.StatusCode, truncateBody(respBody))
	}
	return nil
}

func decodeAPIResponse(body []byte, out any) error {
	var envelope struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Error   string `json:"error"`
		Message string `json:"message"`
		Trace   string `json:"trace"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decode api response: %w", err)
	}
	if !envelope.Success {
		return &APIError{
			Code:      envelope.Code,
			ErrorCode: envelope.Error,
			Message:   envelope.Message,
			Trace:     envelope.Trace,
		}
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode api response body: %w", err)
	}
	return nil
}

func truncateBody(body []byte) string {
	const max = 256
	if len(body) <= max {
		return string(bytes.TrimSpace(body))
	}
	return string(bytes.TrimSpace(body[:max])) + "..."
}
