# WebSocket subscription failures

The `websocket` package exports two errors that callers can inspect with
`errors.Is`, including when an operation wraps the underlying error:

| Error | Meaning |
| --- | --- |
| `ErrConnectionClosed` | The shared transport interrupted an outstanding request or subscription |
| `ErrSubscriptionOverflow` | A TradeHub Scalping or Execution subscription could not retain another event |

Create clients through `websocket.NewModule` and use `module.Scalping()` or
`module.Execution()`. These errors do not cause automatic order submission or
subscription recovery. Callers must invalidate cached candidates, retain the
acknowledged execution ID, resubscribe and reconcile server state before continuing.
If the first Scalping acknowledgement was lost, reuse the same start parameters and
idempotency key. Subscription errors never prove a transaction failed or settled.

```go
if errors.Is(err, websocket.ErrConnectionClosed) ||
    errors.Is(err, websocket.ErrSubscriptionOverflow) {
    // Stop using cached candidates; schedule resubscription and reconciliation.
}
```

Execution and Scalping stream error events also retain their server `retryable`
flag. Unknown errors and `retryable: false` require separate handling. Close an
individual subscription to release it; closing the module closes the shared
connection and affects the other subscriptions.
