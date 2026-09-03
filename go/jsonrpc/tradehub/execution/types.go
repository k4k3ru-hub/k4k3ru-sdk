package execution

type IntentKind string

const (
	IntentKindUnknown         IntentKind = ""
	IntentKindAtomicArbitrage IntentKind = "atomic-arbitrage"
)

type SubmissionMode string

const (
	SubmissionModeUnknown       SubmissionMode = ""
	SubmissionModeTradeHubRelay SubmissionMode = "tradehub-relay"
)

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
)
