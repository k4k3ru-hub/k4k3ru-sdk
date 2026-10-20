# Current estimated funding rates

`MarketHub.Funding.Get` returns venue-specific perpetual funding estimates.
Types belong to this package; use
`client.MarketHub().Funding().Get(ctx, params)` through `httpclient.NewClient`.

The SDK contract and MarketHub memory reader are implemented. Public Gateway
admission, pricing/limits, send-time stamping and Console integration are pending.
This does not imply the method is deployed or publicly callable yet.

```json
{
  "method": "MarketHub.Funding.Get",
  "params": {
    "assetType": "crypto",
    "symbol": "SUI/USDC",
    "markets": [
      {"venue": "hyperliquid", "network": "mainnet"}
    ]
  }
}
```

- All three top-level fields are required. `assetType` accepts only `crypto`.
- `symbol` is a canonical Base/Quote pair, normalized to uppercase.
- `markets` accepts 1–64 distinct `finance/market.MarketTarget` scopes.
  Venue and network are required. Chain and native instrument filters are optional.
  Without `venueSymbol`, registered perpetual instruments for that pair are resolved.
- Market type is always Perpetual. Quantity, account and side are not inputs.
- Unknown/duplicate fields, null required fields and invalid selectors are rejected.
- The service rejects an unresolved scope with `not_found`, as with Quote.Get;
  it does not silently omit the target or fabricate a native instrument.
- Overlapping scopes may resolve the same instrument, returned once. The resolved
  result is bounded to 64 markets.

Illustrative result (not live data):

```json
{
  "symbol": "SUI/USDC",
  "rates": [
    {
      "market": {
        "venue": "hyperliquid",
        "network": "mainnet",
        "venueSymbol": "SUI"
      },
      "status": "live",
      "kind": "current_estimate",
      "rate": "0.000012",
      "intervalMinutes": 60,
      "fundingAt": 1792454400000000
    }
  ],
  "sentAt": 1792450800000000
}
```

`rate` is a signed decimal **ratio**, not percent: `0.000012` means **0.0012%**
per `intervalMinutes`. Positive means longs pay shorts; negative means shorts
pay longs. Zero is valid. No implicit hourly normalization or annualization occurs.

Only `current_estimate` observations are returned. A settled historical rate
never substitutes for a missing estimate. `fundingAt` is the target settlement
time when known, in Unix microseconds; it is not a guarantee of that payment.
`sentAt` is the Gateway send time in Unix microseconds, not data freshness.

| Status | Meaning |
|---|---|
| `live` | Last successful receipt/validation is at most 120 seconds old and the known target settlement time is still in the future. |
| `stale` | A valid estimate exists but is older than 120 seconds or its target settlement time has arrived. Keep its original rate, interval and target time. |
| `unavailable` | No valid current estimate exists. Omit `kind`, `rate`, `intervalMinutes` and `fundingAt`. |

At exactly 120 seconds the age criterion remains live. At exactly `fundingAt`
the estimate becomes stale. A valid stale record has no age-based expiry.
Unknown target time is omitted and age still determines status. A future receipt
timestamp, nonfinite rate, nonpositive interval or conflicting instrument is invalid.

The service reads retained memory only: no request-time Venue RPC, historical
Parquet lookup or account fee retrieval. Mark/index prices and order-book readiness
are not prerequisites for returning a funding rate. Mainnet and Testnet are isolated.
After a process restart, observations must be fetched by the existing background
collectors; this API does not restore estimates from history.
