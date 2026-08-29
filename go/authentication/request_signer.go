package authentication

import (
	"encoding/json"
	"fmt"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKSignature "github.com/k4k3ru-hub/k4k3ru-sdk/go/signature"
)

// SignRequest signs JSON-RPC request parameters and constructs authentication metadata.
//
// Parameters:
//   - method: JSON-RPC method.
//   - params: encoded request parameters.
//   - credential: current API credential.
//   - timestamp: request timestamp in Unix seconds.
//   - nonce: unique request nonce.
//
// Returns:
//   - JSON-RPC authentication metadata.
//   - Validation or signing error.
//
// Version:
//   - 2026-08-29: Added.
func SignRequest(method k4k3ruSDKJSONRPC.Method, params json.RawMessage, credential Credential, timestamp int64, nonce string) (*k4k3ruSDKJSONRPC.Auth, error) {
	if err := credential.Validate(); err != nil {
		return nil, fmt.Errorf("failed to sign json rpc request: %w", err)
	}
	payload, err := k4k3ruSDKSignature.BuildPayload(method, timestamp, nonce, params)
	if err != nil {
		return nil, fmt.Errorf("failed to sign json rpc request: %w", err)
	}
	var requestSignature string
	switch credential.SignatureAlgorithm {
	case SignatureAlgorithmHMACSHA256:
		requestSignature, err = k4k3ruSDKSignature.SignHMACSHA256(credential.SecretKey, payload)
	case SignatureAlgorithmEd25519:
		requestSignature, err = k4k3ruSDKSignature.SignEd25519(credential.SecretKey, payload)
	default:
		return nil, fmt.Errorf("failed to sign json rpc request: signature_algorithm=invalid")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to sign json rpc request: %w", err)
	}
	return &k4k3ruSDKJSONRPC.Auth{
		APIKey:    credential.APIKey,
		Timestamp: timestamp,
		Nonce:     nonce,
		Signature: requestSignature,
	}, nil
}
