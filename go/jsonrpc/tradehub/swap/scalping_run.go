package swap

import (
	"strconv"
	"strings"
)

// ScalpingRunReference assigns a swap to a Run, and a settlement to its initial OMS order.
// Prepare seals this reference without creating an order. Submit validates ownership
// and remaining inventory; it does not reevaluate market conditions.
type ScalpingRunReference struct {
	ExecutionID     string `json:"executionId"`
	PositionOrderID string `json:"positionOrderId,omitempty"`
	OrderRevision   string `json:"orderRevision,omitempty"`
}

// Normalize trims reference identifiers without changing their case.
//
// Version:
//   - 2026-10-02: Added.
func (r ScalpingRunReference) Normalize() ScalpingRunReference {
	r.ExecutionID = strings.TrimSpace(r.ExecutionID)
	r.PositionOrderID = strings.TrimSpace(r.PositionOrderID)
	r.OrderRevision = strings.TrimSpace(r.OrderRevision)
	return r
}

// Validate checks the execution and paired settlement identifiers.
//
// Version:
//   - 2026-10-02: Added.
func (r ScalpingRunReference) Validate() error {
	r = r.Normalize()
	if len(r.ExecutionID) == 0 || len(r.ExecutionID) > 128 {
		return invalidPrepareParameter("scalping_execution_id=invalid")
	}
	for _, c := range r.ExecutionID {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return invalidPrepareParameter("scalping_execution_id=invalid")
		}
	}
	if (r.PositionOrderID == "") != (r.OrderRevision == "") {
		return invalidPrepareParameter("scalping_order_reference=invalid")
	}
	if r.PositionOrderID != "" {
		id, err := strconv.ParseUint(r.PositionOrderID, 10, 64)
		if err != nil || id == 0 || strconv.FormatUint(id, 10) != r.PositionOrderID {
			return invalidPrepareParameter("position_order_id=invalid")
		}
		if len(r.OrderRevision) != 64 {
			return invalidPrepareParameter("order_revision=invalid")
		}
		for _, c := range r.OrderRevision {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
				return invalidPrepareParameter("order_revision=invalid")
			}
		}
	}
	return nil
}
