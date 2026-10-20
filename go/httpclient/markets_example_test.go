package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/markets"
)

// ExampleMarketsClient_List discovers one production market page through the composition root.
//
// Version:
//   - 2026-10-20: Added.
func ExampleMarketsClient_List() {
	client, err := httpclient.NewClient(httpclient.Config{
		EndpointURL: "https://api.k4k3ru.com/", CredentialProvider: environmentCredentialProvider{},
		HTTPClient: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	// Requires a server release enabling Markets.List; pricing is reviewed separately.
	result, err := client.MarketHub().Markets().List(context.Background(), markets.Params{
		AssetType: market.AssetClassCrypto, Asset: "sui", MarketType: market.MarketTypePerpetual,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, entry := range result.Markets {
		fmt.Println(entry.Market.Venue, entry.Symbol, entry.Market.VenueSymbol, entry.CompositeEligible, entry.CandlesSupported)
	}
}
