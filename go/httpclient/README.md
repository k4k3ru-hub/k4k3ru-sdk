# MarketHub over HTTP

`httpclient.NewClient` is the composition root for HTTP JSON-RPC. It constructs
the request sender, MarketHub group, Candles, Price, Price.Change, Price.Range, Volume and Quote operations
once. It is separate from `websocket.NewModule`; Historical candles use HTTP only.

```go
import (
    "context"
    "net/http"
    "time"

    "github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
    "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
    "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/candles"
)

client, err := httpclient.NewClient(httpclient.Config{
    EndpointURL: "https://api.k4k3ru.com/",
    HTTPClient: &http.Client{
        Timeout: 30 * time.Second,
        CheckRedirect: func(*http.Request, []*http.Request) error {
            return http.ErrUseLastResponse
        },
    },
    CredentialProvider: provider, // Your authentication.CredentialProvider.
})
if err != nil {
    return err
}

params := candles.Params{
    AssetType: market.AssetClassCrypto,
    Asset: "btc", MarketType: "spot",
    From: 1791331200, To: 1791417600, // 2026-10-07 UTC, [from,to).
    Interval: "1m", Limit: 300,
}
page, err := client.MarketHub().Candles().Get(context.Background(), params)
if err != nil {
    return err
}
// Process page.Candles and retain quality/revision evidence.
if page.To < params.To {
    params.From = page.To // Next request, after applying your quota/credit policy.
}
```

The caller owns the injected `HTTPDoer` and its timeout, connection and redirect
policies. A standard `*http.Client` works; disable redirects when signing requests
so credentials remain at the configured endpoint. Historical calls require an
`authentication.CredentialProvider`; existing HMAC-SHA256 and Ed25519 credentials
are supported. The provider is called for each request. Supplying a provider that
fails does not fall back to anonymous access. `Now` and `Nonce` can be injected
for tests; their defaults are the system clock and secure nonce generator.
Injected dependencies must support concurrent use if the client is shared.

Public types belong to [`jsonrpc/markethub/candles`](../jsonrpc/markethub/candles/README.md).
There are no root aliases. The method constant is `jsonrpc.MethodMarketHubCandlesGet`.
No new production dependency is required.

`Get` performs one request, with no automatic retries or page fetching. Every
request counts toward the Gateway quota of 60 per rolling minute per account and
method. Authentication is required; there is no free allowance. Each successful
request costs 1,000 credits, including an empty result. A full day at 1m requires
five requests when all 1,440 candles exist, costing 5,000 credits. Gateway confirms
the debit before reading data and returns `out_of_ticks` when funds are insufficient.
Upstream failures enqueue a refund. A nil credential provider leaves the generic
transport unsigned; the Historical endpoint rejects those requests as `unauthorized`.

Application errors remain inspectable with `errors.Is`/`errors.As`. Non-2xx HTTP
responses return `*httpclient.HTTPError`, with `StatusCode` and `RetryAfter` (the raw
header, such as `"60"`), and unwrap the server application error when present.
Malformed/mismatched envelopes, invalid continuation bounds and responses over
2 MiB are rejected. The response-size bound applies after HTTP decompression.
Errors do not include raw response bodies or request authentication payloads.

Candles can also select one Mainnet DEX market and include the current interval:

```go
params.Asset = "sui"
params.Market = &market.MarketTarget{
    Venue: "cetus", Chain: "sui", Network: "mainnet", PoolID: "<pool-id>",
}
params.IncludeCurrent = true
// Keep From/To aligned to Interval. To may reach the current interval's end.
page, err = client.MarketHub().Candles().Get(ctx, params)
```

`market` omitted selects Composite. Without `poolId`/`venueSymbol`, the server
requires a unique match; ambiguous markets are errors. Refresh recent intervals
with new requests and replace bars by start time, rather than summing their volume.
`Closed` separates developing and closed intervals; `SentAt` is microseconds.
Authentication and the 1,000-credit charge remain applicable to every successful call.

The client and contract have fake-transport and composition tests. The compiled
[`ExampleCandlesClient_Get`](example_test.go) uses the same composition root.

## Current price

Import `github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price` and use the
same composed client:

```go
snapshot, err := client.MarketHub().Price().Get(ctx, price.Params{
    AssetType: market.AssetClassCrypto,
    Asset: "sui", MarketType: "spot",
})
if err != nil {
    return err
}
// snapshot.Price can be nil; snapshot.SentAt is Unix microseconds.
```

Unlike Candles, Price permits an omitted CredentialProvider. Its quota and
charging rules are documented in [the price contract](../jsonrpc/markethub/price/README.md).

## Price changes

Import `github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/change`:

```go
changes, err := client.MarketHub().Price().Change().Get(ctx, change.Params{
    AssetType: market.AssetClassCrypto,
    Asset: "sui", MarketType: "spot",
    Periods: []change.Period{change.Period5m, change.Period15m, change.Period24h},
})
if err != nil {
    return err
}
// changes.Price is common to all periods. Each entry has its own status,
// nullable changePercentage, referencePrice and referenceAt (Unix microseconds).
```

Omitting `Periods` selects all six supported periods. One HTTP call retrieves the
entire selection; there is no automatic retry or candle fetch. The client verifies
that the response includes each requested period exactly once, independent of
response order. The [contract](../jsonrpc/markethub/price/change/README.md) documents
backward reference selection and partial availability. The compiled
[`ExamplePriceChangeClient_Get`](price_change_example_test.go) uses the same
composition root. Server deployment and the new method's access/credit policy
are separate from this SDK implementation.

## Price ranges and required asset type

Price.Get, Price.Change.Get, Price.Range.Get and Candles.Get require
`AssetType: market.AssetClassCrypto` (`"assetType":"crypto"` on the wire).
Missing, null, empty or unsupported types are rejected before HTTP is called.
There is no default. Existing requests must add this field before migrating to
the corresponding server version. Result shapes for the existing APIs do not
change. No existing `assetClass` fields on other APIs are renamed.

Import `pricerange "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/range"`:

```go
ranges, err := client.MarketHub().Price().Range().Get(ctx, pricerange.Params{
    AssetType: market.AssetClassCrypto,
    Asset: "sui", MarketType: market.MarketTypeSpot,
    Periods: []pricerange.Period{pricerange.Period5m, pricerange.Period24h},
})
if err != nil {
    return err
}
// ranges.Ranges includes High/Low even when CoverageStatus is insufficient.
// Low/High are nil only when unavailable. All timestamps use Unix microseconds.
```

See the [range contract](../jsonrpc/markethub/price/range/README.md) and compiled
[`ExamplePriceRangeClient_Get`](price_range_example_test.go). Range server
deployment and access/credit policy are not part of this SDK implementation.


## Observed USD volume

`client.MarketHub().Volume().Get(ctx, volume.Params{...})` calls
`MarketHub.Volume.Get` once for the selected periods. Types belong to
`github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/volume`.
The constructor composes this client with the same injected transport and
credential provider; requests do not retry automatically.

```go
volumes, err := client.MarketHub().Volume().Get(ctx, volume.Params{
    AssetType: market.AssetClassCrypto,
    Asset: "sui", MarketType: market.MarketTypeSpot,
    Periods: []volume.Period{volume.Period5m, volume.Period24h},
})
if err != nil {
    return err
}
for _, v := range volumes.Volumes {
    if v.Volume == nil {
        continue // No usable turnover or verified idle observation.
    }
    // *v.Volume is observed USD turnover, not an estimate of missing markets.
    // Retain v.CoverageStatus and v.CoveragePercentage with the value.
}
```

`assetType` is required and currently accepts only `crypto`. `asset` uses the
same lowercase canonical slug as Price APIs, e.g. `sui`; `marketType` accepts
`spot` or `perpetual`. Omitted periods select `5m`, `15m`, `1h`, `4h`, `24h`,
and `7d`. Explicit periods retain their order; duplicates are invalid.

- `method` is `sum`, `definition` is `composite`, and `currency` is `USD`.
- `from` is `floorMinute(to - period)`. Each period includes retained developing
  amounts up to the common evaluation time, so the interval may be less than one
  minute longer than its nominal duration. `from`, `to` and `sentAt` are Unix μs;
  `sentAt` is the Gateway send time, not observation or evaluation time.
- Each source trade contributes once. Supported CEX/DEX markets and pools are
  summed; Perpetual turnover is notional, not margin or open interest.
- USD conversion uses the stored trade-time valuation under existing supported
  quote policies. No current-price revaluation or missing-volume extrapolation.
- Coverage counts only closed minutes in which **all then-expected markets**
  have valid observation and USD conversion evidence. Developing minutes do not
  increase coverage. `sufficient` starts at 50%; it does not mean complete.
- Partial amounts remain available with `insufficient`. An observed zero is
  `"0"`; missing evidence is `null` with `unavailable`. A zero price-candle volume
  by itself does not establish that no trades occurred.
- Input source-minute amounts retain the existing floating-point aggregation
  precision; the decimal wire representation does not add accounting precision.

See [`ExampleVolumeClient_Get`](volume_example_test.go) for a compiled example.
The SDK contract and MarketHub implementation are staged. Public Gateway
admission, pricing/limits and Console integration await the Volume policy review;
this does not imply that the operation is enabled at a deployed endpoint.

## Quantity-specific quotes

`client.MarketHub().Quote().Get(ctx, params)` calls `MarketHub.Quote.Get` once.
Import request/response types from their owning package:

```go
import "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/quote"

estimates, err := client.MarketHub().Quote().Get(ctx, quote.Params{
    AssetType: market.AssetClassCrypto,
    MarketType: market.MarketTypeSpot,
    Symbol: "SUI/USDC", Side: quote.SideBuy,
    Quantity: market.Quantity{Amount: "100", Decimals: 0},
    Markets: []market.MarketTarget{
        {Venue: market.Cetus, Chain: "sui", Network: "mainnet"},
        {Venue: market.Turbos, Chain: "sui", Network: "mainnet"},
    },
})
if err != nil {
    return err
}
for _, q := range estimates.Quotes {
    if q.Status != quote.StatusVWAP {
        continue // Reference-only and unavailable markets cannot be the best full-quantity estimate.
    }
    // Spot: netPayQuantity and netReceiveQuantity are present.
    // netPrice and these quantities already reflect trading fees; gas is excluded.
}
```

Spot Buy quantity is Quote payment; Spot Sell quantity is Base payment.
For Perpetual, both sides specify Base-denominated contract quantity, and the
response omits payment/receipt token quantities. Quantity is not margin or leverage.
`sentAt` is the Gateway send time in Unix microseconds, not market freshness.

The client validates status-dependent fields, echoed selectors/quantity and
resolved market scope; it does not rank, retry, build transactions or send orders.
Constructor composition performs no network I/O. Inject the same HTTP transport
and credential provider used by the other operation groups.

See the [contract and JSON examples](../jsonrpc/markethub/quote/README.md) and
[`ExampleQuoteClient_Get`](quote_example_test.go). The SDK and application
integration are implemented. SDK publication, dependency updates and Docker rollout
remain separate steps; SDK availability does not enable a deployed endpoint.
