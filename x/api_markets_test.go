package x

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const sampleMarketsJSON = `{
  "success": true,
  "cursor": "NDcz",
  "data": [
    {
      "id": 473,
      "imageUrl": "https://static.predict.fail/example",
      "title": "Example Market",
      "question": "Will it happen?",
      "description": "Example description",
      "tradingStatus": "OPEN",
      "status": "REGISTERED",
      "isVisible": true,
      "isNegRisk": true,
      "isYieldBearing": false,
      "feeRateBps": 200,
      "oracleQuestionId": "0xabc",
      "conditionId": "0xdef",
      "resolverAddress": "0x52DA245ac170155391e7607c67b77D549005002d",
      "outcomes": [
        {
          "indexSet": 1,
          "name": "Yes",
          "onChainId": "79444321288970485377395452098264193036482524921501002353825929894382327552408",
          "bestAsk": {"price": 0.585, "size": 9825.7785},
          "bestBid": {"price": 0.584, "size": 98.32}
        }
      ],
      "spreadThreshold": 0.06,
      "shareThreshold": 100,
      "isBoosted": false,
      "polymarketConditionIds": [],
      "categorySlug": "example",
      "createdAt": "2025-12-05T13:36:54.550Z",
      "decimalPrecision": 3,
      "marketVariant": "DEFAULT",
      "rewards": {"current": null, "schedule": []}
    }
  ]
}`

func TestGetMarkets(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		if r.URL.Path != "/v1/markets" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Fatalf("missing api key header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleMarketsJSON))
	}))
	defer srv.Close()

	client := NewAPIClient(APIClientOptions{BaseURL: srv.URL, APIKey: "test-key"})
	first := "10"
	status := MarketStatusOpen
	sort := MarketSortVolume24hDesc
	hasRewards := true
	resp, err := client.GetMarkets(context.Background(), GetMarketsParams{
		First:            &first,
		Status:           &status,
		TagIDs:           []string{"1", "2"},
		Sort:             &sort,
		HasActiveRewards: &hasRewards,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatal("expected success")
	}
	if resp.Cursor == nil || *resp.Cursor != "NDcz" {
		t.Fatalf("cursor = %v", resp.Cursor)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len = %d", len(resp.Data))
	}
	m := resp.Data[0]
	if m.ID != 473 || m.FeeRateBps != 200 || !m.IsNegRisk {
		t.Fatalf("unexpected market: %+v", m)
	}
	if len(m.Outcomes) != 1 || m.Outcomes[0].OnChainID == "" {
		t.Fatalf("outcomes = %+v", m.Outcomes)
	}
	if m.Outcomes[0].BestAsk == nil || m.Outcomes[0].BestAsk.Price != 0.585 {
		t.Fatalf("bestAsk = %+v", m.Outcomes[0].BestAsk)
	}

	if gotQuery == "" {
		t.Fatal("expected query string")
	}
}

func TestGetMarketsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"code":401,"error":"unauthorized","message":"authorization error"}`))
	}))
	defer srv.Close()

	client := NewAPIClient(APIClientOptions{BaseURL: srv.URL})
	_, err := client.GetMarkets(context.Background(), GetMarketsParams{})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != 401 {
		t.Fatalf("code = %d", apiErr.Code)
	}
}

func TestGetMarketsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client := NewAPIClient(APIClientOptions{BaseURL: APIBaseURLTestnet})
	first := "1"
	resp, err := client.GetMarkets(context.Background(), GetMarketsParams{First: &first})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("expected at least one market")
	}

	raw, err := json.Marshal(resp.Data[0])
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip Market
	if err := json.Unmarshal(raw, &roundTrip); err != nil {
		t.Fatalf("round trip: %v", err)
	}
}
