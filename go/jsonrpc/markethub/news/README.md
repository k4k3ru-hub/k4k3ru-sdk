# Retained Asset news

`news.Params` and `news.Result` own `MarketHub.News.List` (HTTP only).
Use `httpclient.NewClient` as the composition root and
`client.MarketHub().News().List(ctx, news.Params{AssetType: "crypto", Asset: "sui"})`.
The caller owns transport, timeout and optional credentials. One request returns
one page; retries and pagination are explicit because authenticated calls may be billed.

The first response's opaque `Snapshot` pins source run membership. Preserve it
with the same Asset/filters for subsequent pages. `Page` defaults to 1, `Limit`
to 20 (max 100). `From`/`To` optionally bound source publication time in Unix
microseconds. `Result.ValidateFor(params)` checks scope and provenance.

Items include source links, short source summaries, item/revision/run identities,
publication/update/event/collection timestamps and Mainnet scope evidence.
`Coverage=latest_source_windows` is not exhaustive Web or archive coverage.
Source health remains current at request time and is separate from the pinned
metadata. Empty, incomplete, failed and unconfigured sources are distinct.

This SDK addition does not deploy the service, migrate DB tables or collect news.

Generic v2 responses add `Item.ArticleID`, `RecordSHA256`, and `AnnotationSetID`.
The existing `ID` remains source-qualified and `Revision` remains the public
metadata hash. `Source.Input` pins the annotation document hash/profile, parser
and producer versions, publisher ID, and captured source name/endpoint. Existing
source run/hash/observed time/counts also describe the pinned window.

`Source.Status` remains collection health. Optional `Source.Classification`
describes the latest completed collection's classification separately:
`never_classified`, `pending`, `running`, `failed`, or `complete`. A newer failed
collection does not reset classification health. Its last-success timestamp is
current; `Stale` compares the selected pair to the latest completed collection
and its latest annotation attempt. Previous good pairs remain usable with that
explicit status. An old pinned empty window may have no Input and current complete
classification health. Current source name/homepage and health may change between
pages; captured Input metadata and selected counts stay pinned.

Legacy v1 responses omit these additions. Preserve either version's opaque
snapshot exactly; v2 also fixes Asset/filters, classification profile and policy.
`ValidateFor` rejects partial additions, mismatched annotation/profile identity,
unsupported totals and control characters in new metadata. Source input tokens
are nonempty ASCII `[A-Za-z0-9_.:/-]+`, at most 64 bytes; source names are at most
256 UTF-8 bytes. Current classification fields are optional for legacy compatibility.
