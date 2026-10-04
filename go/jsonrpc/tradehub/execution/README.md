# Common execution RPCs

Use the owning packages:

```go
import (
    "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
    "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution/prepare"
    "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution/hyperliquid"
)
```

- `TradeHub.Execution.Prepare`: `prepare.Params` -> `prepare.Result`.
  Use `kind: "swap"` with existing `swap.PrepareParams`, or `kind: "perpetual"`
  with `prepare.PerpetualParams`. The latter separates venue signing inputs from
  the order or leverage intent.
- `TradeHub.Execution.Submit`: `execution.SubmitParams` -> `execution.SubmitResult`.
  Onchain transaction shapes are retained. Venue signatures use
  `signedPayload.action`; receipts use `action` without an onchain transaction ID.
- `hyperliquid.PrepareParams`, `Prepared`, `SubmitParams` and `Receipt` bridge
  existing Hyperliquid DTOs/journals to the common contract. `Prepared` validates
  envelope bindings; callers must still independently verify the action and
  digest before signing. Current adapter scope is Testnet SUI/USDC.

Prepare times in the common action envelope are Unix microseconds. Hyperliquid
payload times remain milliseconds. Preserve the returned preparation token and
exact signed payload for reconciliation and retries. Never change RPCs, allocate
another nonce or create a replacement order automatically after a transport error.

Perpetual Prepare accepts optional `perpetual.executionTtlMs`. It sets the signed
action's expiry relative to `preparedAt`; omission preserves 60,000 ms for manual
requests. Scalping execution should pass its configured lifetime (30,000 ms by
default) explicitly. Values must be positive integers no larger than
9,223,372,036,854 ms; explicit null is invalid. A lifetime is neither a request
timeout nor a position holding duration. The adapter may return an already-expired
preparation when the requested lifetime is shorter than preparation work; it must
not be signed or sent. Submit and Agent verification check the exact lifetime,
and retries retain the original signed expiry. No Run ID is required for this field.

`execution.Params` / `execution.Result` retain the earlier internal Spread
contract. They are not the DTOs for the new public Prepare handler. The separate
`prepare` package owns composition of existing swap and perpetual DTOs without
creating package import cycles or facade aliases.

## Execution observation

`TradeHub.Execution.Subscribe` and `TradeHub.Execution.Unsubscribe` observe an
execution previously submitted through TradeHub. They require authenticated
WebSocket requests. Supported executions are one Base Mainnet/Sepolia EVM
transaction (ERC-20 approval or Uniswap V3 swap), or one configured Sui swap:
Cetus, Turbos and Momentum on Testnet/Mainnet, and Bluefin on Mainnet.

Subscribe with `{"executionId":"exec_..."}`. The acknowledgement returns
`executionId` and an opaque `subscriptionKey`. Notifications use event type `exs`
and the `SubscriptionEvent` type in this package. Unsubscribe with both identifiers.

- `pending`: the chain-specific completion condition has not been recorded.
- `success`: EVM receipt status is 1, or the Sui transaction succeeded and was
  included in a checkpoint, with its actual fill reconciled into OMS.
- `failed`: EVM receipt status is 0 (`transaction_reverted`), or the checkpointed
  Sui transaction failed (`transaction_failed`).
- `kind: "error"`: observation was interrupted, with sanitized code
  `observation_unavailable`. This is not a failed transaction.

Transient EVM approvals complete at receipt inclusion. Persisted swaps require
reconciled finalized EVM receipts or Sui checkpoint inclusion. Transaction
completion (`snapshot.status`) and accounting completion (`snapshot.oms.pnl.status`)
are independent. A successful transaction with pending PnL keeps its watch open;
the server releases it after PnL becomes `realized`, `not_applicable` or
`unavailable`, keeping the connection open. A fresh subscription rechecks EVM receipts or reads the committed
Sui OMS snapshot. Sui reconciliation continues independently of subscriptions and
resumes from persisted pending submissions after a server restart.
Unknown or other-account executions return `not_found`. Unsupported chain/leg
shapes return `unsupported`; executions without persisted submission identity
return `invalid_parameter`. No additional observation credit charge is configured.

The composed WebSocket SDK exposes `module.Execution()`. Register through
`Subscribe(ctx, execution.SubscribeParams{ExecutionID: id})`, consume `Events()`
and `Errors()`, and explicitly call `Unsubscribe` if stopping before completion.
Duplicate active SDK calls reuse the same local handle. `snapshot.Complete()` identifies the final snapshot that closes
the event channel; `snapshot.Status.Terminal()` identifies transaction completion.
Release transaction reservations at transaction completion without waiting for accounting; transport/delivery interruptions are reported on `Errors()`.
Treat interruption separately from the terminal status. The SDK does not
automatically resubscribe or send transactions.

Gateway closes the session after five minutes without application traffic.
Protocol heartbeat traffic does not keep it alive. Other application traffic on
the same connection does. A pending transaction can therefore outlive its watch;
resubscribe explicitly to obtain a fresh snapshot. Call `module.Close()` when
the caller is finished with the connection.

## Sui Submit

Use `SubmitParams` with the original Prepare `executionId` and `payloadDigest`.
Set `SignedPayload{ChainFamily: ChainFamilySui, Encoding: PayloadEncodingBase64,
TransactionBytes: preparedBytes, Signatures: []string{signatureBase64}}`.
`TransactionBytes` is the unchanged full TransactionData BCS in base64;
`signatureBase64` encodes the 97-byte Ed25519 `flag || signature || publicKey`.
The server checks the signature, ownership, exact prepared bytes and gas owner.
The signing intent digest and the returned base58 onchain `TransactionID` differ.

The relay supports configured Sui Testnet and Mainnet swaps with sender-paid
gas. Network identity must match the authorized preparation and the selected relay.
Prepare writes Execution only; first valid Submit writes an OMS order and a
submitted/pending record before broadcast. Quantities in the request are atomic
integer strings; OMS order quantity uses input-token decimals. Submit acceptance
does not mean a successful swap or a fill. Retry an uncertain Submit with identical params, including
after Prepare TTL if a claim was committed; do not rebuild or spend reserved coins
until the onchain result has been reconciled.

## Sui OMS snapshots and explicit Close

`ExecutionSnapshot.Onchain.Checkpoint` identifies checkpoint inclusion; EVM block
fields are absent for Sui. `ExecutionSnapshot.OMS` contains the string `orderId`,
optional `openExecutionId`, actual `fill`, separate `fee`, and `pnl`. Amounts in
fill, gas and settlement quantity fields are exact integer strings in the specified
token's smallest units. PnL monetary fields are decimal USDC strings. The
corresponding OMS records use exact token-denominated decimal strings.

For a full Close, set `SubmitParams.OpenExecutionID` to the successful Open's
execution ID on the first Submit. Keep it unchanged on every retry. The server
requires the same account, chain/network, reversed token pair and wallet path,
and the Open's entire acquired amount as Close input. A locked OMS parent order
prevents concurrent Close orders from claiming the same Open. A checkpointed
failed Close releases that claim for a new Close execution. Partial Close and
cross-chain inventory matching are not implemented in this Sui endpoint.

Successful checkpoint reconciliation records actual input/output quantities from
the verified venue event and balance changes. Failed transactions have no fill.
Both record `fee.amount = computationCost + storageCost - storageRebate` in MIST
(`fee.assetId` is SUI, `fee.decimals` is 9), including a negative amount when rebates
exceed costs. No quote-currency conversion is performed.

`oms.pnl` describes only the settlement attributable to this execution. It does
not contain the setting's cumulative PnL or unrealized valuation. An explicit
`openExecutionId` is not required for accounting: managed Scalping orders use
their assigned setting's weighted average basis; ordinary swaps use the account,
wallet, chain/network and actual asset book, excluding assigned inventory.

```json
{
  "status": "realized",
  "settlement": {
    "assetId": "0x2::sui::SUI",
    "quantity": {"amount": "86646", "decimals": 9},
    "currency": "USDC",
    "costBasis": "0.001",
    "proceeds": "0.00098",
    "amount": "-0.00002"
  }
}
```

- `pending`: accounting inputs or calculation are pending. A confirmed transaction
  remains successful; this status does not authorize a new submission.
- `realized`: includes `settlement`, even for zero profit. Quantity is the matched
  disposed asset quantity. Cost is the proportional acquisition basis; proceeds
  include disposal fees. Amount is proceeds minus allocated cost, excluding gas.
- `not_applicable`: no matched disposal (for example an opening-only swap or a
  failed transaction). There is no numeric settlement.
- `unavailable`: required cost/conversion data are missing, chronology is ambiguous,
  or the operation is unsupported. There is no numeric settlement.

Only `realized` includes `settlement`. Quantity uses atomic units; cost, proceeds
and amount use exact rational arithmetic internally and decimal strings rounded
once to at most 18 fractional digits (nearest, ties away from zero). Independently
rounded displayed fields can differ at the last digit. Recognized USDC uses the
same 1:1 reporting policy as Console. Native-funded ordinary cycles use the
settlement's actual conversion ratio; unsupported conversions are unavailable.
Trading fees are counted once, and gas remains in the separate `fee` field.

Replay uses owned OMS event revisions and a bounded cache, not a second accounting
ledger. A fresh subscription re-evaluates corrected facts. Current replay limits
are 1,000 orders and 20,000 facts per scoped wallet history; exceeding them returns
`unavailable`. Unresolved preceding chronology waits as `pending` instead of
inventing a cost basis. No new transaction is prepared or sent by observation.
