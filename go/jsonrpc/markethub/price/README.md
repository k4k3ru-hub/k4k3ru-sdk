# Current Composite price

`MarketHub.Price.Get` is an HTTP snapshot operation. `Params.Asset` is the
canonical asset slug, and `Params.MarketType` uses the owning
`finance/market.MarketType` (`spot` or `perpetual`).

`Params.AssetType` is required: use `market.AssetClassCrypto` from
`finance/market` (`"assetType":"crypto"`). Missing, null, empty and unsupported
values are rejected; no crypto default is applied. Existing requests must add
the field. The Result is unchanged.

`Result.Price` is a decimal string pointer, `nil` for `unavailable`.
`Status` is `live`, `stale`, or `unavailable`. The fixed metadata is
`definition: composite`, `currency: USD`, `method: volume_weighted_latest`, `interval: 60s`.
`VenueCount` counts distinct contributors to that price, not pools or configured
markets. `SentAt` is the Gateway send time in Unix microseconds, never trade time.

Prices use in-memory Trade/Swap contributions, including eligible CEX and DEX
markets. Latest prices use trailing 60-second volume weights. Live prices
become stale after 60 seconds without a newer contributing execution, or earlier
if a required USD conversion input expires. Stale
prices persist until superseded or invalidated; startup does not restore Parquet.

Use `httpclient.NewClient`, then `client.MarketHub().Price().Get(ctx, params)`.
The client does not retry or poll automatically. Anonymous calls have 3 requests
per IP/method per rolling minute. Signed calls share 60 per account/method:
first 3 free, then 100 credits each, including unavailable/unchanged responses.

The current Composite price weights each eligible market's **latest** Trade/Swap
price in USD by its base volume over the trailing 60 seconds. This is not a VWAP
of historical trade prices. `interval` describes the weighting window, not a
five-second candle or the client's polling frequency. CEX and DEX contributions
are separated by market type; `venueCount` counts distinct contributing venues.

Composite minute OHLC uses the prices admitted by MarketHub, at admission time.
Volume uses actual transaction-time totals and is not the sum of rolling weights.
New-method candle history starts at cutover; older five-second VWAP files are
retained separately. Current-price responses are memory-only and do not restore
an old candle close as a fresh price after restart.

For multi-period percentage comparisons, use the separate
[`MarketHub.Price.Change.Get` contract](change/README.md) and
`client.MarketHub().Price().Change().Get(ctx, params)`. It returns the current
price once and exposes the actual reference price/time and status for each period.
