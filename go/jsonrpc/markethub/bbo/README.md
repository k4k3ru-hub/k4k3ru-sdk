# BBO subscription keys

`Params.SubscriptionKey()` generates a canonical key. `ParseSubscriptionKey(key)`
restores the normalized parameters without a network request.

```go
params, err := bbo.ParseSubscriptionKey(key)
if err != nil {
    return err
}
// Use params for the corresponding Subscribe or Unsubscribe request.
```

Import the owning package:
`github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/bbo`.

Parsing accepts only canonical keys: invalid values, duplicate filters, unexpected
fields, and noncanonical casing or ordering return an inspectable
`apperror.InvalidParameter()` error. The parser regenerates the key to verify an
exact match. Symbols may contain punctuation, including colons.

The result preserves normalized subscription conditions, not the original input
formatting or omitted defaults. Future parameter changes must update key generation,
parsing, and round-trip tests together.
