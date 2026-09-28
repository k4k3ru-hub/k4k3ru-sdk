package newpair

// ActivityWindows contains observed, minute-aligned periods independently of the
// monitoring-lifetime totals. Nil periods are unavailable, not zero activity.
type ActivityWindows struct {
	FiveMinutes     *ActivityShortWindow `json:"5m"`
	FifteenMinutes  *ActivityShortWindow `json:"15m"`
	OneHour         *ActivityWindow      `json:"1h"`
	TwentyFourHours *ActivityWindow      `json:"24h"`
}

// ActivityShortWindow adds a preceding, non-overlapping equal-duration comparison.
// Nil Comparison means observation history or period data is insufficient.
type ActivityShortWindow struct {
	ActivityWindow
	Comparison *ActivityComparison `json:"comparison"`
}

type ActivityComparison struct {
	From      int64          `json:"from"`
	To        int64          `json:"to"`
	SwapCount ActivityChange `json:"swapCount"`
	VolumeUSD ActivityChange `json:"volumeUsd"`
}

// ActivityChange preserves exact decimal values; delta and percentage may be
// negative. Percentage is (current-previous)/previous*100, truncated toward zero
// to 18 fractional places. A zero previous value has a nil percentage.
// Unknown previous/current USD values leave the unavailable fields nil.
type ActivityChange struct {
	Previous         *string `json:"previous"`
	Delta            *string `json:"delta"`
	ChangePercentage *string `json:"changePercentage"`
}

// ActivityWindow covers [From, To), in UTC Unix microseconds. ObservedFrom is
// the start of period aggregation within that interval, not a completeness claim.
// To identifies the last published minute boundary, not necessarily the present.
// Liquidity totals describe additions/removals, not reserve changes from swaps.
type ActivityWindow struct {
	From               int64             `json:"from"`
	To                 int64             `json:"to"`
	ObservedFrom       int64             `json:"observedFrom"`
	SwapCount          string            `json:"swapCount"`
	Token0ToToken1     ActivitySwaps     `json:"token0ToToken1"`
	Token1ToToken0     ActivitySwaps     `json:"token1ToToken0"`
	LiquidityAdded     ActivityLiquidity `json:"liquidityAdded"`
	LiquidityRemoved   ActivityLiquidity `json:"liquidityRemoved"`
	NetToken0Liquidity *string           `json:"netToken0Liquidity"`
	NetToken1Liquidity *string           `json:"netToken1Liquidity"`
}

// CloneActivityWindows copies period observations without sharing mutable values.
//
// Version:
//   - 2026-09-27: Added.
//   - 2026-09-28: Copy short periods and comparison values independently.
func CloneActivityWindows(w *ActivityWindows) *ActivityWindows {
	if w == nil {
		return nil
	}
	return &ActivityWindows{
		FiveMinutes: cloneActivityShortWindow(w.FiveMinutes), FifteenMinutes: cloneActivityShortWindow(w.FifteenMinutes),
		OneHour: cloneActivityWindow(w.OneHour), TwentyFourHours: cloneActivityWindow(w.TwentyFourHours),
	}
}

func cloneActivityShortWindow(w *ActivityShortWindow) *ActivityShortWindow {
	if w == nil {
		return nil
	}
	c := &ActivityShortWindow{ActivityWindow: *cloneActivityWindow(&w.ActivityWindow)}
	if w.Comparison != nil {
		v := *w.Comparison
		for _, change := range []*ActivityChange{&v.SwapCount, &v.VolumeUSD} {
			for _, p := range []**string{&change.Previous, &change.Delta, &change.ChangePercentage} {
				if *p != nil {
					value := **p
					*p = &value
				}
			}
		}
		c.Comparison = &v
	}
	return c
}

func cloneActivityWindow(w *ActivityWindow) *ActivityWindow {
	if w == nil {
		return nil
	}
	c := *w
	a := CloneActivity(&Activity{
		Token0ToToken1: w.Token0ToToken1, Token1ToToken0: w.Token1ToToken0,
		LiquidityAdded: w.LiquidityAdded, LiquidityRemoved: w.LiquidityRemoved,
		NetToken0Liquidity: w.NetToken0Liquidity, NetToken1Liquidity: w.NetToken1Liquidity,
	})
	c.Token0ToToken1, c.Token1ToToken0 = a.Token0ToToken1, a.Token1ToToken0
	c.LiquidityAdded, c.LiquidityRemoved = a.LiquidityAdded, a.LiquidityRemoved
	c.NetToken0Liquidity, c.NetToken1Liquidity = a.NetToken0Liquidity, a.NetToken1Liquidity
	return &c
}
