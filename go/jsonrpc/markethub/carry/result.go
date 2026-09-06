package carry

type Leg struct {
	Venue              Venue      `json:"v"`
	MarketType         MarketType `json:"mt"`
	Side               Side       `json:"sd"`
	BaseAsset          Asset      `json:"ba"`
	BaseAssetID        string     `json:"bai,omitempty"`
	QuoteAsset         Asset      `json:"qa"`
	QuoteAssetID       string     `json:"qai,omitempty"`
	VenueSymbol        string     `json:"vs,omitempty"`
	PoolID             string     `json:"pid,omitempty"`
	Chain              Chain      `json:"c,omitempty"`
	Network            Network    `json:"n,omitempty"`
	Quantity           string     `json:"q"`
	VWAP               string     `json:"vwap"`
	QuoteAmount        string     `json:"qam"`
	ConsumedLevelCount uint16     `json:"lc"`
	BookVersion        uint64     `json:"bv"`
	Timestamp          int64      `json:"ts"`
}

type FundingObservation struct {
	Kind              string `json:"kind"`
	Rate              string `json:"rate"`
	IntervalMinutes   int32  `json:"intervalMinutes"`
	FundingTimestamp  int64  `json:"fundingTimestamp"`
	ReceivedTimestamp int64  `json:"receivedTimestamp"`
	MarkPrice         string `json:"markPrice"`
	EstimatedAmount   string `json:"estimatedAmount"`
}
type EntrySpread struct {
	Amount string `json:"amount"`
	Bps    string `json:"bps,omitempty"`
	Asset  Asset  `json:"asset"`
}
type FundingEstimate struct {
	Amount               string `json:"amount"`
	Bps                  string `json:"bps,omitempty"`
	AnnualizedRate       string `json:"annualizedRate,omitempty"`
	ReferenceNotional    string `json:"referenceNotional,omitempty"`
	Asset                Asset  `json:"asset"`
	HoldingPeriodMinutes uint32 `json:"holdingPeriodMinutes"`
	Model                string `json:"model"`
}
type Assessment struct {
	ExecutionFeasibility string `json:"executionFeasibility"`
	TradingFees          string `json:"tradingFees"`
	BorrowAvailability   string `json:"borrowAvailability"`
	BorrowCost           string `json:"borrowCost"`
	ExitCost             string `json:"exitCost"`
}
type Route struct {
	Selector        RouteSelector       `json:"route"`
	RouteID         string              `json:"id"`
	Family          RouteFamily         `json:"f"`
	Buy             Leg                 `json:"b"`
	Sell            Leg                 `json:"s"`
	BuyFunding      *FundingObservation `json:"buyFunding,omitempty"`
	SellFunding     *FundingObservation `json:"sellFunding,omitempty"`
	EntrySpread     EntrySpread         `json:"entrySpread"`
	FundingEstimate FundingEstimate     `json:"fundingEstimate"`
	Assessment      Assessment          `json:"assessment"`
}
type SearchResult struct {
	AssetClass                 AssetClass    `json:"ac"`
	Symbol                     Symbol        `json:"s"`
	BaseAsset                  Asset         `json:"ba"`
	QuoteAsset                 Asset         `json:"qa"`
	Quantity                   string        `json:"q"`
	HoldingPeriodMinutes       uint32        `json:"hpm"`
	MinimumEstimatedFundingBps string        `json:"mefb"`
	RouteFamilies              []RouteFamily `json:"rf"`
	SourceFilter               *SourceFilter `json:"sourceFilter,omitempty"`
	EvaluatedMarketCount       uint16        `json:"emc"`
	EvaluatedRouteCount        uint32        `json:"erc"`
	PricedRouteCount           uint32        `json:"prc"`
	FundingEvaluatedRouteCount uint32        `json:"ferc"`
	EligibleRoutes             []Route       `json:"er"`
	EvaluatedAt                int64         `json:"ts"`
}

// Availability describes whether an individual metric can be evaluated.
type Availability struct {
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	ObservedAt int64  `json:"observedAt,omitempty"`
	ValidUntil int64  `json:"validUntil,omitempty"`
}
type EvaluationAvailability struct {
	BuyPrice      Availability `json:"buyPrice"`
	SellPrice     Availability `json:"sellPrice"`
	BuyFunding    Availability `json:"buyFunding"`
	SellFunding   Availability `json:"sellFunding"`
	FundingAmount Availability `json:"fundingAmount"`
	FundingBps    Availability `json:"fundingBps"`
	EntrySpread   Availability `json:"entrySpread"`
}
type Result struct {
	AssetClass           AssetClass             `json:"ac"`
	Symbol               Symbol                 `json:"s"`
	BaseAsset            Asset                  `json:"ba"`
	QuoteAsset           Asset                  `json:"qa"`
	Quantity             string                 `json:"q"`
	HoldingPeriodMinutes uint32                 `json:"hpm"`
	Route                RouteSelector          `json:"route"`
	RouteID              string                 `json:"routeId"`
	EvaluationKey        string                 `json:"evaluationKey"`
	EvaluationID         string                 `json:"evaluationId"`
	Status               string                 `json:"status"`
	Buy                  *Leg                   `json:"buy,omitempty"`
	Sell                 *Leg                   `json:"sell,omitempty"`
	BuyFunding           *FundingObservation    `json:"buyFunding,omitempty"`
	SellFunding          *FundingObservation    `json:"sellFunding,omitempty"`
	FundingEstimate      *FundingEstimate       `json:"fundingEstimate,omitempty"`
	EntrySpread          *EntrySpread           `json:"entrySpread,omitempty"`
	Assessment           Assessment             `json:"assessment"`
	Availability         EvaluationAvailability `json:"availability"`
	EvaluatedAt          int64                  `json:"ts"`
}

// Params returns the fixed-route parameters identifying this result.
//
// Version:
//   - 2026-09-06: Reconstruct fixed-route subscription identity.
func (r Result) Params() Params {
	return (Params{AssetClass: r.AssetClass, Symbol: r.Symbol, BaseAsset: r.BaseAsset, Quantity: r.Quantity, HoldingPeriodMinutes: r.HoldingPeriodMinutes, Route: r.Route}).Normalize()
}
