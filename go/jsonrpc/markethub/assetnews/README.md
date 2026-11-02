# Official Asset context

`MarketHub.AssetNews.List` reads retained metadata from registered official sources.
Types belong to this package. Compose the operation through
`httpclient.NewClient(...).MarketHub().AssetNews().List(ctx, assetnews.Params{...})`.
The constructor accepts the caller's HTTP client and makes no requests itself.

The result uses `scope=asset_context`; it is separate from the existing Mainnet-only
`MarketHub.News.List`. Items use generic `articleId`, `recordSha256` and
`annotationSetId`, explicit `networkScope`/`claimStatus`, source attribution and
publication/observation times. Sources report `eligibleCount` and
`associationBasis=registered_official_source`. `input.classifiedAt` identifies the
selected annotation's completion time, independently of current classification health.

The initial metadata profile retains unknown, non-Mainnet and mixed scope. Blog
metadata defaults to `unspecified` and claims to `unverified`; source wording about
future plans is preserved and never treated as completed deployment. Official-source
association does not establish Mainnet operation, price causation or investment value.
Network-non-applicable and unknown have different meanings.

Publication bounds use UTC Unix microseconds (`from` inclusive, `to` exclusive).
Page 2+ requires the unchanged opaque snapshot from page 1. Context snapshots are
not interchangeable with Mainnet News snapshots. The client sends one request and
does not retry or paginate automatically. Full article bodies are not redistributed.

This source implementation does not establish release or production deployment.
