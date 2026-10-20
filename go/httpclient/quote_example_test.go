package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/quote"
)

// ExampleQuoteClient_Get requests estimates through the composition root and filters full-quantity results.
//
// Version:
//   - 2026-10-20: Added.
func ExampleQuoteClient_Get() {
	client, err := httpclient.NewClient(httpclient.Config{
		EndpointURL: "https://api.k4k3ru.com/", CredentialProvider: environmentCredentialProvider{},
		HTTPClient: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	result, err := client.MarketHub().Quote().Get(context.Background(), quote.Params{
		AssetType: market.AssetClassCrypto, MarketType: market.MarketTypeSpot,
		Symbol: "SUI/USDC", Side: quote.SideBuy, Quantity: market.Quantity{Amount: "100", Decimals: 0},
		Markets: []market.MarketTarget{{Venue: market.Cetus, Chain: "sui", Network: "mainnet"}, {Venue: market.Turbos, Chain: "sui", Network: "mainnet"}},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, q := range result.Quotes {
		if q.Status != quote.StatusVWAP {
			fmt.Println(q.Market.Venue, q.Status)
			continue
		}
		fmt.Println(q.Market.Venue, *q.NetPrice, *q.NetReceiveQuantity)
	}
}
