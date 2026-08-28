package jsonrpc

import (
	"encoding/json"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

type Response struct {
	ID     json.RawMessage    `json:"id,omitempty"`
	Result json.RawMessage    `json:"result,omitempty"`
	Error  *apperror.AppError `json:"error,omitempty"`
}
