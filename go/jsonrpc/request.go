package jsonrpc

import (
	"encoding/json"
	"fmt"
)

type Request struct {
	ID     json.RawMessage `json:"id"`
	Method Method          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	Auth   *Auth           `json:"auth,omitempty"`
}

// Validate validates the JSON-RPC request.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-28: Added.
func (r *Request) Validate() error {
	if r == nil {
		return fmt.Errorf("failed to validate json rpc request: request=null")
	}
	if len(r.ID) == 0 {
		return fmt.Errorf("failed to validate json rpc request: id=empty")
	}
	if err := r.Method.Validate(); err != nil {
		return fmt.Errorf("failed to validate json rpc request: %w", err)
	}

	return nil
}
