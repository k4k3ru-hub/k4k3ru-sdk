package scalping

type SubscribeResult struct {
	SubscriptionKey string `json:"subscriptionKey"`
}

type UnsubscribeParams struct {
	SubscriptionKey string `json:"subscriptionKey"`
}

type UnsubscribeResult struct {
	SubscriptionKey string `json:"subscriptionKey"`
}

type EventKind string

const (
	EventKindSnapshot EventKind = "snapshot"
	EventKindError    EventKind = "error"
)

type SubscriptionEvent struct {
	SubscriptionKey string       `json:"subscriptionKey"`
	Sequence        uint64       `json:"sequence"`
	Kind            EventKind    `json:"kind"`
	Snapshot        *Result      `json:"snapshot,omitempty"`
	Error           *StreamError `json:"error,omitempty"`
}

type StreamError struct {
	Code      string `json:"code"`
	Retryable bool   `json:"retryable"`
}
