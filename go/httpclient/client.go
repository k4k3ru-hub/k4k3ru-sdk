// Package httpclient provides composed clients for K4K3RU HTTP JSON-RPC operations.
package httpclient

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/signature"
)

// HTTPDoer is the HTTP dependency used by the client. Implementations must be
// safe for concurrent calls and should not retry billable requests automatically.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Config struct {
	EndpointURL        string
	HTTPClient         HTTPDoer
	CredentialProvider authentication.CredentialProvider // Nil selects anonymous access.
	Now                func() time.Time                  // Nil selects time.Now.
	Nonce              func() (string, error)            // Nil selects a secure random nonce.
}

type Client struct {
	marketHub *MarketHubClient
}

type MarketHubClient struct {
	candles      *CandlesClient
	price        *PriceClient
	volume       *VolumeClient
	quote        *QuoteClient
	funding      *FundingClient
	openInterest *OpenInterestClient
}

type endpointParseError struct{ cause error }

// Error describes an invalid endpoint without echoing URL credentials or query parameters.
//
// Version:
//   - 2026-10-11: Added.
func (e endpointParseError) Error() string {
	return "failed to parse http endpoint: endpoint_url=invalid"
}

// Unwrap preserves the parse error for inspection without rendering its sensitive URL.
//
// Version:
//   - 2026-10-11: Added.
func (e endpointParseError) Unwrap() error { return e.cause }

// NewClient composes the HTTP transport, signer and MarketHub operation clients.
// The caller supplies and owns the HTTP client, including its timeout policy.
// This constructor does not make requests. Injected dependencies must support
// concurrent use when the resulting client is shared between goroutines.
//
// Version:
//   - 2026-10-20: Compose current open interest with the shared transport.
//   - 2026-10-20: Compose current estimated funding with the shared transport.
//   - 2026-10-20: Compose the quantity-specific quote client with the shared transport.
//   - 2026-10-19: Compose price-range and volume clients with the shared transport.
//   - 2026-10-18: Compose current-price and price-change clients with the shared transport.
//   - 2026-10-11: Added.
func NewClient(config Config) (*Client, error) {
	u, err := url.Parse(config.EndpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", endpointParseError{cause: err})
	}
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("failed to create http client: endpoint_url=invalid")
	}
	if config.HTTPClient == nil {
		return nil, fmt.Errorf("failed to create http client: http_client=null")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Nonce == nil {
		config.Nonce = signature.GenerateNonce
	}
	sender := &requestSender{endpoint: u.String(), http: config.HTTPClient, credentials: config.CredentialProvider, now: config.Now, nonce: config.Nonce}
	candles, err := newCandlesClient(sender)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}
	change, err := newPriceChangeClient(sender)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}
	ranges, err := newPriceRangeClient(sender)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}
	price, err := newPriceClient(sender, change, ranges)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}
	volume, err := newVolumeClient(sender)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}
	quote, err := newQuoteClient(sender)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}
	funding, err := newFundingClient(sender)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}
	openInterest, err := newOpenInterestClient(sender)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}
	return &Client{marketHub: &MarketHubClient{candles: candles, price: price, volume: volume, quote: quote, funding: funding, openInterest: openInterest}}, nil
}

// Funding returns the current estimated perpetual funding operation client.
//
// Version:
//   - 2026-10-20: Added.
func (c *MarketHubClient) Funding() *FundingClient {
	if c == nil {
		return nil
	}
	return c.funding
}

// Price returns the Composite price group with current-price, change and range operations.
//
// Version:
//   - 2026-10-19: Include the composed price-range operation.
//   - 2026-10-18: Include the composed price-change operation.
func (c *MarketHubClient) Price() *PriceClient {
	if c == nil {
		return nil
	}
	return c.price
}

// MarketHub returns the composed MarketHub HTTP API group.
//
// Version:
//   - 2026-10-11: Added.
func (c *Client) MarketHub() *MarketHubClient {
	if c == nil {
		return nil
	}
	return c.marketHub
}

// Candles returns the historical Composite candle operation client.
//
// Version:
//   - 2026-10-11: Added.
func (c *MarketHubClient) Candles() *CandlesClient {
	if c == nil {
		return nil
	}
	return c.candles
}

// Volume returns the observed USD volume operation client.
//
// Version:
//   - 2026-10-19: Added.
func (c *MarketHubClient) Volume() *VolumeClient {
	if c == nil {
		return nil
	}
	return c.volume
}

// Quote returns the quantity-specific market estimate operation client.
//
// Version:
//   - 2026-10-20: Added.
func (c *MarketHubClient) Quote() *QuoteClient {
	if c == nil {
		return nil
	}
	return c.quote
}

// OpenInterest returns the current perpetual open interest operation client.
//
// Version:
//   - 2026-10-20: Added.
func (c *MarketHubClient) OpenInterest() *OpenInterestClient {
	if c == nil {
		return nil
	}
	return c.openInterest
}
