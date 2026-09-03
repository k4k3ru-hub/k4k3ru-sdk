package arbitrage

type SubmissionMode string

const (
	SubmissionModeUnknown       SubmissionMode = ""
	SubmissionModeTradeHubRelay SubmissionMode = "tradehub-relay"
)

// Validate validates the Trade Hub submission mode.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-03: Added.
func (m SubmissionMode) Validate() error {
	if m != SubmissionModeTradeHubRelay {
		return invalidParameterError("submission_mode=invalid")
	}
	return nil
}
