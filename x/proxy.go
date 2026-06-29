package x

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ConfigureHTTPClient 根据代理地址构建 HTTP 客户端；proxyAddr 为空时使用环境变量代理。
// proxyAddr 格式为 host:port，未带 scheme 时默认 http。
func ConfigureHTTPClient(proxyAddr, proxyUser, proxyPass string, timeout time.Duration) (*http.Client, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	proxyFn, err := buildProxyFunc(proxyAddr, proxyUser, proxyPass)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: newHTTPTransport(proxyFn),
	}, nil
}

func buildProxyFunc(proxyAddr, proxyUser, proxyPass string) (func(*http.Request) (*url.URL, error), error) {
	trimmed := strings.TrimSpace(proxyAddr)
	if trimmed == "" {
		return http.ProxyFromEnvironment, nil
	}

	proxyURL := trimmed
	if !strings.HasPrefix(proxyURL, "http://") && !strings.HasPrefix(proxyURL, "https://") {
		proxyURL = "http://" + proxyURL
	}
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse proxy address: %w", err)
	}
	user := strings.TrimSpace(proxyUser)
	if parsed.User == nil && user != "" {
		parsed.User = url.UserPassword(user, proxyPass)
	}
	p := http.ProxyURL(parsed)
	return p, nil
}

func newHTTPTransport(proxy func(*http.Request) (*url.URL, error)) *http.Transport {
	dt, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t := &http.Transport{Proxy: proxy}
		forceHTTP11Only(t)
		return t
	}
	tr := dt.Clone()
	tr.Proxy = proxy
	forceHTTP11Only(tr)
	return tr
}

func forceHTTP11Only(t *http.Transport) {
	t.ForceAttemptHTTP2 = false
	if t.TLSClientConfig == nil {
		t.TLSClientConfig = &tls.Config{}
	} else {
		t.TLSClientConfig = t.TLSClientConfig.Clone()
	}
	t.TLSClientConfig.NextProtos = []string{"http/1.1"}
}
