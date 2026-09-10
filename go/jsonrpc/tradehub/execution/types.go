package execution

type IntentKind string

const (
	IntentKindUnknown         IntentKind = ""
	IntentKindAtomicArbitrage IntentKind = "atomic-arbitrage"
	IntentKindSpread          IntentKind = "spread"
)

type SubmissionMode string

const (
	SubmissionModeUnknown       SubmissionMode = ""
	SubmissionModeClientDirect  SubmissionMode = "client-direct"
	SubmissionModeTradeHubRelay SubmissionMode = "tradehub-relay"
)

// Validate validates an execution submission mode.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-08: Added.
func (m SubmissionMode) Validate() error {
	if m != SubmissionModeClientDirect && m != SubmissionModeTradeHubRelay {
		return invalidParameterError("submission_mode=invalid")
	}
	return nil
}

type Status string

const (
	StatusUnknown           Status = ""
	StatusAwaitingSignature Status = "awaiting-signature"
	StatusRejected          Status = "rejected"
)

type RejectionReason string

const (
	RejectionReasonUnknown                   RejectionReason = ""
	RejectionReasonSimulationFailed          RejectionReason = "simulation-failed"
	RejectionReasonOpportunityExpired        RejectionReason = "opportunity-expired"
	RejectionReasonNetProfitBelowMinimum     RejectionReason = "net-profit-below-minimum"
	RejectionReasonSlippageAboveMaximum      RejectionReason = "slippage-above-maximum"
	RejectionReasonExecutionCostAboveMaximum RejectionReason = "execution-cost-above-maximum"
)

type ChainFamily string

const (
	ChainFamilyUnknown ChainFamily = ""
	ChainFamilySui     ChainFamily = "sui"
	ChainFamilyEVM     ChainFamily = "evm"
	ChainFamilySolana  ChainFamily = "solana"
)

type PayloadEncoding string

const (
	PayloadEncodingUnknown PayloadEncoding = ""
	PayloadEncodingHex     PayloadEncoding = "hex"
	PayloadEncodingBase64  PayloadEncoding = "base64"
)
