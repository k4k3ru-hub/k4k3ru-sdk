package spread

type Leg struct {
	Venue              Venue      `json:"v"`
	MarketType         MarketType `json:"mt"`
	Side               Side       `json:"sd"`
	BaseAsset          Asset      `json:"ba"`
	QuoteAsset         Asset      `json:"qa"`
	Quantity           string     `json:"q"`
	VWAP               string     `json:"vwap"`
	QuoteAmount        string     `json:"qam"`
	ConsumedLevelCount uint16     `json:"lc"`
	BookVersion        uint64     `json:"bv"`
	Timestamp          int64      `json:"ts"`
}

type Route struct {
	RouteID        string      `json:"id"`
	Family         RouteFamily `json:"f"`
	Buy            Leg         `json:"b"`
	Sell           Leg         `json:"s"`
	GrossSpread    string      `json:"gs"`
	GrossSpreadBps string      `json:"gsb"`
}

type Result struct {
	AssetClass            AssetClass    `json:"ac"`
	Symbol                Symbol        `json:"s"`
	BaseAsset             Asset         `json:"ba"`
	QuoteAsset            Asset         `json:"qa"`
	Quantity              string        `json:"q"`
	MinimumGrossSpreadBps string        `json:"mgsb"`
	RouteFamilies         []RouteFamily `json:"rf"`
	SourceFilter          *SourceFilter `json:"sourceFilter,omitempty"`
	EvaluatedMarketCount  uint16        `json:"emc"`
	EvaluatedRouteCount   uint32        `json:"erc"`
	PricedRouteCount      uint32        `json:"prc"`
	EligibleRoutes        []Route       `json:"er"`
	EvaluatedAt           int64         `json:"ts"`
}
