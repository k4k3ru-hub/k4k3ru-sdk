# MarketHub over HTTP

`httpclient.NewClient` is the composition root for HTTP JSON-RPC. It constructs
the request sender, MarketHub group, Candles, Price, Price.Change and Price.Range operations
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
