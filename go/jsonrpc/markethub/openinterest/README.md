# MarketHub.OpenInterest.Get

Current normalized Perpetual open interest, read from retained market snapshots.
Public Gateway availability depends on the server release; pricing and Console
integration are a separate rollout.

```json
{
  "assetType": "crypto",
  "symbol": "SUI/USDC",
  "markets": [
    {"venue": "hyperliquid", "network": "mainnet", "venueSymbol": "SUI"}
  ]
}
```

All three fields are required. `markets` uses `market.MarketTarget`, with 1–64
selectors. Each requires venue/network; chain, venueSymbol and poolId narrow
the scope. MarketHub resolves registered Perpetual instruments. Unknown targets
return `not_found`; overlapping selectors deduplicate the resulting markets.
No account, side, order quantity or marketType is accepted.

The response contains `symbol`, `interests[]` and `sentAt` (Unix microseconds,
Gateway send time). Each interest contains:

| Field | Meaning |
| --- | --- |
| market | Resolved market identity |
| status | live / stale / unavailable |
| quantity | Base-asset equivalent as `{amount, decimals}`, not raw contract count |
| notionalValue | Nonnegative decimal string in units of notionalCurrency |
| notionalCurrency | Native valuation currency; USD/USDC/USDT are not merged |
| valuation | mark / index / oracle / venue_reported |

The four value fields are present for live/stale and omitted for unavailable.
Zero quantity/value is valid. `amount / 10^decimals` is a quantity; the scale is
independent of token decimals. This is market-wide OI, not a trading account's
position or available liquidity.

Freshness uses the last successful receipt/validation: up to and including
120 seconds is live; older valid observations remain stale. Unchanged
observations refresh receipt time. Missing/invalid observations are unavailable.
`sentAt` is not an observation timestamp.

Hyperliquid's existing normalization uses OI quantity × the Mark price from
the same asset-context response. Other adapters retain their existing
normalized value/currency/valuation. The API does not revalue stale OI using
a newer Composite price, infer a USD conversion, sum venues or calculate
24h changes. Bluefin requires a separate collector connection.

MarketHub reads memory without request-time Venue calls or Parquet access.
The current internal cache uses float64; decimal encoding preserves its
represented value and does not add precision absent from the collector.

Use the owning package and the HTTP composition root:

```go
result, err := client.MarketHub().OpenInterest().Get(ctx, openinterest.Params{
    AssetType: market.AssetClassCrypto,
    Symbol: "SUI/USDC",
    Markets: []market.MarketTarget{
        {Venue: market.Hyperliquid, Network: "mainnet", VenueSymbol: "SUI"},
    },
})
```

Imports: `finance/market`, `jsonrpc/markethub/openinterest` and `httpclient`
under `github.com/k4k3ru-hub/k4k3ru-sdk/go`. The client validates response scope,
preserves error causes, performs no automatic retry, and leaves input unchanged.
