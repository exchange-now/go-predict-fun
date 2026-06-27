package x

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestGetMarkets2(t *testing.T) {
	client := NewAPIClient(APIClientOptions{BaseURL: APIBaseURLTestnet})
	first := "1"
	resp, err := client.GetMarkets(context.Background(), GetMarketsParams{First: &first})
	require.NoError(t, err)
	if len(resp.Data) == 0 {
		t.Fatal("expected at least one market")
	}

	raw, err := json.Marshal(resp.Data[0])
	require.NoError(t, err)

	var roundTrip Market
	if err := json.Unmarshal(raw, &roundTrip); err != nil {
		require.NoError(t, err)
	}
	//fmt.Printf("roundTrip: %+v\n", roundTrip)
	spew.Dump(roundTrip)
}
