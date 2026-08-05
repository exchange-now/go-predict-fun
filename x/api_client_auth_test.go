package x

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerTokenAndInvalidateAuth(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/message":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data":    map[string]string{"message": "sign-me"},
			})
		case "/v1/auth":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data":    map[string]string{"token": "jwt-abc"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := NewAPIClient(APIClientOptions{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := client.BearerToken(); got != "" {
		t.Fatalf("expected empty bearer, got %q", got)
	}
	if err := client.PostAuth(t.Context(), PostAuthRequest{
		Signer: "0x1", Message: "sign-me", Signature: "0xsig",
	}); err != nil {
		t.Fatal(err)
	}
	if got := client.BearerToken(); got != "jwt-abc" {
		t.Fatalf("bearer=%q want jwt-abc", got)
	}
	client.InvalidateAuth()
	if got := client.BearerToken(); got != "" {
		t.Fatalf("after invalidate got %q", got)
	}
}

func TestListOrdersUsesBearer(t *testing.T) {
	t.Parallel()
	var sawAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    []any{},
		})
	}))
	defer srv.Close()

	client, err := NewAPIClient(APIClientOptions{
		BaseURL:     srv.URL,
		BearerToken: "tok-1",
		HTTPClient:  srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListOrders(t.Context(), ListOrdersParams{Status: OrderListStatusOpen, First: "100"}); err != nil {
		t.Fatal(err)
	}
	if sawAuth != "Bearer tok-1" {
		t.Fatalf("Authorization=%q", sawAuth)
	}
}
