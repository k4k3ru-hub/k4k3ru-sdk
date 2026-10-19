# Composite price ranges

`MarketHub.Price.Range.Get` is an HTTP snapshot contract for asset-level USD
High/Low values. Public types belong to this package (`pricerange`), with the
client composed by `httpclient.NewClient`:
`client.MarketHub().Price().Range().Get(ctx, params)`.

```json
{
  "id": "range-1",
  "method": "MarketHub.Price.Range.Get",
  "params": {
    "assetType": "crypto",
    "asset": "sui",
    "marketType": "spot",
    "periods": ["5m"]
  }
}
```

`assetType` is required and accepts only `crypto`. Its Go type is the existing
`finance/market.AssetClass`, using `market.AssetClassCrypto`; the other constants
in that shared enum are not supported by this API. Omitted, null, empty and
unsupported values are errors, including after `Normalize`.

`asset` is a canonical lowercase slug. `marketType` is `spot` or `perpetual`.
Supported periods are `5m`, `15m`, `1h`, `4h`, `24h`, `7d`; omitted, null or empty
periods select all six. Explicit periods must be distinct, and their order is
preserved in `ranges[]`. Days are elapsed 24-hour periods, not calendar days.

The Result contains `asset`, `marketType`, `definition: "composite"`,
`currency: "USD"`, `method: "volume_weighted_latest"`, `ranges[]`, and `sentAt`.
Each range contains:

| Field | Meaning |
| --- | --- |
| `period` | Requested elapsed period. |
| `low`, `high` | Positive decimal USD prices, including the observed developing minute; both null when unavailable. |
| `from` | Calculation start, `floor_minute(to - period)`, in Unix microseconds. Not the first available observation. |
| `to` | Common snapshot calculation time in Unix microseconds. Not the last observation or response send time. |
| `coveragePercentage` | Decimal percentage of usable **closed** minute candles, from `"0"` through `"100"`. |
| `coverageStatus` | `sufficient`, `insufficient`, or `unavailable`. This is not price freshness. |

The source is shared in-memory Composite one-minute history. Range reads require
neither a Parquet scan nor a Venue RPC. High/Low use actual usable observations;
missing minutes are not filled by carrying a prior close. One-minute OHLC cannot
be trimmed inside the leftmost minute. At 12:10:30, a 5m range therefore starts
at 12:05:00 and includes the developing 12:10 candle through calculation time.
The actual span can exceed the requested period by less than one minute.

Coverage counts the five closed minutes starting at 12:05 through 12:09 in that
example. The developing candle contributes only to High/Low. Missing minutes and
history not yet accumulated remain in the denominator. Coverage is not a claim
that every trade or every Venue was observed.

| Coverage status | Condition |
| --- | --- |
| `sufficient` | At least 50% of the expected closed minute candles are usable. |
| `insufficient` | Usable price data exists, but closed-minute coverage is below 50%. High/Low are still returned. |
| `unavailable` | No usable price candle exists, including the developing minute. High/Low are null and coverage is `"0"`. |

If only the developing candle is usable, return its High/Low, coverage `"0"` and
`insufficient`. The server determines status from the exact count ratio before
rounding. Percentages use up to 18 fractional decimal digits with nearest-even
rounding and trailing-zero removal. Zero is not a missing price.

All periods use one snapshot and calculation time. `sentAt` is the Gateway's
response send time in Unix microseconds and can be later than `to`. The client
validates metadata, calculation boundaries, coverage/price consistency and the
requested selectors/period order. It neither fills history nor retries requests.

This SDK contract does not imply that the server endpoint is deployed. Range
backend calculation, Gateway quota/billing policy and Console integration are
separate implementation steps.
