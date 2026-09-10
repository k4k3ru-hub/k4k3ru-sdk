package swap

type Kind string

const (
	KindUnknown     Kind = ""
	KindExactInput  Kind = "exact-input"
	KindExactOutput Kind = "exact-output"
)

type PrepareStatus string

const (
	PrepareStatusUnknown          PrepareStatus = ""
	PrepareStatusReady            PrepareStatus = "ready"
	PrepareStatusApprovalRequired PrepareStatus = "approval-required"
)
