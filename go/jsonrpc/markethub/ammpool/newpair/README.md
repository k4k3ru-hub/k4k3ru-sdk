# MarketHub.AMMPool.NewPair

`activity.windows["5m"]` and `["15m"]` may include `transactionSenders`.
It reports outer transaction senders, not people, buyers or holders. The overall
and directional `uniqueCount` values are nullable exact decimal strings;
`resolvedSwapCount` is the number of corresponding Activity Swaps with a verified
sender. Compare it with the same window's Swap count. Partial counts are lower
bounds, never estimates. No resolved sender in a nonempty window means a null
unique count and `"0"` resolved Swaps; an empty observed window has both `"0"`.
An absent/null object means unavailable, unsupported or invalid state. These
fields do not claim completeness of on-chain history or change listing rules.
`CloneTransactionSenders` and the Activity clone helpers detach all pointers.

`Pair.LPPrincipal` optionally returns Token0/Token1 principal amounts as exact
decimal strings in whole-token units. `AmountPercentage` compares those quantities
without price weighting: 1 token versus 99,999 tokens gives 0.001% versus 99.999%.
Percentages use 0–100, unlike the fractional rates in `Fees`. Token0 is truncated
to 18 decimal places; Token1 is its complement to 100. Both percentages are nil
when the total amount is zero. A rounded 0% does not imply an exactly zero amount.
Principal covers all price ranges, excluding uncollected fees and direct transfers;
it is distinct from active liquidity, USD value and LP protection.
`EvaluatedAt` (Unix microseconds) and `Position` describe the adopted LP observation
and do not change on USD-reference-only updates. Missing/unverified LP state returns
nil; older JSON without the field remains readable. `CloneLPPrincipal` copies an
observation for independent retention. Quantity imbalance is not a listing filter.

`Pair.LPProtection` optionally describes Token0/Token1 principal protection across
all positions. `lockedLiquidityPercentage` covers time-limited locks;
`permanentlyProtectedLiquidityPercentage` covers verified permanent protection.
These are decimal strings in [0,100], or null when unresolved or that token has no
principal. `allPositionsProtected` is an independent nullable boolean, never
derived from rounded percentages. `canWeakenProtection` separates current custody
from authority to weaken it; current 100% and a true authority finding can coexist.

Statuses are `pending`, `available`, `stale`, `unavailable`, `unsupported`, and may
be extended. Only `available` is usable for current conditions. `stale` retains the
past values, original `observedAt` (analysis completion in Unix microseconds) and
block `position`; expiry/reconnect do not advance them. Reorg-cancelled observations
are cleared. Pending or unresolved values are null, never inferred zero/false.
Authority findings use `observed` with a boolean, or `pending`, `unknown`,
`not_applicable` with null; token definitions never make LP custody trusted.
Old JSON without `lpProtection` remains readable. `CloneLPProtection` detaches all
mutable fields. LP protection does not determine NewPair listing eligibility.

Methods: `MarketHub.AMMPool.NewPair.List`, `.Get`, `.Subscribe`, `.Unsubscribe`.
Use this group for pool discovery. The former Launch RPC group and its SDK types have been removed.

```go
import newpair "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool/newpair"

filter := newpair.Params{Chain: "base"}
// ws is a websocket.Module constructed with websocket.NewModule.
subscription, err := ws.AMMPoolNewPair().Subscribe(ctx, filter)
if err != nil {
    return err
}
for result := range subscription.Events() {
    // Read result.Pairs and result.ExcludedPairs.
    // With truncated=true, absence from Pairs does not prove exclusion.
    _ = result
}
```

`ListParams{Filter: filter, Limit: 100}` is the List request payload. Get uses
`GetParams{Chain, Network, Venue, PoolID}`. Timestamps are Unix microseconds.
Optional filters are chain, network and venue; empty values mean all.

### Optional comparison queries

`Params.Query == nil` retains legacy filters, keys and pagination. A non-nil
`Query` (JSON `"query": {}`) opts into comparison queries. Defaults are `5m`,
no conditions, and `poolCreatedAt` descending. The same `Params` is used for
List's filter and Subscribe/Unsubscribe; notifications remain `apnp` without
a subscription key. `Params.Equal` compares normalized semantics, including the
query. Do not compare query pointers to identify equivalent filters.

```go
filter := newpair.Params{
    Chain: "base", Network: "mainnet", Venue: "uniswap-v4",
    Query: &newpair.Query{
        ActivityPeriod: "5m",
        Conditions: newpair.QueryConditions{
            MinLiquidityUSD: "10000",
            MinSwapCount: "20",
            MaxPoolFeeRate: "0.01", // 1%; an example, not a default threshold.
        },
        Sort: newpair.QuerySort{Field: "volumeUsd", Direction: "desc"},
    },
}
params := newpair.ListParams{Filter: filter, Limit: 20}
// For continuation, preserve Filter/Limit and use the returned NextCursor.
_ = params
```

All active conditions are ANDed against the entire listed scope before sorting
and pagination. Unknown required metrics do not match. Missing conditions impose
no constraint; `"0"` is an active threshold. `require*` booleans only constrain
when true. Counts and amounts are exact decimal strings; rates are fractions
(0–1), whereas change percentages use percent units (`"25"` means +25%).
Limits are 78 integer digits, 18 fractional digits and 4 KiB per query JSON.
Explicit null, unknown/duplicate fields and exponent notation are invalid.
Normalize detaches the query and canonicalizes equivalent decimals; SubscriptionKey
adds a SHA-256 digest while preserving the exact legacy key when Query is nil.

Periods are `5m`, `15m`, `1h` and `24h`. Sender counts and preceding-period
comparisons only support `5m`/`15m`; incompatible conditions/sorts are rejected.
Sorting supports `poolCreatedAt`, `liquidityUsd`, `swapCount`, `volumeUsd`,
`uniqueSenderCount`, `swapCountChangePercentage` and `volumeUsdChangePercentage`.
Unknowns sort last in either direction; ties use creation time descending, then
chain/network/venue/pool identity ascending. Sender counts remain lower bounds.

Token conditions permit server-verified, onchain-defined trusted tokens to skip
tax-rate and control checks. No rate or boolean is filled in: trusted USDC can
retain null TokenTaxes. Trust matches chain/network/token ID, never a symbol.
Other tokens need the requested observed values; renounced ownership is not
proof that minting or other permissions are absent. LP conditions require current
`available` protection and reject stale/expired findings. Trust never bypasses LP
protection checks. These filters do not change the service's listing policy.

Query results add `queryResult: {matchedCount: "85", expiresAt: ...}`. Live uses
null expiry and reports the count before wire-size truncation. List holds immutable
public data, timestamp and ordering for at most 120 seconds (earlier LP unlocks
shorten this). Ordinary updates do not alter captured pages. Withdrawn evidence
or sender-generation changes invalidate them. Subsequent pages need no DB read.

Initial limits: 2,000 matched pools/8 MiB per search, 64 searches/64 MiB per process,
two concurrent builders, 10-second build timeout, and 220 KB response budget.
The cache is process-local: restart, expiry or routing to a different replica
requires a new first-page search; multi-replica deployments need affinity.
`expired` and `conflict` require a deliberate refresh. `CodeQueryTooLarge` means
narrow the scope/conditions; `CodeQueryBusy` means retry later. Invalid or mismatched
cursors use `invalid_parameter`. Existing valid searches are not evicted to admit
new ones. Deploy the query-capable SDK/server before enabling query clients.

### Pool observations

The new server contract uses pool creation time for the 24-hour lifecycle, a
confirmed observed swap, liquidity of at least 1,000 USD, and a synchronized LP state with a successful
valuation. `SwapObservedAt` is not necessarily the first swap in pool history;
null means no swap has been observed, not that no swap ever occurred.
`SwapObservedPosition` identifies the observed event; `ConfirmedAt` is nullable
until its confirmation completes. `LiquidityEvaluatedAt` records the last
successful evaluation. All nullable timestamps are Unix microseconds.

`LiquidityUSD` keeps its Finding representation (decimal string value or unknown).
`LiquidityMethod` describes the valuation basis. The agreed valuation uses LP
principal across all price ranges, excluding uncollected fees/direct transfers,
with 1 USDC = 1 USD as a conversion assumption. FDV/MarketCap are not included.
A failed refresh or stale evaluation must not be rewritten as zero liquidity.
Capture retry budgets are server configuration, not request parameters. The removed age, swap and USD threshold request fields remain rejected.

Token `id` may represent an EVM contract, native currency, Solana mint, or Sui coin
type. Pool identifiers preserve case. Position number/index strings and optional
chain-specific details support blocks, slots and checkpoints without loss of
integer precision. These types do not imply an adapter is deployed for every chain.

For ERC20 tokens registered in onchain's token metadata definitions, `security.owner`,
`security.implementation`, `security.admin`, and `security.beacon` return
`{"status":"trusted","reason":"sdk_definition"}` without `value`. The server
matches chain, network, and token address, never the symbol. These four RPC checks
are skipped by policy; this does not mean ownership was renounced, no proxy exists,
or the pool/counter-token is trusted. Symbol and decimals retain `known` with their
definition values. Native currency keeps `not_applicable` for contract checks.
When the token definition includes a non-empty name, `name` also returns `known`
with its definition value and `sdk_definition`, skipping the `name()` RPC.
Unregistered tokens and definitions without a name retain the existing name RPC.
Trust is current configuration, not a historical observation: it applies at discovery,
snapshot restoration, and even when block-dependent metadata is unavailable.
`assessmentStatus`/`assessedAt` still describe metadata processing separately.
Finding statuses are extensible strings; clients should display unrecognized values.
This policy does not infer tax rates or modify the `TokenTaxes` contract.

WebSocket event type: `apnp`. Each event is a bounded replacement snapshot; inspect
`truncated` and use List pagination for larger result sets. Subscription identity
includes all normalized filters. Unsubscribe with the owning subscription client.

## Listing and exclusions

`Result.Pairs` contains listed pairs only (`isListed=true`). Subscribe snapshots
also carry `Result.ExcludedPairs`: the last full data for previously listed pools
that left the listing (`isListed=false`, `exclusionReason` set). Previously unlisted
candidates need not be exposed. Get/List use the same listing conditions; the
exclusion list supplements Subscribe. Exclusion reasons include `age_exceeded`,
`liquidity_below_minimum`, `liquidity_unavailable`, `liquidity_stale`,
`swap_unconfirmed`, and `creation_reverted`. Treat reasons as extensible strings.

For `liquidity_stale`, the last value and evaluation time remain available. A pair
that qualifies again returns to `pairs` without an exclusion reason. SDK routing
preserves `excludedPairs` when replacing an unread snapshot with a newer snapshot.
This is latest-state delivery, not a durable history of every transition.
Exclusion retention limits remain a server implementation detail to be finalized;
do not assume every exclusion can be replayed. When `truncated=true`, absence from
`pairs` does not prove exclusion. Resynchronize on epoch changes or reconnects.

## Compatibility and rollout

This is a breaking type/wire update: `firstLiquidityAt`, `firstSwapAt` and
`backfillAbandonedAt` are removed rather than aliased to fields with different
meanings. New nullable evidence fields serialize as JSON null when unknown.
`isListed` always serializes; no listing inference is made from an older response
that omits it. Deploy matching server and client versions together.

This SDK change defines and decodes the new contract. It does not implement the
server's listing policy, valuation worker, exclusion retention or notification
production; those service changes follow separately.

### Live LP synchronization

`lpStateStatus` is `syncing`, `synced` or `unavailable`. A missing/unknown status
must not be interpreted as synced. `lpStatePosition` identifies the last established
LP state, including an optional transaction/event position; it is not proof that
all subsequent blocks were checked. Synchronization is process-local and must be
re-established after a restart.

`liquidityEvaluatedAt` is the time of the last adopted USD calculation. It is no
longer the LP-state block timestamp. There is no fifteen-minute expiry: quiet
synchronized pools remain eligible until their creation-based 24-hour limit.
Disconnects or detected gaps withdraw listing with `lp_state_unavailable`;
initial/recovery capture uses `lp_state_syncing`. Missing reference prices use
`usd_reference_unavailable`. Last amounts/times remain available in excludedPairs.


## Cumulative activity

`Pair.activity` is optional for compatibility with older servers. It contains
observed cumulative activity throughout the monitoring lifetime, including time
before listing and temporary exclusion. It has no completeness or gap status.
Missing history is not inferred as zero and no full-history guarantee is made.

- `startedAt` / `updatedAt`: aggregation start / last incorporated activity update,
  in Unix microseconds. `startedAt` is not a claim that recovered events began then.
- `swapCount`: total Swap event count, as a decimal string.
- `token0ToToken1` / `token1ToToken0`: `count`, `token0Amount`, `token1Amount`,
  `volumeUsd`. A zero-amount Swap is included only in `swapCount`.
- `liquidityAdded` / `liquidityRemoved`: `count`, `token0Amount`, `token1Amount`.
- `netToken0Liquidity` / `netToken1Liquidity`: additions minus removals.

Quantities use token units and normalized decimal strings, without float64.
Unavailable quantities are JSON null. V4 LP quantities are null after a liquidity
change because ModifyLiquidity does not emit token amounts. LP reductions refer
to principal removed from positions, not subsequent Collect transfers; fees and
Donate events are excluded. Swap amounts are pool-level, not final user receipts.

USD volume uses one available shared reference leg per swap and excludes gas.
The existing 1 USDC = 1 USD convention applies. Values are fixed at observation
and truncated to 18 decimals. A directional USD volume is null if any of its
counted swaps could not be priced. Recovery does not use current spot prices for
old trades; only directly configured USDC amounts can be priced during recovery.

HTTP List/Get and the `apnp` stream use the same Activity type. Request parameters,
listing conditions and other existing response fields remain unchanged.

`Activity.Windows` adds optional `1h` and `24h` observations without changing the
monitoring-lifetime fields. Each period covers `[from,to)` on UTC minute
boundaries; times use Unix microseconds. It contains the same directional swap
counts/amounts, single-leg USD volumes and LP addition/removal/net quantities.
`observedFrom` identifies the start of period aggregation within the window;
young pools need not have a full hour/day of observations. This timestamp is not
a completeness guarantee. `to` is the last published boundary, not necessarily
the current time. Partial minutes are excluded from these periods.

Missing `windows` supports older servers. A null period is unavailable; it does
not mean zero trades. Unknown quantities and USD volumes remain null, and net
LP quantities may be negative. LP additions minus removals do not include reserve
changes caused by swaps or changes in USD valuation. `CloneActivity` and
`CloneActivityWindows` detach all period pointers. The DTO addition does not
by itself enable period aggregation on a server; storage and watcher rollout
are required.

`windows["5m"]` and `windows["15m"]` use the same closed-minute totals and add
`comparison` against the immediately preceding, non-overlapping period of equal
length. For a 5m window `[12:05,12:10)`, comparison covers `[12:00,12:05)`.
Each comparison contains `from`, `to`, `swapCount` and `volumeUsd`. Each metric
has nullable decimal strings `previous`, `delta` (current minus previous), and
`changePercentage` ((current minus previous) / previous * 100). Percentage is
truncated toward zero to 18 fractional places: `"200"` means +200%, not 2%.
A zero previous value leaves only the rate null, including when current is zero.
An unknown USD value leaves the affected comparison values null, independently
of the count comparison. USD volume is the sum of both directional volumes only
when both are known; partial known volume is never used as the total.

Young pools still return their observed current totals and `observedFrom`, but
`comparison` remains null until monitoring started no later than the preceding
period's start and both periods can be constructed. This requires approximately
10/30 minutes plus less than one minute of boundary alignment, and is not a
historical completeness guarantee. No new history fetch is required to create a
comparison. Missing short keys from older servers decode as nil. Hourly/daily
windows retain their existing shape without a comparison field.

The server reconstructs short windows from existing minute rows, keeps them in
memory, and includes them in List/Get/Subscribe responses. It does not duplicate
short windows/comparisons in durable snapshots. Before restoration finishes,
short windows may be null. Period-only advancement does not write the database.

## Pool swap fee observations

`Pair.Fees` is optional for compatibility with older servers. Current MarketHub
listing requires both directional rates. `rate` is a decimal fraction (`"0.003"`
means 0.3%); nil does not mean zero. Fees exclude token taxes, gas and price impact.
Variable fees are reference values for `Position` and `ReferenceSender` at
`ObservedAt` (Unix microseconds), not next-order guarantees. No fee status or
freshness TTL is supplied. Use `CloneFees` when retaining independently mutable
snapshots. `fees_unavailable` is a possible exclusion reason.

## Token tax observations

`Pair.TokenTaxes` is nullable and contains independently nullable `token0` and
`token1` observations. A missing field in an older response and explicit JSON null
both decode as nil. Each observation contains nullable `buyRate`, `sellRate`,
`canChange`, `hasExemptions`, plus `source`, `observedAt` and `position`.
Rates are decimal fractions: `"0.01"` means 1%; `"0"` is confirmed zero and differs
from null. False is a confirmed negative finding and differs from null as well.
`observedAt` uses Unix microseconds; `position` identifies the evaluated block.
`CloneTokenTaxes` / `CloneTokenTax` detach pointer fields for retained snapshots.

Initial server analysis covers reviewed code-only tax-free models on Base mainnet
and native currency. The onchain-defined Base USDC address is trusted and skipped:
its observation is null, not an inferred zero-tax verdict. An unknown, unsupported,
or failed analysis also remains null, without excluding an otherwise listed pool.
Analysis runs asynchronously, so a later `apnp` replacement snapshot may contain
observations missing in an earlier List/Get/Subscribe result.

`source` is `contract_analysis` or `native_currency`. These findings describe token
tax rules at the observed block; they do not prove tradability, LP protection,
owner renouncement, or future fees. Pool swap fees, token taxes, gas and price
impact remain separate fields/responsibilities. There is no new request parameter,
status enum, or subscription operation for this additive response field.

## Token Controls

`Pair.Token0.Controls` and `Pair.Token1.Controls` expose token capabilities using
`TokenControls`. Existing JSON without Controls decodes to nil. Every finding
contains `status` and an explicit nullable `value`; `observed` false is a real
boolean, while `unknown`, `pending`, `trusted` and `not_applicable` carry null.

The groups are `ownership`, `transferRestrictions`, `minting`, `upgrade` and
`balanceControl`. Taxes remain in `Pair.TokenTaxes`. Controls describe the block
pinned at analysis start, independently of the pool-creation observations.
`ObservedAt` is Unix microseconds, and `Position` identifies that block. Both are
null when there are no observed fields. A partial owner acquisition failure can
coexist with verified code findings at the same block.

Base onchain-defined token addresses are trusted by policy without analysis;
this does not mean their controls are absent. Native currency is not applicable.
Unsupported chains or unavailable analysis may leave Controls nil. Unknown
models do not affect NewPair listing. `CloneTokenControls` detaches every mutable
value and position; `NewTokenControls(status, reason)` creates a uniform null
observation for pending, unknown, trusted or not-applicable states.

`minting.present` describes a runtime mint entry point. `canMint` describes the
token's permission path at the observed block, not transaction success. Supply
cap values, when available, use raw smallest-unit decimal strings. Controls do
not cover LP protection or promise continuous monitoring.
