package jsonrpc

import "encoding/json"

type AccountAppListProductsParams struct {
	Page uint64 `json:"page"`
}

type AccountAppListProductsProduct struct {
	ID            uint64          `json:"id,string"`
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	CreditTicks   uint64          `json:"creditTicks,string"`
	BonusTicks    uint64          `json:"bonusTicks,string"`
	PriceAmount   uint64          `json:"priceAmount,string"`
	PriceCurrency string          `json:"priceCurrency"`
	ExpiresInDays uint32          `json:"expiresInDays"`
	PurchaseLimit uint32          `json:"purchaseLimit"`
	Description   *string         `json:"description"`
	MetaData      json.RawMessage `json:"metaData"`
}

type AccountAppListProductsResult struct {
	Products   []*AccountAppListProductsProduct `json:"products"`
	Page       uint64                           `json:"page"`
	Limit      uint64                           `json:"limit"`
	Total      uint64                           `json:"total"`
	TotalPages uint64                           `json:"totalPages"`
}
