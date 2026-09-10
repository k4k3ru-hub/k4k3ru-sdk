# Onchain payment wire contracts

This package provides JSON-RPC parameters and results. Applications supply their
transport and signer; these types do not send requests or enforce authorization.
Use a server version that implements the corresponding RPC.

## PaymentOnchain.ListIntents

Use `jsonrpc.MethodPaymentOnchainListIntents` with `ListIntentsParams` and decode
the response into `ListIntentsResult`.

```json
{"page":1}
```

The method requires a user signature. The server derives account ownership from
the authenticated credential; the public parameters do not contain `accountId`.
The response includes pending, confirming, completed and expired intents for that
account. Entries are ordered by `created_at DESC, id DESC` with 20 per page.
Page numbering starts at 1; `total` counts that account's intents and `totalPages`
is the ceiling of total / 20. An empty page returns `intents: []`, never null.
New payments can shift entries between pages; pages are not a historical snapshot.

Each entry includes its Intent ID, account ID, status, chain, network, symbol,
recipient address, amount, expiry, creation time and saved metadata. IDs are JSON
decimal strings; timestamps use RFC 3339. `amount` is an integer string in the
asset's smallest units (for example, 1 USDC is `"1000000"` for a six-decimal asset).
This follows the current GetIntent server unit contract; CreateIntent returns
decimal token units instead. Convert using the asset decimals without float math.

Keep the Intent ID to retrieve the same payment through `PaymentOnchain.GetIntent`.
Listing or reopening a payment must not create another Intent. Metadata describes
the terms saved at creation, independently of later catalog changes.
