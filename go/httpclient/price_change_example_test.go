package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/change"
)

// ExamplePriceChangeClient_Get shows a multi-period request through the composition root.
//
// Version:
//   - 2026-10-19: Update required crypto asset selectors.
//   - 2026-10-18: Added.
func ExamplePriceChangeClient_Get() {
	client, err := httpclient.NewClient(httpclient.Config{
		EndpointURL: "https://api.k4k3ru.com/", CredentialProvider: environmentCredentialProvider{},
		HTTPClient: &http.Client{
			Timeout:       30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	result, err := client.MarketHub().Price().Change().Get(context.Background(), change.Params{AssetType: market.AssetClassCrypto,
		Asset: "sui", MarketType: "spot", Periods: []change.Period{change.Period5m, change.Period24h},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, value := range result.Changes {
		if value.ChangePercentage == nil {
			fmt.Println(value.Period, value.Status)
			continue
		}
		fmt.Println(value.Period, *value.ChangePercentage, "%", value.Status, "reference end:", *value.ReferenceAt)
	}
}
