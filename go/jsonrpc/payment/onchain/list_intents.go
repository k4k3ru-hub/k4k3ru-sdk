package onchain

import "encoding/json"

// ListIntentsParams selects a page belonging to the authenticated signing account.
// Page starts at 1. The server uses a fixed page size of 20.
type ListIntentsParams struct {
	Page uint64 `json:"page"`
}

// ListIntentsIntent describes a payment without exposing signing or wallet secrets.
type ListIntentsIntent struct {
	IntentID         uint64  `json:"intentId,string"`
	AccountID        uint64  `json:"accountId,string"`
	Status           string  `json:"status"`
	Chain            Chain   `json:"chain"`
	Network          Network `json:"network"`
	Token            Token   `json:"symbol"`
	RecipientAddress string  `json:"recipientAddress"`
	// Amount is an unsigned decimal integer string in the token's smallest units.
	Amount    string `json:"amount"`
	ExpiresAt string `json:"expiresAt"`
	CreatedAt string `json:"createdAt"`
	// Metadata contains the intent's saved terms, not the current product catalog.
	Metadata json.RawMessage `json:"metadata"`
}

type ListIntentsResult struct {
	Intents    []*ListIntentsIntent `json:"intents"`
	Page       uint64               `json:"page"`
	Limit      uint64               `json:"limit"`
	Total      uint64               `json:"total"`
	TotalPages uint64               `json:"totalPages"`
}
