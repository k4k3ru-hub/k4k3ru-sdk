# Bluefin manual perpetual contracts

Owning package: `github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/perpetual/bluefin`.

| RPC | Request | Response |
| --- | --- | --- |
| `TradeHub.Perpetual.Session.Create` | `SessionCreateParams` | `SessionResult` |
| `TradeHub.Perpetual.Session.Delete` | `SessionDeleteParams` | `SessionDeleteResult` |
| `TradeHub.Perpetual.Account.Get` | `AccountParams` | `AccountResult` |
| `TradeHub.Perpetual.Order.Get` | `OrderParams` | `OrderResult` |

Method constants are owned by `jsonrpc`. All calls require K4K3RU authentication. These models expose SDK-owned values; Bluefin protocol implementation types are not re-exported.

Writes use the existing `TradeHub.Execution.Prepare` and `TradeHub.Execution.Submit` entrances. `OrderIntent.PrepareParams(sessionID)` builds the common preparation request. `SessionID` is a transient authentication reference and is excluded from the stable intent.

```go
intent := bluefin.OrderIntent{
    Scope: bluefin.Scope{
        Venue: "bluefin", Network: "testnet", Symbol: "SUI/USDC",
        AccountAddress: walletAddress,
    },
    SignerAddress: walletAddress,
    Order: perpetual.OrderIntent{
        Side: "buy", Quantity: "1", LimitPrice: chosenLimit,
        TimeInForce: "ioc", ClientOrderID: allocatedClientID,
    },
    Margin: bluefin.Margin{Leverage: 1, Mode: "cross"},
    Salt: allocatedSalt, SignedAtMS: signedAtMillis, ExecutionTTLMS: 60_000,
}
params, err := intent.PrepareParams(currentSessionID)
```

The caller chooses price, size and margin terms. Do not treat this example as a price recommendation. Initial execution is Testnet-first, own-wallet Ed25519 signing, IOC limits and reduce-only closing. TradeHub rejects Mainnet execution until the Testnet gate is complete.

Before signing, decode `prepare.Result.Action.Payload` as `Prepared` and independently reconstruct its full intent, exact signing bytes, digest, order hash, contract and expiry. IOC, reduce-only and client ID are authenticated by the K4K3RU preparation proof although they are absent from Bluefin's own signing JSON. Forward `Action.SubmitParams` and add `SignedPayload.Action` with `Venue: "bluefin"`, the network, the current session ID and `Signature{SerializedSignature: ...}` as JSON.

Private keys stay in the caller. Login signatures use Sui PersonalMessage; Bluefin tokens are held only in TradeHub memory. Obtain another Agent login signature at expiry or after TradeHub restarts. Deleting a session closes authentication and monitoring; it does not cancel orders or delete evidence.

An acknowledgment is not a fill. Order.Get reconciles the owned execution without resending it. `FillsComplete` requires terminal quantity evidence, and `FeesComplete` requires explicit fee facts. Results may remain `unknown` or `review_required` when terminal cancellation cannot be recovered. Neither an empty history nor a flat account proves an IOC order was never sent.

For closing, read Account.Get and use the opposite side with `ReduceOnly: true` and quantity no greater than the observed remainder. Preserve the actual position's margin mode and leverage. No implicit leverage update is performed because Bluefin's leverage operation also cancels market orders.

Keep the local journal private and save the verified intent before the one possible Submit. Recovery needs the saved execution ID and a new session; it does not require persisting Bluefin tokens, session references or usable signatures. The Agent's `resume` performs read-only reconciliation.
