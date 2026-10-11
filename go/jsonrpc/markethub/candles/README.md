# MarketHub.Candles.Get

The HTTP operation reads stored Composite candles. Its stable series identifier
is `composite`; calculation versions and Parquet paths are not request parameters.
Use [`httpclient.NewClient`](../../../httpclient/README.md) to compose the client,
then `client.MarketHub().Candles().Get(ctx, candles.Params{...})`.

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
volume or coverage. Idle publications have zero trades and nil prices; missing
publications have no row. Higher intervals aggregate whole intervals of stored
one-minute publications; pages do not freeze revisions across requests.

This replaces the previous interval-slot limit and opaque cursor contract.
Update requests using limits over 300 or a cursor together with the server.
