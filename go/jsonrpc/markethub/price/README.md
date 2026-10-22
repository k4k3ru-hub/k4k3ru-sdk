# Current Composite and market prices

`MarketHub.Price.Get` is an HTTP snapshot operation. `Params.Asset` is the
canonical asset slug, and `Params.MarketType` uses the owning
`finance/market.MarketType` (`spot` or `perpetual`).

`Params.AssetType` is required: use `market.AssetClassCrypto` from
`finance/market` (`"assetType":"crypto"`). Missing, null, empty and unsupported
values are rejected; no crypto default is applied. Existing requests must add
the field. Omitting `markets` preserves the Composite-only response.

`Result.Price` is a decimal string pointer, `nil` for `unavailable`.
`Status` is `live`, `stale`, or `unavailable`. The fixed metadata is
`definition: composite`, `currency: USD`, `method: volume_weighted_reference`, `interval: 60s`.
`VenueCount` counts distinct contributors to that price, not pools or configured
markets. `SentAt` is the Gateway send time in Unix microseconds, never trade time.

Prices use in-memory Trade/Swap contributions, including eligible CEX and DEX
markets. Latest prices use trailing 60-second volume weights. Live prices
become stale after 60 seconds without eligible contributing inputs, or earlier
if a required USD conversion input expires. Stale
prices persist until superseded or invalidated; startup does not restore Parquet.

Use `httpclient.NewClient`, then `client.MarketHub().Price().Get(ctx, params)`.
The client does not retry or poll automatically. Anonymous calls have 3 requests
per IP/method per rolling minute. Signed calls share 60 per account/method:
first 3 free, then 100 credits each, including unavailable/unchanged responses.

The current Composite price weights each eligible market's **latest** order-book trade price or verified AMM Pool state
price in USD by its base volume over the trailing 60 seconds. This is not a VWAP
of historical trade prices. `interval` describes the weighting window, not a
five-second candle or the client's polling frequency. CEX and DEX contributions
are separated by market type; `venueCount` counts distinct contributing venues.

Composite minute OHLC uses the prices admitted by MarketHub, at admission time.
Volume uses actual transaction-time totals and is not the sum of rolling weights.
New-method candle history starts at cutover; older five-second VWAP files are
retained separately. Current-price responses are memory-only and do not restore
an old candle close as a fresh price after restart.

## Individual markets

Optionally pass `Params.Markets` with 1–64 exact `finance/market.MarketRef` values
returned by `MarketHub.Markets.List`. The initial scope is Mainnet DEX markets.
AMMs require chain/network/pool identity; order books require network/venueSymbol.
Empty, null, duplicate, unsupported and mismatched asset/market-type selections
are rejected. The requested order is preserved in `Result.Markets` and verified
by the HTTP client. The selection does **not** filter Composite contributors.

Each `MarketPrice` contains `market`, canonical `symbol`, `currency: USD`,
`method: pool_price` (AMM) or `last_trade` (order book), nullable `price`, and `status`. It is the latest valid
reference price, converted with retained USD rates during background refresh; it is not a size-specific quote. Order books retain trade-time freshness. AMMs use adapter-confirmed Pool freshness and withdraw unavailable or interrupted states without a trade fallback.
All entries share the response's `sentAt`; it does not indicate execution time.
No per-market interval or contributor count is returned. Use `Quote.Get` for
size-specific execution estimates.

One batch is one request under the existing quota/credit policy. No additional
Venue RPC or Parquet read is made by Price.Get, and restarting does not restore
individual current prices from historical candles.

For multi-period percentage comparisons, use the separate
[`MarketHub.Price.Change.Get` contract](change/README.md) and
`client.MarketHub().Price().Change().Get(ctx, params)`. It returns the current
price once and exposes the actual reference price/time and status for each period.

## 2026-10-22 reference-price cutover

The active Composite method is `volume_weighted_reference`. AMM inputs use verified Pool state prices; order-book inputs retain last-trade prices. Trailing 60-second base volume supplies the weights. Pool states that are unavailable never fall back to effective Swap prices. Market entries use `pool_price` for AMMs and `last_trade` for order books.

Requests are unchanged. Old `volume_weighted_latest` histories remain stored but are not served or spliced into the active series. Update SDK consumers and services together; the current validator accepts only the new Composite method. Actual Swap quantities, fees and execution records are unchanged.
