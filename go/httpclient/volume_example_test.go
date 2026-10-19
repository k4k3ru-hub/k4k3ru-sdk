package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/volume"
)

// ExampleVolumeClient_Get shows a multi-period volume request through the composition root.
//
// Version:
//   - 2026-10-19: Added.
func ExampleVolumeClient_Get() {
	client, err := httpclient.NewClient(httpclient.Config{
		EndpointURL: "https://api.k4k3ru.com/", CredentialProvider: environmentCredentialProvider{},
		HTTPClient: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	result, err := client.MarketHub().Volume().Get(context.Background(), volume.Params{
		AssetType: market.AssetClassCrypto, Asset: "sui", MarketType: market.MarketTypeSpot,
		Periods: []volume.Period{volume.Period5m, volume.Period24h},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, r := range result.Volumes {
		if r.Volume == nil {
			fmt.Println(r.Period, r.CoverageStatus)
			continue
		}
		fmt.Println(r.Period, *r.Volume, r.CoveragePercentage+"%", r.CoverageStatus)
	}
}
