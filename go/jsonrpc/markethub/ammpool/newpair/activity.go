package newpair

// Activity contains cumulative observed activity throughout the NewPair monitoring
// lifetime, including time before listing and temporary exclusion. Counts and
// amounts are decimal strings. No historical completeness guarantee is implied.
type Activity struct {
	StartedAt          int64             `json:"startedAt"`
	UpdatedAt          int64             `json:"updatedAt"`
	SwapCount          string            `json:"swapCount"`
	Token0ToToken1     ActivitySwaps     `json:"token0ToToken1"`
	Token1ToToken0     ActivitySwaps     `json:"token1ToToken0"`
	LiquidityAdded     ActivityLiquidity `json:"liquidityAdded"`
	LiquidityRemoved   ActivityLiquidity `json:"liquidityRemoved"`
	NetToken0Liquidity *string           `json:"netToken0Liquidity"`
	NetToken1Liquidity *string           `json:"netToken1Liquidity"`
}

type ActivitySwaps struct {
	Count        string  `json:"count"`
	Token0Amount *string `json:"token0Amount"`
	Token1Amount *string `json:"token1Amount"`
	// VolumeUSD uses one side of each observed swap and excludes gas. Nil means
	// at least one counted swap could not be priced; a partial sum is never exposed.
	VolumeUSD *string `json:"volumeUsd"`
}

type ActivityLiquidity struct {
	Count        string  `json:"count"`
	Token0Amount *string `json:"token0Amount"`
	Token1Amount *string `json:"token1Amount"`
}

// CloneActivity copies an activity value without sharing optional amount pointers.
//
// Version:
//   - 2026-09-21: Added.
func CloneActivity(a *Activity) *Activity {
	if a == nil {
		return nil
	}
	c := *a
	copyString := func(p **string) {
		if *p != nil {
			v := **p
			*p = &v
		}
	}
	for _, p := range []**string{&c.Token0ToToken1.Token0Amount, &c.Token0ToToken1.Token1Amount, &c.Token0ToToken1.VolumeUSD, &c.Token1ToToken0.Token0Amount, &c.Token1ToToken0.Token1Amount, &c.Token1ToToken0.VolumeUSD, &c.LiquidityAdded.Token0Amount, &c.LiquidityAdded.Token1Amount, &c.LiquidityRemoved.Token0Amount, &c.LiquidityRemoved.Token1Amount, &c.NetToken0Liquidity, &c.NetToken1Liquidity} {
		copyString(p)
	}
	return &c
}
