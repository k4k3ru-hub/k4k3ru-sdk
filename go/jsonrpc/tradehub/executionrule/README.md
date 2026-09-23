# Shared round-trip execution rules

This package owns `Rule`, the Spot/Perp Open variants, Close triggers, and
`AssetRef` / `MarketRef`. Import these types from their owning package; they
are not re-exported from the SDK root or borrowed from MarketHub strategies.

Both `Rule.Open` and `Rule.Close` are required values. `marketType` belongs to
the enclosing request, not either leg or the Rule. Call `rule.Validate(marketType)`
to enforce the enclosing type and the corresponding Open variant. A future
single-swap request can omit the entire rule; it cannot supply an incomplete
round trip. Existing Swap DTOs are unchanged.

`Normalize` returns an independent copy without adding trading defaults.
JSON decoding rejects absent/null legs and unknown or duplicate fields.
The structural validator does not verify asset equivalence, venue support,
inventory, lot sizes, leverage limits, trigger accounting, or executable prices.

See the [Scalping contract and examples](../scalping/README.md) for reference
asset units, required per-market fields, and the service responsibilities.
