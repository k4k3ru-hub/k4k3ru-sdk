# MarketHub.AMMPool.NewPair

Methods: `MarketHub.AMMPool.NewPair.List`, `.Get`, `.Subscribe`, `.Unsubscribe`.
The legacy `MarketHub.AMMPool.Launch` group remains available; new integrations should use NewPair.

```go
import newpair "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool/newpair"

filter := newpair.Params{Chain: "base", MaxPoolAgeSeconds: 86400}
// ws is a websocket.Module constructed with websocket.NewModule.
subscription, err := ws.AMMPoolNewPair().Subscribe(ctx, filter)
if err != nil {
    return err
}
for result := range subscription.Events() {
    // Replace the previous list with result.Pairs; absence removes a pair.
    _ = result
}
```

`ListParams{Filter: filter, Limit: 100}` is the List request payload. Get uses
`GetParams{Chain, Network, Venue, PoolID}`. Timestamps are Unix microseconds.
Pool age is independent of token deployment time. `minLiquidityUsd` is a decimal
string; unknown valuations never pass this filter, including a threshold of zero.
`hasSwap` is optional: true requires an observed swap, false requires no observed
swap, omitted includes both.

Token `id` may represent an EVM contract, native currency, Solana mint, or Sui coin
type. Pool identifiers preserve case. Position number/index strings and optional
chain-specific details support blocks, slots and checkpoints without loss of
integer precision. These types do not imply an adapter is deployed for every chain.

WebSocket event type: `apnp`. Each event is a bounded replacement snapshot; inspect
`truncated` and use List pagination for larger result sets. Subscription identity
includes all normalized filters. Unsubscribe with the owning subscription client.
