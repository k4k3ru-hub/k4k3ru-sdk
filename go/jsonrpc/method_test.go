package jsonrpc

import (
	"errors"
	"strings"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestMethodValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		method  Method
		wantErr string
	}{
		{
			name:   "known method",
			method: MethodAccountEmailRequestCredentialCreationOTP,
		},
		{
			name:   "known account email sign in otp method",
			method: MethodAccountEmailRequestSignInOTP,
		},
		{
			name:   "known account email sign in method",
			method: MethodAccountEmailSignIn,
		},
		{
			name:   "known account api otp method",
			method: MethodAccountAPIRequestCredentialCreationOTP,
		},
		{
			name:   "known account api creation method",
			method: MethodAccountAPICreateCredential,
		},
		{
			name:   "known account app usage balance method",
			method: MethodAccountAppGetUsageBalance,
		},
		{
			name:   "known account app list products method",
			method: MethodAccountAppListProducts,
		},
		{
			name:   "known market hub list venues method",
			method: MethodMarketHubListVenues,
		},
		{
			name:   "known market hub list symbols method",
			method: MethodMarketHubListSymbols,
		},
		{
			name:   "known market hub bbo subscribe method",
			method: MethodMarketHubBBOSubscribe,
		},
		{
			name:   "known market hub bbo unsubscribe method",
			method: MethodMarketHubBBOUnsubscribe,
		},
		{
			name:   "known market hub bbo get method",
			method: MethodMarketHubBBOGet,
		},
		{
			name:   "known market hub order book subscribe method",
			method: MethodMarketHubOrderBookSubscribe,
		},
		{
			name:   "known market hub order book unsubscribe method",
			method: MethodMarketHubOrderBookUnsubscribe,
		},
		{
			name:   "known market hub order book get method",
			method: MethodMarketHubOrderBookGet,
		},
		{
			name:   "known market hub arbitrage subscribe method",
			method: MethodMarketHubArbitrageSubscribe,
		},
		{
			name:   "known market hub arbitrage unsubscribe method",
			method: MethodMarketHubArbitrageUnsubscribe,
		},
		{
			name:   "known payment onchain create intent method",
			method: MethodPaymentOnchainCreateIntent,
		},
		{
			name:   "known payment onchain get intent method",
			method: MethodPaymentOnchainGetIntent,
		},
		{
			name:   "known trade hub arbitrage subscribe method",
			method: MethodTradeHubArbitrageSubscribe,
		},
		{
			name:   "known trade hub arbitrage unsubscribe method",
			method: MethodTradeHubArbitrageUnsubscribe,
		},
		{
			name:   "known trade hub execution prepare method",
			method: MethodTradeHubExecutionPrepare,
		},
		{
			name:   "known trade hub execution submit method",
			method: MethodTradeHubExecutionSubmit,
		},
		{
			name:   "custom method",
			method: "Example.CustomMethod",
		},
		{
			name:   "maximum length",
			method: Method(strings.Repeat("a", maxMethodLength)),
		},
		{
			name:    "empty",
			wantErr: "failed to validate json rpc method: err_code=\"invalid_parameter\": method=empty",
		},
		{
			name:    "too long",
			method:  Method(strings.Repeat("a", maxMethodLength+1)),
			wantErr: "failed to validate json rpc method: err_code=\"invalid_parameter\": method=too_long actual_length=65 max_length=64",
		},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.method.Validate()
			if testCase.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate() error = nil")
			}
			if err.Error() != testCase.wantErr {
				t.Fatalf("Validate() error = %q, want %q", err.Error(), testCase.wantErr)
			}
			if !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
				t.Fatalf("errors.Is(Validate(), InvalidParameter()) = false")
			}
		})
	}
}
