package app

import "time"

// UsageHistoryParams uses exclusive descending ID pagination. Zero means the first page.
// AccountID is supplied by the trusted Console BFF, never by the browser.
type UsageHistoryParams struct {
	AccountID uint64 `json:"accountId,string"`
	BeforeID  uint64 `json:"beforeId,string,omitempty"`
	CreditID  uint64 `json:"creditId,string,omitempty"`
	Type      uint8  `json:"type,omitempty"`
}

type UsageCredit struct {
	ID           uint64     `json:"id,string"`
	Type         uint8      `json:"type"`
	BalanceTicks uint64     `json:"balanceTicks,string"`
	ExpiresAt    *time.Time `json:"expiresAt"`
	Description  *string    `json:"description"`
	CreatedAt    time.Time  `json:"createdAt"`
}
type UsageCreditEvent struct {
	ID          uint64    `json:"id,string"`
	OperationID uint64    `json:"operationId,string"`
	CreditID    uint64    `json:"creditId,string"`
	Type        uint8     `json:"type"`
	DeltaTicks  int64     `json:"deltaTicks,string"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}
type ListUsageCreditsResult struct {
	AccountID    uint64        `json:"accountId,string"`
	Items        []UsageCredit `json:"items"`
	NextBeforeID uint64        `json:"nextBeforeId,string,omitempty"`
}
type ListUsageCreditEventsResult struct {
	AccountID    uint64             `json:"accountId,string"`
	Items        []UsageCreditEvent `json:"items"`
	NextBeforeID uint64             `json:"nextBeforeId,string,omitempty"`
}
