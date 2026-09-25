package market

import (
	"testing"
)

func validAMMSwap() AMMSwap {
	fee := "0.0005"
	return AMMSwap{
		AssetClass: AssetClassCrypto, MarketType: MarketTypeSpot,
		Symbol: BTCUSDC, Venue: UniswapV3, Chain: ChainEthereum, PoolID: "0xpool",
		SwapID: "0xtx:7", TransactionID: "0xtx", EventIndex: "7",
		StateReferenceType: AMMStateReferenceTypeBlockNumber, StateReferenceValue: "123",
		Side: TradeSideBuy, Price: "118000.25", BaseQuantity: "1.25", QuoteQuantity: "147500.3125",
		EffectiveFeeRate: &fee, Timestamp: 1786845600000000,
	}
}

// TestAMMSwapValidateAcceptsSupportedChainReferences verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestAMMSwapValidateAcceptsSupportedChainReferences(t *testing.T) {
	tests := []struct {
		chain         Chain
		referenceType AMMStateReferenceType
	}{
		{chain: ChainEthereum, referenceType: AMMStateReferenceTypeBlockNumber},
		{chain: ChainRobinhood, referenceType: AMMStateReferenceTypeBlockNumber},
		{chain: ChainSolana, referenceType: AMMStateReferenceTypeSlot},
		{chain: ChainSui, referenceType: AMMStateReferenceTypeCheckpoint},
	}
	for _, test := range tests {
		t.Run(string(test.chain), func(t *testing.T) {
			swap := validAMMSwap()
			swap.Chain = test.chain
			swap.StateReferenceType = test.referenceType
			if err := swap.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

// TestRobinhoodAMMSwapReferences verifies block references for both Uniswap venues.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-07: Added.
func TestRobinhoodAMMSwapReferences(t *testing.T) {
	for _, venue := range []Venue{UniswapV3, UniswapV4} {
		for _, reference := range []AMMStateReferenceType{AMMStateReferenceTypeBlockNumber, AMMStateReferenceTypeSlot, AMMStateReferenceTypeCheckpoint} {
			swap := validAMMSwap()
			swap.Chain, swap.Venue, swap.Symbol = ChainRobinhood, venue, PONSUSDG
			swap.StateReferenceType = reference
			err := swap.Validate()
			if (err == nil) != (reference == AMMStateReferenceTypeBlockNumber) {
				t.Fatalf("venue=%s reference=%s error=%v", venue, reference, err)
			}
		}
	}
}

// TestAMMSwapValidateRejectsInvalidValues verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestAMMSwapValidateRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*AMMSwap)
	}{
		{name: "asset class", mutate: func(s *AMMSwap) { s.AssetClass = AssetClassUnknown }},
		{name: "market type", mutate: func(s *AMMSwap) { s.MarketType = MarketTypePerpetual }},
		{name: "symbol", mutate: func(s *AMMSwap) { s.Symbol = "" }},
		{name: "venue", mutate: func(s *AMMSwap) { s.Venue = Unknown }},
		{name: "chain", mutate: func(s *AMMSwap) { s.Chain = ChainNone }},
		{name: "pool id", mutate: func(s *AMMSwap) { s.PoolID = " " }},
		{name: "swap id", mutate: func(s *AMMSwap) { s.SwapID = "" }},
		{name: "transaction id", mutate: func(s *AMMSwap) { s.TransactionID = "" }},
		{name: "event index", mutate: func(s *AMMSwap) { s.EventIndex = "" }},
		{name: "state reference type", mutate: func(s *AMMSwap) { s.StateReferenceType = "block_hash" }},
		{name: "chain reference mismatch", mutate: func(s *AMMSwap) { s.Chain = ChainSolana }},
		{name: "state reference value", mutate: func(s *AMMSwap) { s.StateReferenceValue = "" }},
		{name: "side", mutate: func(s *AMMSwap) { s.Side = "unknown" }},
		{name: "price", mutate: func(s *AMMSwap) { s.Price = "NaN" }},
		{name: "base quantity", mutate: func(s *AMMSwap) { s.BaseQuantity = "0" }},
		{name: "quote quantity", mutate: func(s *AMMSwap) { s.QuoteQuantity = "-1" }},
		{name: "fee rate", mutate: func(s *AMMSwap) { value := "-0.1"; s.EffectiveFeeRate = &value }},
		{name: "timestamp", mutate: func(s *AMMSwap) { s.Timestamp = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			swap := validAMMSwap()
			test.mutate(&swap)
			if err := swap.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}
