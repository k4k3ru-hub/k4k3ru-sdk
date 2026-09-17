package ammpool_test

import (
	"encoding/json"
	"fmt"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/ammpool"
)

// ExampleListParams demonstrates constructing an outbound list request.
//
// Version:
//   - 2026-09-17: Added.
func ExampleListParams() {
	params := (ammpool.ListParams{Chain: " Base ", Network: " Sepolia "}).Normalize()
	if err := params.Validate(); err != nil {
		fmt.Println(err)
		return
	}
	data, err := json.Marshal(params)
	if err != nil {
		fmt.Println(err)
		return
	}
	request := rpc.Request{ID: json.RawMessage(`"pools-1"`), Method: rpc.MethodTradeHubAMMPoolList, Params: data}
	encoded, err := json.Marshal(request)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(encoded))
	// Output: {"id":"pools-1","method":"TradeHub.AMMPool.List","params":{"chain":"base","network":"sepolia"}}
}
