package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/candles"
)

// ExampleCandlesClient_Get shows one HTTP request and the next range position.
//
// Version:
//   - 2026-10-11: Added.
func ExampleCandlesClient_Get() {
	client, err := httpclient.NewClient(httpclient.Config{
		EndpointURL: "https://api.k4k3ru.com/",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	params := candles.Params{Asset: "btc", MarketType: "spot", From: 1791331200, To: 1791417600, Interval: "1m", Limit: 300}
	page, err := client.MarketHub().Candles().Get(context.Background(), params)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("returned candles:", len(page.Candles))
	if page.To < params.To {
		params.From = page.To
		fmt.Println("next from:", params.From)
	}
}
