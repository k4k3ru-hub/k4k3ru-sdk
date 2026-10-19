package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	pricerange "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/range"
)

// ExamplePriceRangeClient_Get shows a bounded range request through the composition root.
//
// Version:
//   - 2026-10-19: Added.
func ExamplePriceRangeClient_Get() {
	client, err := httpclient.NewClient(httpclient.Config{
		EndpointURL: "https://api.k4k3ru.com/", CredentialProvider: environmentCredentialProvider{},
		HTTPClient: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	result, err := client.MarketHub().Price().Range().Get(context.Background(), pricerange.Params{
		AssetType: market.AssetClassCrypto, Asset: "sui", MarketType: market.MarketTypeSpot,
		Periods: []pricerange.Period{pricerange.Period5m, pricerange.Period24h},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, r := range result.Ranges {
		if r.Low == nil {
			fmt.Println(r.Period, r.CoverageStatus)
			continue
		}
		fmt.Println(r.Period, *r.Low, *r.High, r.CoveragePercentage+"%", r.CoverageStatus)
	}
}
