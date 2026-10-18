# MarketHub.Candles.Get

The HTTP operation reads stored Composite candles. Its stable series identifier
is `composite`; calculation versions and Parquet paths are not request parameters.
Use [`httpclient.NewClient`](../../../httpclient/README.md) to compose the client,
then `client.MarketHub().Candles().Get(ctx, candles.Params{...})`.

Signed authentication is required: inject an `authentication.CredentialProvider`.
The account/method quota is 60 requests per rolling 60 seconds. Each successful
request costs 1,000 credits from the first call, including empty results and
continuation pages. There is no anonymous access or free allowance.

| Parameter | Contract |
| --- | --- |
| `asset` | Canonical lowercase asset slug, such as `btc`; required. |
| `marketType` | `spot` or `perp`; required. |
| `from`, `to` | Positive Unix **seconds**; half-open `[from,to)` range of at most 7 days, aligned to the interval. |
| `interval` | `1m`, `5m`, `15m`, `1h`, `4h`, `1d`, aligned in UTC. |
| `limit` | Optional integer, 1–300; omitted or zero means 300. Counts actual returned candles, including idle/unusable rows. |

The server rejects ranges containing unclosed minutes. `to` must be at or before
`floor((now - 2 seconds) / 60) * 60`, and aligned to the selected interval.
The `perp` spelling is part of this operation's contract.

Results are ordered by candle start. Save the original requested `to`, and set
the next request's `from` to **response `to`** until that original end is reached.
Missing intervals do not consume the limit and are never forward-filled. When
the limit is reached, `to` is the end of the final returned candle. Otherwise it
is the requested end, including trailing gaps. An entirely missing range returns
an empty array and advances to the requested end. An empty final page is possible
after a full page. There are no `cursor` / `nextCursor` fields.

`Result.From`, `To`, `AsOf`, and `Candle.Time` use Unix seconds. `AsOf` is the
request evaluation time, not a completeness watermark or a snapshot selector.
`Candle.PublishedAt` retains Unix **microseconds**, with zero meaning unknown.
Open/high/low/close are USD prices; pointers are nil when `Usable` is false.
Volume is base-asset quantity, quote volume is USD, and trade count is the
retained execution count. `Closed` is true, but later corrections may change
stored values and `Revision`.

`Coverage` reports membership evidence and missing minute publications inside
each candle. Result-level `MissingMinutes` also counts entirely absent intervals
inside the processed `[From,To)` window. A valid price does not imply complete
volume or coverage. Idle publications have zero trades and may retain a valid
admitted price from the trailing window; missing
publications have no row. Higher intervals aggregate whole intervals of stored
one-minute publications; pages do not freeze revisions across requests.

This replaces the previous interval-slot limit and opaque cursor contract.
Update requests using limits over 300 or a cursor together with the server.

The current method is `volume_weighted_latest`: each minute's OHLC is the path
of Composite prices admitted by MarketHub, using latest market prices weighted
by trailing 60-second base volume. `method` identifies this calculation separately
from the candle `interval`. A usable price candle can have zero `tradeCount` when
recent trades still contribute to the rolling price window. Volume is actual
transaction-time volume, not repeated rolling volume. Missing conversion or
observation evidence remains visible in coverage; it does not mean zero activity.

The new method reads only its own history from a seven-day memory cache, populated
by background Parquet restoration. It never splices the old five-second VWAP
history. A range predating cutover may therefore be empty or shorter than requested.
