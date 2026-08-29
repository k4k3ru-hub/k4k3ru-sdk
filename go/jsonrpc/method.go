package jsonrpc

import (
	"fmt"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

const maxMethodLength = 64

type Method string

const (
	MethodAccountEmailRequestCredentialCreationOTP Method = "AccountEmail.RequestCredentialCreationOTP"
	MethodAccountEmailCreateCredential             Method = "AccountEmail.CreateCredential"
	MethodAccountEmailRequestSignInOTP             Method = "AccountEmail.RequestSignInOTP"
	MethodAccountEmailSignIn                       Method = "AccountEmail.SignIn"
	MethodAccountAPIRequestCredentialCreationOTP   Method = "AccountAPI.RequestCredentialCreationOTP"
	MethodAccountAPICreateCredential               Method = "AccountAPI.CreateCredential"
	MethodAccountAppGetUsageBalance                Method = "AccountApp.GetUsageBalance"
	MethodAccountAppListProducts                   Method = "AccountApp.ListProducts"
	MethodMarketHubListVenues                      Method = "MarketHub.ListVenues"
	MethodMarketHubListSymbols                     Method = "MarketHub.ListSymbols"
	MethodMarketHubAggregationSubscribe            Method = "MarketHub.Aggregation.Subscribe"
	MethodMarketHubAggregationUnsubscribe          Method = "MarketHub.Aggregation.Unsubscribe"
	MethodMarketHubAggregationGet                  Method = "MarketHub.Aggregation.Get"
	MethodPaymentOnchainCreateIntent               Method = "PaymentOnchain.CreateIntent"
	MethodPaymentOnchainGetIntent                  Method = "PaymentOnchain.GetIntent"
)

// Validate validates the JSON-RPC method.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-28: Added.
func (m Method) Validate() error {
	if m == "" {
		return fmt.Errorf(
			"failed to validate json rpc method: %w: method=empty",
			k4k3ruSDKAppError.InvalidParameter(),
		)
	}
	if len(m) > maxMethodLength {
		return fmt.Errorf(
			"failed to validate json rpc method: %w: method=too_long actual_length=%d max_length=%d",
			k4k3ruSDKAppError.InvalidParameter(),
			len(m),
			maxMethodLength,
		)
	}

	return nil
}
