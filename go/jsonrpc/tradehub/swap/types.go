package swap

type Kind string

const (
	KindUnknown     Kind = ""
	KindExactInput  Kind = "exact-input"
	KindExactOutput Kind = "exact-output"
)
