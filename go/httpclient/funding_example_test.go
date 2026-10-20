package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/funding"
)

// ExampleFundingClient_Get reads current estimates through the composition root.
//
// Version:
//   - 2026-10-20: Added.
func ExampleFundingClient_Get() {
	client, err := httpclient.NewClient(httpclient.Config{
		EndpointURL: "https://api.k4k3ru.com/", CredentialProvider: environmentCredentialProvider{},
		HTTPClient: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	// Requires a server release enabling public Funding.Get.
	result, err := client.MarketHub().Funding().Get(context.Background(), funding.Params{
		AssetType: market.AssetClassCrypto, Symbol: "SUI/USDC",
		Markets: []market.MarketTarget{{Venue: market.Hyperliquid, Network: "mainnet"}},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, rate := range result.Rates {
		if rate.Status == funding.StatusUnavailable {
			fmt.Println(rate.Market.Venue, rate.Status)
			continue
		}
		fmt.Println(rate.Market.Venue, rate.Status, *rate.Rate, *rate.IntervalMinutes)
	}
}
