package newpair

// TokenTaxes contains independent, nullable observations for both pool tokens.
// It excludes pool fees, gas, price impact and address-specific trade simulations.
type TokenTaxes struct {
	Token0 *TokenTax `json:"token0"`
	Token1 *TokenTax `json:"token1"`
}

// TokenTax describes base transfer-tax fractions at one observed position.
// Nil fields mean unconfirmed; "0" and false require evidence. CanChange and
// HasExemptions concern taxation, not all token permissions or token safety.
type TokenTax struct {
	BuyRate       *string  `json:"buyRate"`
	SellRate      *string  `json:"sellRate"`
	CanChange     *bool    `json:"canChange"`
	HasExemptions *bool    `json:"hasExemptions"`
	Source        string   `json:"source"`
	ObservedAt    int64    `json:"observedAt"`
	Position      Position `json:"position"`
}

// CloneTokenTax copies one observation without sharing pointers or position details.
//
// Version:
//   - 2026-09-22: Added.
func CloneTokenTax(v *TokenTax) *TokenTax {
	if v == nil {
		return nil
	}
	c := *v
	copyString := func(v *string) *string {
		if v == nil {
			return nil
		}
		c := *v
		return &c
	}
	copyBool := func(v *bool) *bool {
		if v == nil {
			return nil
		}
		c := *v
		return &c
	}
	c.BuyRate = copyString(v.BuyRate)
	c.SellRate = copyString(v.SellRate)
	c.CanChange = copyBool(v.CanChange)
	c.HasExemptions = copyBool(v.HasExemptions)
	c.Position.Details = append([]byte(nil), v.Position.Details...)
	return &c
}

// CloneTokenTaxes copies both token observations while preserving partial unknowns.
//
// Version:
//   - 2026-09-22: Added.
func CloneTokenTaxes(v *TokenTaxes) *TokenTaxes {
	if v == nil {
		return nil
	}
	return &TokenTaxes{Token0: CloneTokenTax(v.Token0), Token1: CloneTokenTax(v.Token1)}
}
