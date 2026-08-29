package onchain

import "encoding/json"

type CreateIntentParams struct {
	ProductName string `json:"productName"`
}

type CreateIntentResult struct {
	IntentID  uint64          `json:"intentId,string"`
	AccountID uint64          `json:"accountId,string"`
	Status    string          `json:"status"`
	Chain     Chain           `json:"chain"`
	Network   Network         `json:"network"`
	Token     Token           `json:"token"`
	Amount    string          `json:"amount"`
	Address   string          `json:"address"`
	ExpiresAt string          `json:"expiresAt"`
	Metadata  json.RawMessage `json:"metadata"`
}
