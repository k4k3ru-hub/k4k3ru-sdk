package scalping

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observations "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// RunEvent replaces the complete view for one acknowledged stream generation.
// A stream error invalidates actionable candidates, without settling OMS orders.
type RunEvent struct {
	ExecutionID     string       `json:"executionId"`
	SubscriptionKey string       `json:"subscriptionKey"`
	Sequence        uint64       `json:"sequence"`
	Kind            EventKind    `json:"kind"`
	Snapshot        *RunSnapshot `json:"snapshot,omitempty"`
	Error           *StreamError `json:"error,omitempty"`
}

type RunSnapshot struct {
	EvaluationID     string                `json:"evaluationId"`
	MarketType       market.MarketType     `json:"marketType"`
	Symbol           market.Symbol         `json:"symbol"`
	EvaluatedAt      int64                 `json:"evaluatedAt"`
	PriceEvaluatedAt *int64                `json:"priceEvaluatedAt,omitempty"`
	Metrics          *observations.Metrics `json:"metrics,omitempty"`
	Entry            RunEvaluation         `json:"entry"`
	// Orders contains all unsettled or uncertain initial orders, never closed history.
	Orders []RunOrder `json:"orders"`
}

type RunEvaluation struct {
	Status  EvaluationStatus      `json:"status"`
	Markets []RunMarketEvaluation `json:"markets"`
	Reasons []string              `json:"reasons,omitempty"`
}

type RunMarketEvaluation struct {
	Price     observations.MarketPrice `json:"price"`
	Status    EvaluationStatus         `json:"status"`
	Candidate *RunCandidate            `json:"candidate,omitempty"`
	Trigger   string                   `json:"trigger,omitempty"`
	ReturnBPS *string                  `json:"returnBps,omitempty"`
	Reasons   []string                 `json:"reasons,omitempty"`
}

type RunCandidate struct {
	CandidateID string `json:"candidateId"`
	Revision    uint64 `json:"revision"`
	// ExpiresAt is omitted when no observation age constraint supplies a deadline.
	ExpiresAt *int64 `json:"expiresAt,omitempty"`
}

type RunOrderStatus string

const (
	RunOrderPending     RunOrderStatus = "pending"
	RunOrderHolding     RunOrderStatus = "holding"
	RunOrderSettling    RunOrderStatus = "settling"
	RunOrderUnavailable RunOrderStatus = "unavailable"
)

type RunOrder struct {
	// OrderID identifies the initial OMS order, encoded exactly as a decimal string.
	OrderID           string           `json:"orderId"`
	Revision          string           `json:"revision"`
	Status            RunOrderStatus   `json:"status"`
	Market            market.MarketRef `json:"market"`
	AccountAddress    string           `json:"accountAddress"`
	Side              Side             `json:"side"`
	RemainingQuantity *market.Quantity `json:"remainingQuantity,omitempty"`
	// EntryValue is the allocated Quote basis for the remaining Base quantity.
	// It is acquisition cost for Spot Buy, sale proceeds for Spot Sell, or entry
	// notional for Perpetual. Trading fees must follow the product's accounting.
	EntryValue *market.Quantity `json:"entryValue,omitempty"`
	AcquiredAt *int64           `json:"acquiredAt,omitempty"`
	// Exit prices for Spot Buy are observed for this revision's full remaining
	// quantity, independently of Observation.Sell. VWAP net receipts can size
	// settlement constraints; reference prices cannot supply a quantity estimate.
	Exit RunEvaluation `json:"exit"`
}

// Validate checks a complete replacement snapshot and its independently managed orders.
//
// Version:
//   - 2026-10-01: Added.
func (r RunSnapshot) Validate() error {
	const op = "validate scalping run snapshot"
	if err := v.Text(op, "evaluation_id", r.EvaluationID, 128); err != nil {
		return err
	}
	if err := validateMarketType(r.MarketType); err != nil {
		return fmt.Errorf("failed to validate scalping run snapshot: %w", err)
	}
	if err := r.Symbol.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping run snapshot: %w", err)
	}
	parts := strings.Split(string(r.Symbol), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == parts[1] {
		return v.Invalid(op, "symbol", "invalid")
	}
	if r.EvaluatedAt <= 0 || r.PriceEvaluatedAt != nil && (*r.PriceEvaluatedAt <= 0 || *r.PriceEvaluatedAt > r.EvaluatedAt) {
		return v.Invalid(op, "timestamps", "out_of_range")
	}
	if r.Orders == nil {
		return v.Invalid(op, "orders", "null")
	}
	if err := validateConsolidatedMetrics(r.Metrics); err != nil {
		return fmt.Errorf("failed to validate scalping run snapshot: %w", err)
	}
	if err := validateRunEvaluation(r.Entry, r.EvaluatedAt, r.MarketType, false); err != nil {
		return err
	}
	seen := make(map[string]bool, len(r.Orders))
	for _, order := range r.Orders {
		id, err := strconv.ParseUint(order.OrderID, 10, 64)
		if err != nil || id == 0 || strconv.FormatUint(id, 10) != order.OrderID || seen[order.OrderID] {
			return v.Invalid(op, "order_id", "invalid")
		}
		seen[order.OrderID] = true
		if err := v.Text(op, "revision", order.Revision, 128); err != nil {
			return err
		}
		if err := order.Market.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping run order: %w", err)
		}
		if err := v.Text(op, "account_address", order.AccountAddress, 256); err != nil {
			return err
		}
		if order.Side != SideBuy && order.Side != SideSell {
			return v.Invalid(op, "side", "invalid")
		}
		switch order.Status {
		case RunOrderPending, RunOrderHolding, RunOrderSettling, RunOrderUnavailable:
		default:
			return v.Invalid(op, "order_status", "invalid")
		}
		for _, q := range []*market.Quantity{order.RemainingQuantity, order.EntryValue} {
			if q != nil {
				if err := q.Validate(); err != nil {
					return fmt.Errorf("failed to validate scalping run order: %w", err)
				}
			}
		}
		if order.Status == RunOrderHolding && (order.RemainingQuantity == nil || strings.Trim(order.RemainingQuantity.Amount, "0") == "") {
			return v.Invalid(op, "remaining_quantity", "empty")
		}
		if order.AcquiredAt != nil && (*order.AcquiredAt <= 0 || *order.AcquiredAt > r.EvaluatedAt) {
			return v.Invalid(op, "acquired_at", "out_of_range")
		}
		if err := validateRunEvaluation(order.Exit, r.EvaluatedAt, r.MarketType, true); err != nil {
			return err
		}
		if order.Status != RunOrderHolding && order.Exit.Status == EvaluationStatusMatched {
			return v.Invalid(op, "order_exit", "invalid")
		}
		if order.Exit.Status == EvaluationStatusMatched && r.Entry.Status == EvaluationStatusMatched {
			return v.Invalid(op, "settlement_priority", "invalid")
		}
	}
	return nil
}

func validateRunEvaluation(e RunEvaluation, at int64, kind market.MarketType, exit bool) error {
	const op = "validate scalping run evaluation"
	if e.Markets == nil {
		return v.Invalid(op, "markets", "null")
	}
	matched := false
	seen := make(map[market.MarketRef]bool)
	for _, m := range e.Markets {
		if err := m.Price.Market.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping run evaluation: %w", err)
		}
		ref := m.Price.Market.Normalize()
		if seen[ref] {
			return v.Invalid(op, "duplicate_market", "invalid")
		}
		seen[ref] = true
		// Reuse price validation without the old mandatory trade-age/candidate rule.
		price := MarketEvaluation{Price: m.Price, Status: EvaluationStatusNotMatched}
		if m.Price.Status == observations.PriceStatusUnavailable {
			price.Status = EvaluationStatusUnavailable
			price.Reasons = []string{"price_unavailable"}
		}
		if err := validateEvaluation(price, at, kind); err != nil {
			return fmt.Errorf("failed to validate scalping run evaluation: %w", err)
		}
		if err := validateRunEvaluationStatus(m.Status, m.Reasons); err != nil {
			return err
		}
		if (m.Status == EvaluationStatusMatched) != (m.Candidate != nil) {
			return v.Invalid(op, "candidate", "invalid")
		}
		if m.Candidate != nil {
			matched = true
			if m.Price.Status == observations.PriceStatusUnavailable {
				return v.Invalid(op, "matched_price", "invalid")
			}
			if err := v.Text(op, "candidate_id", m.Candidate.CandidateID, 128); err != nil {
				return err
			}
			if m.Candidate.Revision == 0 || m.Candidate.ExpiresAt != nil && *m.Candidate.ExpiresAt <= at {
				return v.Invalid(op, "candidate", "invalid")
			}
		}
		if !exit && (m.Trigger != "" || m.ReturnBPS != nil) {
			return v.Invalid(op, "entry_trigger", "invalid")
		}
		if exit {
			switch m.Trigger {
			case "", "condition", "take_profit", "stop_loss", "maximum_holding":
			default:
				return v.Invalid(op, "trigger", "invalid")
			}
			if (m.Status == EvaluationStatusMatched) != (m.Trigger != "") {
				return v.Invalid(op, "exit_trigger", "invalid")
			}
		}
		if m.ReturnBPS != nil {
			if _, err := v.Number(op, "return_bps", *m.ReturnBPS, false, true); err != nil {
				return err
			}
		}
	}
	if err := validateRunEvaluationStatus(e.Status, e.Reasons); err != nil {
		return err
	}
	if (e.Status == EvaluationStatusMatched) != matched {
		return v.Invalid(op, "evaluation_status", "invalid")
	}
	return nil
}

func validateRunEvaluationStatus(status EvaluationStatus, reasons []string) error {
	const op = "validate scalping run evaluation status"
	switch status {
	case EvaluationStatusMatched:
		if len(reasons) != 0 {
			return v.Invalid(op, "matched_reasons", "invalid")
		}
	case EvaluationStatusNotMatched:
	case EvaluationStatusUnavailable:
		if len(reasons) == 0 {
			return v.Invalid(op, "reasons", "empty")
		}
	default:
		return v.Invalid(op, "status", "invalid")
	}
	for _, reason := range reasons {
		if err := v.Text(op, "reason", reason, 128); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks stream identity, sequence and the mutually exclusive event bodies.
//
// Version:
//   - 2026-10-01: Added.
func (e RunEvent) Validate() error {
	const op = "validate scalping run event"
	if err := validateSubscriptionReference(op, e.ExecutionID, e.SubscriptionKey); err != nil {
		return err
	}
	if e.Sequence == 0 {
		return v.Invalid(op, "sequence", "empty")
	}
	switch e.Kind {
	case EventKindSnapshot:
		if e.Snapshot == nil || e.Error != nil {
			return v.Invalid(op, "snapshot", "invalid")
		}
		return e.Snapshot.Validate()
	case EventKindError:
		if e.Snapshot != nil || e.Error == nil {
			return v.Invalid(op, "error", "invalid")
		}
		return v.Text(op, "code", e.Error.Code, 64)
	default:
		return v.Invalid(op, "kind", "invalid")
	}
}

// UnmarshalJSON decodes one validated replacement snapshot or stream error atomically.
//
// Version:
//   - 2026-10-01: Added.
func (e *RunEvent) UnmarshalJSON(data []byte) error {
	if e == nil {
		return v.Invalid("decode scalping run event", "destination", "null")
	}
	type wire RunEvent
	var decoded wire
	if err := decodeRunJSON(data, &decoded, "executionId", "subscriptionKey", "sequence", "kind"); err != nil {
		return fmt.Errorf("failed to decode scalping run event: %w", err)
	}
	value := RunEvent(decoded)
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping run event: %w", err)
	}
	*e = value
	return nil
}
