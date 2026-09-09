# EVM allowance workflow

This package checks ERC-20 balances and allowances and can ensure a required allowance by coordinating one approval transaction.

The SDK does not store private keys or choose an RPC implementation. Applications inject a trusted spender registry, transaction builder, signer, sender, and receipt waiter. `CheckOperation` is read-only. `EnsureOperation` may sign and send an approval transaction, so callers should invoke it during explicit strategy initialization rather than silently during quote handling.

```go
check, err := allowance.NewCheckOperation(reader, spenderRegistry)
if err != nil {
	return err
}

ensure, err := allowance.NewEnsureOperation(allowance.EnsureOperationDeps{
	CheckOperation:     check,
	TransactionBuilder: builder,
	TransactionSigner:  localSigner,
	TransactionSender:  sender,
	ReceiptWaiter:      receiptWaiter,
})
if err != nil {
	return err
}

result, err := ensure.Execute(ctx, allowance.EnsureParams{
	CheckParams: allowance.CheckParams{
		Chain:          core.ChainBase,
		Network:        core.NetworkSepolia,
		Token:          usdcAddress,
		Owner:          localSigner.Address(),
		Spender:        uniswapRouterAddress,
		RequiredAmount: "10000",
	},
	Policy:      allowance.ApprovalPolicyFixed,
	FixedAmount: "100000000",
})
```

Use `ApprovalPolicyExact` to approve only the current requirement. `ApprovalPolicyFixed` sets an explicit reusable limit. `ApprovalPolicyMax` is opt-in and should be used only when the caller accepts the increased spender risk.
