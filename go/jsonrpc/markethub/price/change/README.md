# Composite price changes

`MarketHub.Price.Change.Get` is an HTTP snapshot contract for one canonical asset
and one market type. Types belong to `jsonrpc/markethub/price/change`; market types
come from `finance/market`. This SDK provides the contract and HTTP client. The
MarketHub calculation and Gateway/Console rollout are a separate implementation
step; publishing this SDK does not deploy the endpoint.

## Request

```json
{
  "id": "price-change-1",
  "method": "MarketHub.Price.Change.Get",
  "params": {
    "assetType": "crypto",
    "asset": "sui",
    "marketType": "spot",
    "periods": ["5m", "15m", "24h"]
  }
}
```

- `assetType` is required and must be `crypto`; the Go type is
  `finance/market.AssetClass` (`market.AssetClassCrypto`). Omission, null, empty
  and unsupported values are errors. Normalize does not supply a default.
- `asset` and `marketType` use the same canonical selectors as `Price.Get`:
  lowercase asset slug and `spot` or `perpetual`. `perp` is not accepted.
- Supported periods are `5m`, `15m`, `1h`, `4h`, `24h`, `7d`; these are elapsed
  durations, not calendar boundaries. Omitted, empty or null `periods` selects
  all six in that order. An explicit list may contain each period once, at most
  six entries. Unsupported periods and duplicates return invalid-parameter errors.
- `Params.Normalize` copies the period slice and supplies defaults; it does not
  coerce invalid selectors or remove unsupported/duplicate periods. Validate
  before normalization. The HTTP client sends the normalized selection once.

## Result

```json
{
  "asset": "sui",
  "marketType": "spot",
  "definition": "composite",
  "currency": "USD",
  "method": "volume_weighted_latest",
  "price": "1.02",
  "changes": [
    {
      "period": "5m",
      "changePercentage": "2",
      "referencePrice": "1",
      "referenceAt": 1792326600000000,
      "status": "live"
    },
    {
      "period": "15m",
      "changePercentage": "6.25",
      "referencePrice": "0.96",
      "referenceAt": 1792325880000000,
      "status": "stale"
    },
    {
      "period": "24h",
      "changePercentage": null,
      "referencePrice": null,
      "referenceAt": null,
      "status": "unavailable"
    }
  ],
  "sentAt": 1792326930000000
}
```

The example values are illustrative. `price` is the one current Composite USD
price used for all periods, or null when unknown. Numerical values are decimal
strings; `changePercentage: "2"` means +2%, not 2bps or a ratio of 2. Consumers
identify entries by `period`; the client accepts any response order but requires
exactly the selected set, including unavailable periods.

All periods use one server calculation time `T`. The server selects the latest
usable completed one-minute Composite candle whose **end is at or before
`T - period`**. If it is absent or unusable, the search moves backward through
retained same-method history. A later candle, a developing candle, or an old
five-second VWAP series must not substitute for the reference.

```
changePercentage = (price / referencePrice - 1) * 100
```

`referenceAt` is the chosen candle's **end**, not its start or last trade time.
Both `referenceAt` and `sentAt` are UTC Unix **microseconds**. `sentAt` is the
Gateway send time, not the server calculation time or the price observation time.
For example, at 12:35:30 the 5m target is 12:30:30; the normal reference ends at
12:30:00. The candle ending at 12:31:00 includes prices after the target and
cannot be used. A missing 12:30 candle can fall back to one ending at 12:29 or
earlier. The actual reference price and end remain visible in the result.

| Status | Meaning |
| --- | --- |
| `live` | Current price is live and the normal reference candle is usable. |
| `stale` | Current price is stale, or the normal reference is missing/unusable and an older candle was selected. |
| `unavailable` | A required price is missing, or there is no valid price observation in the requested period. No percentage is returned. |

Gaps inside a period do not themselves prevent endpoint-price comparison when
the required prices and at least one observation in that period exist. They do
not imply continuous coverage. An unavailable percentage is null, never zero.
A known reference may still be returned as a price/time pair if the percentage
is unavailable; absent reference fields are both null. Available percentages
require a positive current price and a positive reference price.

The SDK validates shape, decimal syntax, reference time bounds and matching
selectors. It cannot independently verify observation coverage or distinguish
fresh from stale prices, and it does not recompute or round the percentage.
The reference may not be later than `sentAt - period`; the server must enforce
the stronger bound using its own calculation time.

## Data and integration

The agreed server inputs are the current in-memory Composite snapshot and shared
closed one-minute Composite history. Normal requests perform no Venue RPC or
Parquet reads. Background restoration supplies durable same-method history.
The seven-day boundary requires retaining a preceding reference close as well as
the seven-day display window; that cache change belongs to the backend step.
If no suitable reference is retained, the period is unavailable.

Use `httpclient.NewClient`, then
`client.MarketHub().Price().Change().Get(ctx, params)`. The operation uses the
existing injected HTTP transport, clock and credential provider. It sends one
request without retries, polling or separate candle requests. Gateway access,
quotas and credit prices for this new method remain to be finalized; the
`Price.Get` tariff is not part of this SDK contract.
