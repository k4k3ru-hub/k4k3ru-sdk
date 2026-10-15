# Historical candles over HTTP

`httpclient.NewClient` is the composition root for HTTP JSON-RPC. It constructs
the request sender, MarketHub group and Candles operation once. It is separate
from `websocket.NewModule`; Historical candles use HTTP only.

```go
import (
    "context"
    "net/http"
    "time"

    "github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
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
