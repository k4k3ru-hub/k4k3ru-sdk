# Retained asset overview

`MarketHub.Asset.Overview.Get` reads one canonical crypto asset with both registered
market types. This SDK addition is unpublished; deployment support must be checked
before calling the public service.

```go
result, err := client.MarketHub().Asset().Overview().Get(ctx, overview.Params{
    AssetType: "crypto",
    Asset:     "sui",
})
```

Types belong to `jsonrpc/markethub/asset/overview`. The HTTP client validates strict
selectors, response scope, production identities, paired current price, decimal
and availability relationships, original timestamps, section states and the 4 MiB
response limit. It does not recompute financial metrics or retry automatically.

Groups are ordered spot, then perpetual. Sections contain `state`, nullable
`reason`, original `calculatedAt` and `expiresAt`, and nullable typed `data`.
Computation state (`ready`, `stale`, `unavailable`, `not_applicable`), per-point
source status and historical coverage describe separate properties. Point values
also retain `observedAt`, exclusive `freshUntil` and exclusive `expiresAt`.

The server attempts generation every 60 seconds. Within a running process,
fallback expires within 24 hours of the original calculation; point values are
also bounded by their original observation time plus 24 hours. Reads do not
restart those deadlines. Authoritative withdrawals can remove values sooner.
After restart persisted numerical values are unavailable until current-process
verified captures succeed; safe registered metadata may recover independently.

Anonymous reads have a 3-per-60-second IP/method allowance. Authenticated reads
share a 60-per-60-second account/method allowance across assets and API keys, with
the first 3 free and later reads costing 200 credits. Successful partial/stale
responses have the same price. Existing Asset.Get remains a separate metadata
operation with its 100-credit policy. No public Overview.List is added.
