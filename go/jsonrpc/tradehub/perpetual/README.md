# Manual perpetual RPC models

Owning import: `github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/perpetual`.
Method constants live in `jsonrpc`; no third-party Exchange types are exposed.

| Method suffix under `TradeHub.Perpetual` | Request | Response |
| --- | --- | --- |
| `Account.Get` | `AccountParams` | `AccountResult` |
| `Prepare` | `PrepareParams` | `PrepareResult` |
| `Submit` | `SubmitParams` | `SubmitResult` |
| `Order.Get` | `OrderParams` | `OrderResult` |

These are authenticated HTTP RPCs. Initial scope is Hyperliquid Testnet SUI/USDC.
Use `jsonrpc.MethodTradeHubPerpetualPrepare` and a `perpetual.PrepareParams` value
in the existing signed JSON-RPC envelope. There is no new transport constructor
or WebSocket subscription in this package.

For example, construct the intent using the owning package:

```go
params := perpetual.PrepareParams{
    Scope: perpetual.Scope{
        Venue: "hyperliquid", Network: "testnet", Symbol: "SUI/USDC",
        AccountAddress: tradingAddress,
    },
    SignerAddress: signingAddress,
    Kind: "order",
    Nonce: allocatedNonce,
    Order: &perpetual.OrderIntent{
        Side: "buy", Quantity: "1", LimitPrice: "2",
        TimeInForce: "ioc", ReduceOnly: false, ClientOrderID: clientOrderID,
    },
}
if err := params.Validate(); err != nil {
    return err
}
```

The addresses, nonce, client ID and price above must come from the caller's
explicit intent and durable nonce/journal workflow. `nonce`, `preparedAt`,
`expiresAfter`, fill `time` and `observedAt` use Unix milliseconds. Order/trade
IDs use decimal strings. Sizes, prices and fees use decimal strings.

Prepare returns normalized action JSON, contract metadata, original intent,
action hash, EIP-712 digest and a 60-second authenticated token. Reconstruct and
verify the full intent locally before signing. Submit accepts only that token,
preparation ID, digest and canonical low-S signature (`r`, `s`, `v: 27|28`).
Unknown fields, duplicate keys and missing required fields are rejected during
JSON decoding. `Validate` checks semantic constraints; current market precision,
account role, agent ownership, expiration and cryptography are server checks.

Do not confuse K4K3RU request authentication with the Hyperliquid Exchange
signature. Keep tokens/signatures private. A separate `kind: "leverage"` intent
uses `LeverageIntent{Value: ..., MarginMode: "cross"|"isolated"}` with no order.

Submission statuses are `accepted` (leverage), `filled`, `partially_filled`,
`rejected`, `resting`, `observed`, or `unknown`. Exact retries retain the original
nonce and signature. `observed` suppresses resubmission of a matching client ID.
Unknown outcomes and network errors require venue reconciliation; they do not
authorize creating a new order. Expired tokens cannot be resubmitted.

Order results report observed aggregates, not invented complete history.
`fillsComplete` is only true when a venue-filled order reconciles to its original
quantity. `fillsLimited` describes local pagination limitations; a false value
does not remove venue retention limits. `observedAveragePrice` is rounded to at
most 18 fractional digits. Read Account.Get to determine residual positions.

Only ordinary user accounts with account abstraction explicitly `disabled` are
supported for trading. Other modes return `tradingSupported: false` with no
perpetual balance fields. TradeHub execution must be enabled independently of
MarketHub; Mainnet execution is not allowed by this API version.
