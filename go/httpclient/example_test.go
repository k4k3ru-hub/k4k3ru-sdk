package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/candles"
)

type environmentCredentialProvider struct{}

// Credential loads the caller's signing credentials from environment variables.
//
// Version:
//   - 2026-10-12: Added.
func (environmentCredentialProvider) Credential(context.Context) (authentication.Credential, error) {
	credential := authentication.Credential{
		APIKey: os.Getenv("K4K3RU_API_KEY"), SecretKey: os.Getenv("K4K3RU_SECRET_KEY"),
		SignatureAlgorithm: authentication.SignatureAlgorithm(os.Getenv("K4K3RU_SIGNATURE_ALGORITHM")),
	}
	if err := credential.Validate(); err != nil {
		return authentication.Credential{}, fmt.Errorf("failed to load example credential: %w", err)
	}
	return credential, nil
}

// ExampleCandlesClient_Get shows one HTTP request and the next range position.
//
// Version:
//   - 2026-10-19: Update required crypto asset selectors.
//   - 2026-10-12: Require a credential provider for paid Historical requests.
//   - 2026-10-11: Added.
func ExampleCandlesClient_Get() {
	client, err := httpclient.NewClient(httpclient.Config{
		EndpointURL:        "https://api.k4k3ru.com/",
		CredentialProvider: environmentCredentialProvider{},
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
	params := candles.Params{AssetType: market.AssetClassCrypto, Asset: "btc", MarketType: "spot", From: 1791331200, To: 1791417600, Interval: "1m", Limit: 300}
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
