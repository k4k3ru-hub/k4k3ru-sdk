package newpair

type ControlBoolFinding struct {
	Status string `json:"status"`
	Value  *bool  `json:"value"`
	Reason string `json:"reason,omitempty"`
}
type ControlStringFinding struct {
	Status string  `json:"status"`
	Value  *string `json:"value"`
	Reason string  `json:"reason,omitempty"`
}
type ControlOwnership struct {
	OwnerAddress ControlStringFinding `json:"ownerAddress"`
	Renounced    ControlBoolFinding   `json:"renounced"`
}
type ControlTransferRestrictions struct {
	BlacklistPresent                 ControlBoolFinding `json:"blacklistPresent"`
	AllowlistPresent                 ControlBoolFinding `json:"allowlistPresent"`
	AutomaticBuyerRestrictionPresent ControlBoolFinding `json:"automaticBuyerRestrictionPresent"`
	TransferLimitsPresent            ControlBoolFinding `json:"transferLimitsPresent"`
	PausePresent                     ControlBoolFinding `json:"pausePresent"`
	Paused                           ControlBoolFinding `json:"paused"`
	CanChange                        ControlBoolFinding `json:"canChange"`
}
type ControlMinting struct {
	Present         ControlBoolFinding   `json:"present"`
	CanMint         ControlBoolFinding   `json:"canMint"`
	Authorization   ControlStringFinding `json:"authorization"`
	RequiresBacking ControlBoolFinding   `json:"requiresBacking"`
	HasSupplyCap    ControlBoolFinding   `json:"hasSupplyCap"`
	SupplyCapRaw    ControlStringFinding `json:"supplyCapRaw"`
}
type ControlUpgrade struct {
	CanUpgrade ControlBoolFinding `json:"canUpgrade"`
}
type ControlBalanceControl struct {
	CanForceTransfer ControlBoolFinding `json:"canForceTransfer"`
	CanForceBurn     ControlBoolFinding `json:"canForceBurn"`
}
type TokenControls struct {
	ObservedAt           *int64                      `json:"observedAt"`
	Position             *Position                   `json:"position"`
	Ownership            ControlOwnership            `json:"ownership"`
	TransferRestrictions ControlTransferRestrictions `json:"transferRestrictions"`
	Minting              ControlMinting              `json:"minting"`
	Upgrade              ControlUpgrade              `json:"upgrade"`
	BalanceControl       ControlBalanceControl       `json:"balanceControl"`
}

// NewTokenControls creates independent nullable findings for a policy or job state.
// No blockchain observation position is manufactured.
//
// Version:
//   - 2026-09-23: Added.
func NewTokenControls(status, reason string) *TokenControls {
	v := &TokenControls{}
	v.Ownership.OwnerAddress = ControlStringFinding{Status: status, Reason: reason}
	v.Ownership.Renounced = ControlBoolFinding{Status: status, Reason: reason}
	v.TransferRestrictions.BlacklistPresent = ControlBoolFinding{Status: status, Reason: reason}
	v.TransferRestrictions.AllowlistPresent = ControlBoolFinding{Status: status, Reason: reason}
	v.TransferRestrictions.AutomaticBuyerRestrictionPresent = ControlBoolFinding{Status: status, Reason: reason}
	v.TransferRestrictions.TransferLimitsPresent = ControlBoolFinding{Status: status, Reason: reason}
	v.TransferRestrictions.PausePresent = ControlBoolFinding{Status: status, Reason: reason}
	v.TransferRestrictions.Paused = ControlBoolFinding{Status: status, Reason: reason}
	v.TransferRestrictions.CanChange = ControlBoolFinding{Status: status, Reason: reason}
	v.Minting.Present = ControlBoolFinding{Status: status, Reason: reason}
	v.Minting.CanMint = ControlBoolFinding{Status: status, Reason: reason}
	v.Minting.Authorization = ControlStringFinding{Status: status, Reason: reason}
	v.Minting.RequiresBacking = ControlBoolFinding{Status: status, Reason: reason}
	v.Minting.HasSupplyCap = ControlBoolFinding{Status: status, Reason: reason}
	v.Minting.SupplyCapRaw = ControlStringFinding{Status: status, Reason: reason}
	v.Upgrade.CanUpgrade = ControlBoolFinding{Status: status, Reason: reason}
	v.BalanceControl.CanForceTransfer = ControlBoolFinding{Status: status, Reason: reason}
	v.BalanceControl.CanForceBurn = ControlBoolFinding{Status: status, Reason: reason}
	return v
}

// CloneTokenControls copies all findings and observation coordinates.
//
// Version:
//   - 2026-09-23: Added.
func CloneTokenControls(v *TokenControls) *TokenControls {
	if v == nil {
		return nil
	}
	c := *v
	copyBool := func(f ControlBoolFinding) ControlBoolFinding {
		if f.Value != nil {
			x := *f.Value
			f.Value = &x
		}
		return f
	}
	copyString := func(f ControlStringFinding) ControlStringFinding {
		if f.Value != nil {
			x := *f.Value
			f.Value = &x
		}
		return f
	}
	if v.ObservedAt != nil {
		x := *v.ObservedAt
		c.ObservedAt = &x
	}
	if v.Position != nil {
		x := *v.Position
		x.Details = append([]byte(nil), v.Position.Details...)
		c.Position = &x
	}
	c.Ownership.OwnerAddress = copyString(v.Ownership.OwnerAddress)
	c.Ownership.Renounced = copyBool(v.Ownership.Renounced)
	c.TransferRestrictions.BlacklistPresent = copyBool(v.TransferRestrictions.BlacklistPresent)
	c.TransferRestrictions.AllowlistPresent = copyBool(v.TransferRestrictions.AllowlistPresent)
	c.TransferRestrictions.AutomaticBuyerRestrictionPresent = copyBool(v.TransferRestrictions.AutomaticBuyerRestrictionPresent)
	c.TransferRestrictions.TransferLimitsPresent = copyBool(v.TransferRestrictions.TransferLimitsPresent)
	c.TransferRestrictions.PausePresent = copyBool(v.TransferRestrictions.PausePresent)
	c.TransferRestrictions.Paused = copyBool(v.TransferRestrictions.Paused)
	c.TransferRestrictions.CanChange = copyBool(v.TransferRestrictions.CanChange)
	c.Minting.Present = copyBool(v.Minting.Present)
	c.Minting.CanMint = copyBool(v.Minting.CanMint)
	c.Minting.Authorization = copyString(v.Minting.Authorization)
	c.Minting.RequiresBacking = copyBool(v.Minting.RequiresBacking)
	c.Minting.HasSupplyCap = copyBool(v.Minting.HasSupplyCap)
	c.Minting.SupplyCapRaw = copyString(v.Minting.SupplyCapRaw)
	c.Upgrade.CanUpgrade = copyBool(v.Upgrade.CanUpgrade)
	c.BalanceControl.CanForceTransfer = copyBool(v.BalanceControl.CanForceTransfer)
	c.BalanceControl.CanForceBurn = copyBool(v.BalanceControl.CanForceBurn)
	return &c
}
