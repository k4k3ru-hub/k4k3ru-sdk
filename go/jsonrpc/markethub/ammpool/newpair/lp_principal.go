package newpair

// LPPrincipal describes principal across all liquidity ranges at one observed position.
// Amounts exclude uncollected fees. Percentages compare token quantities, not values.
type LPPrincipal struct {
	Token0      LPPrincipalToken `json:"token0"`
	Token1      LPPrincipalToken `json:"token1"`
	EvaluatedAt int64            `json:"evaluatedAt"`
	Position    Position         `json:"position"`
}

type LPPrincipalToken struct {
	Amount string `json:"amount"`
	// AmountPercentage is in [0,100], or null when both amounts are zero.
	AmountPercentage *string `json:"amountPercentage"`
}

// CloneLPPrincipal copies a principal observation without sharing mutable fields.
//
// Version:
//   - 2026-09-22: Added.
func CloneLPPrincipal(p *LPPrincipal) *LPPrincipal {
	if p == nil {
		return nil
	}
	c := *p
	c.Position.Details = append([]byte(nil), p.Position.Details...)
	if p.Token0.AmountPercentage != nil {
		v := *p.Token0.AmountPercentage
		c.Token0.AmountPercentage = &v
	}
	if p.Token1.AmountPercentage != nil {
		v := *p.Token1.AmountPercentage
		c.Token1.AmountPercentage = &v
	}
	return &c
}
