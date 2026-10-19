package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/candles"
)

// ExampleCandlesClient_Get_market selects a concrete market and refreshable current candle.
//
// Version:
//   - 2026-10-19: Added.
func ExampleCandlesClient_Get_market() {
	client, err := httpclient.NewClient(httpclient.Config{
		EndpointURL:        "https://api.k4k3ru.com/",
		CredentialProvider: environmentCredentialProvider{},
		HTTPClient:         &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	minute := time.Now().UTC().Truncate(time.Minute)
	params := candles.Params{
		AssetType: market.AssetClassCrypto, Asset: "sui", MarketType: "perp",
		Market: &market.MarketTarget{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: "SUI"},
		From:   minute.Add(-time.Minute).Unix(), To: minute.Add(time.Minute).Unix(),
		Interval: "1m", IncludeCurrent: true,
	}
	page, err := client.MarketHub().Candles().Get(context.Background(), params)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, bar := range page.Candles {
		// Replace chart values by market and bar.Time, including full volume totals.
		fmt.Println(page.Symbol, bar.Time, bar.Closed, bar.Revision)
	}
}
