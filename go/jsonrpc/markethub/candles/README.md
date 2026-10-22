# MarketHub.Candles.Get

`assetType` is required and only `crypto` is accepted. Use
`AssetType: market.AssetClassCrypto` from `finance/market` in Go. Missing, null,
empty and unsupported types are errors. Existing callers must add the selector;
the existing `marketType` values (`spot` / `perp`) remain unchanged.

The HTTP operation reads Composite or one resolved Mainnet DEX market's candles.
The recent seven days come from shared memory; earlier minutes come from retained
Parquet history. `market` omitted selects Composite, including eligible CEX
and DEX sources. Calculation versions and Parquet paths are not request parameters.
Use [`httpclient.NewClient`](../../../httpclient/README.md) to compose the client,
then `client.MarketHub().Candles().Get(ctx, candles.Params{...})`.

Signed authentication is required: inject an `authentication.CredentialProvider`.
The account/method quota is 60 requests per rolling 60 seconds. Each successful
request costs 1,000 credits from the first call, including empty results and
continuation pages. There is no anonymous access or free allowance.

| Parameter | Contract |
| --- | --- |
| `assetType` | Required `crypto`, using the shared `finance/market.AssetClass` enum. No omission default. |
| `asset` | Canonical lowercase asset slug, such as `btc`; required. |
| `marketType` | `spot` or `perp`; required. |
| `market` | Optional `finance/market.MarketTarget`: required `venue` and `network: "mainnet"`, optional `chain`, and at most one of `poolId` / `venueSymbol`. Resolve exactly one configured, eligible DEX market for the asset and market type. No match or multiple matches return `invalid_parameter`. |
| `from`, `to` | Positive Unix **seconds**; half-open `[from,to)` range of at most 7 days, aligned to the interval. |
| `interval` | `1m`, `5m`, `15m`, `1h`, `4h`, `1d`, aligned in UTC. |
| `limit` | Optional integer, 1–300; omitted or zero means 300. Counts actual returned candles, including idle/unusable rows. |
| `includeCurrent` | Optional boolean, default false. Include developing candles within the requested window. |

Without `includeCurrent`, the server rejects ranges containing unclosed minutes. `to` must be at or before
`floor((now - 2 seconds) / 60) * 60`, and aligned to the selected interval.
The `perp` spelling is part of this operation's contract.

With `includeCurrent: true`, `to` may reach the end of the interval containing
the server's current time, but no later. For example, at 12:34:20 UTC, `1m` may
end at 12:35:00 and `1h` at 13:00:00. Only minutes that have started are read;
future minutes are excluded from missing-minute counts. No prices are invented
for an unobserved interval. The current minute counts as one started minute, not
as proof of a completely observed minute.

For a venue request, `Result.Market` is the concrete `finance/market.MarketRef`,
and `Result.Symbol` is its canonical pair. Pin that resolved market for subsequent
pages and refreshes. Omitting pool/instrument filters is allowed only when the
selector resolves uniquely; different pools are never silently combined.
Individual CEX candles and Testnet markets are outside this initial scope.

Results are ordered by candle start. Save the original requested `to`, and set
the next request's `from` to **response `to`** until that original end is reached.
Missing intervals do not consume the limit and are never forward-filled. When
the limit is reached, `to` is the end of the final returned candle. Otherwise it
is the requested end, including trailing gaps. An entirely missing range returns
an empty array and advances to the requested end. An empty final page is possible
after a full page. There are no `cursor` / `nextCursor` fields.

`Result.HistoryFrom` optionally gives the earliest retained one-minute publication
for the selected series, in Unix seconds, independent of this page's range. It is
omitted while the background history index is unknown or the series is empty.
This boundary can be an idle/unusable minute; it does not promise usable prices
or continuous coverage. The index refreshes in the background every five minutes.
To load all history, first discover this boundary, align it down to the requested
interval, and paginate to the desired end. Each request still has a maximum
seven-day span and 300 returned candles. The SDK does not fetch pages automatically.

`Result.From`, `To`, `AsOf`, and `Candle.Time` use Unix seconds. `AsOf` is the
request evaluation time, not a completeness watermark or a snapshot selector.
`Result.SentAt` is the public Gateway send time in Unix **microseconds**.
`Result.Currency` is `USD`. `Definition` is `composite` or `venue`; venue candles
use `method: "pool_price"` and admission-time Pool OHLC for AMMs; order-book markets retain `method: "trade"` and transaction-time OHLC.
`Candle.PublishedAt` retains Unix **microseconds**, with zero meaning unknown.
Open/high/low/close are USD prices; pointers are nil when `Usable` is false.
Volume is base-asset quantity, quote volume is USD, and trade count is the
retained execution count. `Closed` is false until the interval end plus the
two-second closing delay; later corrections can still change closed values.
Native venue history without a durable publication timestamp keeps `PublishedAt`
at zero. Read time is never substituted for an unknown publication time.

Fetch history in pages, then refresh overlapping recent intervals. Replace each
bar by series and start time; do not add repeated volume totals. Revision changes
on data/evidence updates and when a developing bar closes. A developing final bar
still advances response `to` to its interval end: do not use that continuation
alone to refresh the current bar. The HTTP client does not start polling automatically.

`Coverage` reports membership evidence and missing minute publications inside
each candle. Result-level `MissingMinutes` also counts entirely absent intervals
inside the processed `[From,To)` window. A valid price does not imply complete
volume or coverage. Idle publications have zero trades and may retain a valid
admitted price from the trailing window; missing
publications have no row. Higher intervals aggregate whole intervals of stored
one-minute publications; pages do not freeze revisions across requests.

This replaces the previous interval-slot limit and opaque cursor contract.
Update requests using limits over 300 or a cursor together with the server.

The Composite method is `volume_weighted_reference`: each minute's OHLC is the path
of Composite prices admitted by MarketHub, using Pool state prices for AMMs and latest trade prices for order books, weighted
by trailing 60-second base volume. `method` identifies this calculation separately
from the candle `interval`. A usable price candle can have zero `tradeCount` when
recent trades still contribute to the rolling price window. Volume is actual
transaction-time volume, not repeated rolling volume. Missing conversion or
observation evidence remains visible in coverage; it does not mean zero activity.

The new method reads only its own history. The seven-day memory cache is populated
by background Parquet restoration; older requests read the same method's archive.
An interval crossing the cache boundary aggregates the archive and memory portions
once each. It never splices the old five-second VWAP
history. A range predating cutover may therefore be empty or shorter than requested.

Venue USD charts use the existing USD/USDT/USDC/USDG equivalence policy, or retained
transaction-time USD valuations for supported non-USD pairs. A current conversion
rate is never applied to old native OHLC. Missing or invalidated conversion yields
unusable prices and incomplete coverage, while known base quantities/trade counts
remain visible. Venue and Composite OHLC have different time bases; both retain
actual transaction-time volume.

The existing 1,000-credit success charge and authenticated quota also apply to
archive pages, selectors and refresh requests. The Console Market chart uses a
free internal route to this same calculation and shares the account/method quota.
The paid Historical page remains unchanged.

AMM display candles are stored independently of execution candles. Only the new `pool_price` history is served for AMM markets; there is no price-basis selector or trade-price fallback. Composite `volume_weighted_reference` history also starts at cutover. Existing files are retained. Volume remains actual transaction-time volume and is not inferred from state changes.
