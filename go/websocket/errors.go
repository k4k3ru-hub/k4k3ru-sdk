package websocket

import "errors"

// ErrConnectionClosed identifies interrupted requests and subscriptions. Recovery
// requires explicit resubscription and reconciliation by the caller.
var ErrConnectionClosed = errors.New("failed to receive websocket message: connection=closed")

// ErrSubscriptionOverflow identifies lost events caused by a full subscription
// buffer. Callers must discard cached candidates and restore a fresh snapshot.
var ErrSubscriptionOverflow = errors.New("failed to receive subscription: buffer=full")
