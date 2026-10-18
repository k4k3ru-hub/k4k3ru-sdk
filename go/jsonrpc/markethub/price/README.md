# Current Composite price

`MarketHub.Price.Get` is an HTTP snapshot operation. `Params.Asset` is the
canonical asset slug, and `Params.MarketType` uses the owning
`finance/market.MarketType` (`spot` or `perpetual`).

`Result.Price` is a decimal string pointer, `nil` for `unavailable`.
`Status` is `live`, `stale`, or `unavailable`. The fixed metadata is
`definition: composite`, `currency: USD`, `method: vwap`, `interval: 5s`.
`VenueCount` counts distinct contributors to that price, not pools or configured
markets. `SentAt` is the Gateway send time in Unix microseconds, never trade time.

Prices use in-memory Trade/Swap contributions, including eligible CEX and DEX
markets. The latest priced five-second bucket may still be forming. Live prices
become stale after 60 seconds without a newer contributing execution. Stale
prices persist until superseded or invalidated; startup does not restore Parquet.

Use `httpclient.NewClient`, then `client.MarketHub().Price().Get(ctx, params)`.
The client does not retry or poll automatically. Anonymous calls have 3 requests
per IP/method per rolling minute. Signed calls share 60 per account/method:
first 3 free, then 100 credits each, including unavailable/unchanged responses.
