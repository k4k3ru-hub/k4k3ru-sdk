package app

import "time"

// CreditPaymentParams identifies a payment grant and its immutable credit terms.
// ExpiresInDays is resolved once by the receiver, at the first successful grant.
type CreditPaymentParams struct {
	Source        string `json:"source"`
	IntentKind    string `json:"intentKind"`
	IntentID      uint64 `json:"intentId,string"`
	AccountID     uint64 `json:"accountId,string"`
	CreditTicks   uint64 `json:"creditTicks,string"`
	ExpiresInDays uint32 `json:"expiresInDays"`
}

type CreditPaymentResult struct {
	AccountID   uint64     `json:"accountId,string"`
	CreditID    uint64     `json:"creditId,string"`
	OperationID uint64     `json:"operationId,string"`
	CreditTicks uint64     `json:"creditTicks,string"`
	ExpiresAt   *time.Time `json:"expiresAt"`
}
