package newpair

// LPProtection describes all-position principal custody at an observed block.
// Stale values retain their original coordinates and must not be used as current.
type LPProtection struct {
	Status                string                  `json:"status"`
	Reason                string                  `json:"reason,omitempty"`
	Token0                LPProtectionToken       `json:"token0"`
	Token1                LPProtectionToken       `json:"token1"`
	AllPositionsProtected *bool                   `json:"allPositionsProtected"`
	EarliestUnlockAt      *int64                  `json:"earliestUnlockAt"`
	CanWeakenProtection   LPProtectionBoolFinding `json:"canWeakenProtection"`
	ObservedAt            *int64                  `json:"observedAt"`
	Position              *Position               `json:"position"`
}

type LPProtectionToken struct {
	LockedLiquidityPercentage               *string `json:"lockedLiquidityPercentage"`
	PermanentlyProtectedLiquidityPercentage *string `json:"permanentlyProtectedLiquidityPercentage"`
}

type LPProtectionBoolFinding struct {
	Status string `json:"status"`
	Value  *bool  `json:"value"`
	Reason string `json:"reason,omitempty"`
}

// CloneLPProtection copies all nullable values and observation coordinates.
//
// Version:
//   - 2026-09-23: Added.
func CloneLPProtection(v *LPProtection) *LPProtection {
	if v == nil {
		return nil
	}
	c := *v
	for _, p := range []**string{&c.Token0.LockedLiquidityPercentage, &c.Token0.PermanentlyProtectedLiquidityPercentage, &c.Token1.LockedLiquidityPercentage, &c.Token1.PermanentlyProtectedLiquidityPercentage} {
		if *p != nil {
			x := **p
			*p = &x
		}
	}
	for _, p := range []**bool{&c.AllPositionsProtected, &c.CanWeakenProtection.Value} {
		if *p != nil {
			x := **p
			*p = &x
		}
	}
	for _, p := range []**int64{&c.ObservedAt, &c.EarliestUnlockAt} {
		if *p != nil {
			x := **p
			*p = &x
		}
	}
	if v.Position != nil {
		x := *v.Position
		x.Details = append([]byte(nil), v.Position.Details...)
		c.Position = &x
	}
	return &c
}
