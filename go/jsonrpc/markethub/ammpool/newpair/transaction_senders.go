package newpair

// TransactionSenderCount describes outer transaction senders, not people or holders.
// UniqueCount is a lower bound when ResolvedSwapCount is below the corresponding
// Activity count, and nil when no sender was resolved for a nonempty window.
type TransactionSenderCount struct {
	UniqueCount       *string `json:"uniqueCount"`
	ResolvedSwapCount string  `json:"resolvedSwapCount"`
}

type TransactionSenders struct {
	TransactionSenderCount
	Token0ToToken1 TransactionSenderCount `json:"token0ToToken1"`
	Token1ToToken0 TransactionSenderCount `json:"token1ToToken0"`
}

// CloneTransactionSenders copies nullable sender counts without sharing pointers.
//
// Version:
//   - 2026-09-29: Added.
func CloneTransactionSenders(v *TransactionSenders) *TransactionSenders {
	if v == nil {
		return nil
	}
	c := *v
	for _, count := range []*TransactionSenderCount{&c.TransactionSenderCount, &c.Token0ToToken1, &c.Token1ToToken0} {
		if count.UniqueCount != nil {
			n := *count.UniqueCount
			count.UniqueCount = &n
		}
	}
	return &c
}
