package onchain

type GetIntentParams struct {
	IntentID uint64 `json:"intentId,string"`
}

type GetIntentResult struct {
	IntentID         uint64  `json:"intentId,string"`
	AccountID        uint64  `json:"accountId,string"`
	Status           string  `json:"status"`
	Chain            Chain   `json:"chain"`
	Network          Network `json:"network"`
	Token            Token   `json:"symbol"`
	RecipientAddress string  `json:"recipientAddress"`
	Amount           string  `json:"amount"`
	ExpiresAt        string  `json:"expiresAt"`
}
