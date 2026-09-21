package newpair

// Fees describes the Pool swap rate at one observed position and caller condition.
// Rates are decimal fractions, not percentages. Token taxes and gas are excluded.
// Variable rates are historical reference values, not guarantees for future swaps.
type Fees struct {
	Model           string   `json:"model"`
	Token0ToToken1  FeeRate  `json:"token0ToToken1"`
	Token1ToToken0  FeeRate  `json:"token1ToToken0"`
	Source          string   `json:"source"`
	ObservedAt      int64    `json:"observedAt"`
	Position        Position `json:"position"`
	ReferenceSender *string  `json:"referenceSender"`
}

type FeeRate struct {
	Rate string `json:"rate"`
}

// CloneFees copies a fee observation without sharing mutable fields.
//
// Version:
//   - 2026-09-22: Added.
func CloneFees(f *Fees) *Fees {
	if f == nil {
		return nil
	}
	c := *f
	c.Position.Details = append([]byte(nil), f.Position.Details...)
	if f.ReferenceSender != nil {
		v := *f.ReferenceSender
		c.ReferenceSender = &v
	}
	return &c
}
