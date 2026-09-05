# K4K3RU SDK

K4K3RUサービスをUser Applicationから利用するための公開SDKです。

## Go SDK

Go moduleは`go`ディレクトリで公開されています。

```bash
go get github.com/k4k3ru-hub/k4k3ru-sdk/go@latest
```

JSON-RPCの共通Envelopeは`jsonrpc`、各メソッドの`params`と`result`はドメイン別パッケージからimportします。

```go
import (
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruAccountApp "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/account/app"
)
```

### WebSocket OrderBook購読

`websocket.NewModule`が返すComposition Rootから`OrderBook()`を取得し、統合板を購読できます。`Params`の`depth`は1〜20で、未指定時は3です。

```go
subscription, err := module.OrderBook().Subscribe(ctx, orderbook.Params{
	MarketType: orderbook.MarketTypeSpot,
	Symbol:     "BTC/USDC",
	Depth:      3,
})
if err != nil {
	return err
}
result := <-subscription.Events()
fmt.Printf("bids=%v asks=%v\n", result.Bids, result.Asks)
if err := module.OrderBook().Unsubscribe(ctx, subscription); err != nil {
	return err
}
```

接続先とCredential Providerは`websocket.ModuleConfig`で明示的に渡します。個別Venueは指定せず、必要に応じて`SourceFilter`でVenueカテゴリや流動性モデルを絞り込みます。

### WebSocket Spread購読

`Spread()`は、同一Canonical SymbolのSpot–Spot、Spot–Perp、Perp–Spot、Perp–Perp経路を、指定した基準資産数量で評価する購読クライアントです。

```go
subscription, err := module.Spread().Subscribe(ctx, spread.Params{
	Symbol:                "BTC/USDC",
	BaseAsset:             "BTC",
	Quantity:              "0.1",
	MinimumGrossSpreadBps: "5",
})
if err != nil {
	return err
}
result := <-subscription.Events()
fmt.Printf("eligible routes=%v\n", result.EligibleRoutes)
if err := module.Spread().Unsubscribe(ctx, subscription); err != nil {
	return err
}
```

`RouteFamilies`を省略すると4種類すべてを評価します。`SourceFilter`はBBO／OrderBookと同じVenueカテゴリ、流動性モデル、AMM chainによる絞り込みです。

## Goで署名認証付きリクエストを送る

署名が必要なメソッドでは、API CredentialのAPI keyと、作成時に取得したsecret keyを使用します。次の例は`AccountApp.GetUsageBalance`を呼び出す、実行可能な最小構成です。

認証情報は環境変数から読み込み、ソースコードへ直接記載しないでください。

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruAccountApp "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/account/app"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/signature"
)

const gatewayURL = "https://api.k4k3ru.com/"

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	apiKey := os.Getenv("K4K3RU_API_KEY")
	secretKey := os.Getenv("K4K3RU_SECRET_KEY")
	algorithm := os.Getenv("K4K3RU_SIGNATURE_ALGORITHM")
	if apiKey == "" || secretKey == "" || algorithm == "" {
		return errors.New("K4K3RU_API_KEY, K4K3RU_SECRET_KEY, and K4K3RU_SIGNATURE_ALGORITHM are required")
	}

	accountID, err := strconv.ParseUint(os.Getenv("K4K3RU_ACCOUNT_ID"), 10, 64)
	if err != nil || accountID == 0 {
		return errors.New("K4K3RU_ACCOUNT_ID must be a positive integer")
	}

	params, err := json.Marshal(k4k3ruAccountApp.GetUsageBalanceParams{
		AccountID: accountID,
	})
	if err != nil {
		return fmt.Errorf("failed to encode params: %w", err)
	}

	timestamp := time.Now().Unix()
	nonce, err := signature.GenerateNonce()
	if err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}
	payload, err := signature.BuildPayload(
		jsonrpc.MethodAccountAppGetUsageBalance,
		timestamp,
		nonce,
		params,
	)
	if err != nil {
		return fmt.Errorf("failed to build signature payload: %w", err)
	}

	requestSignature, err := sign(algorithm, secretKey, payload)
	if err != nil {
		return err
	}

	requestBody, err := json.Marshal(&jsonrpc.Request{
		ID:     json.RawMessage(`1`),
		Method: jsonrpc.MethodAccountAppGetUsageBalance,
		Params: params,
		Auth: &jsonrpc.Auth{
			APIKey:    apiKey,
			Timestamp: timestamp,
			Nonce:     nonce,
			Signature: requestSignature,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayURL, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	responseBody, readErr := io.ReadAll(httpResponse.Body)
	closeErr := httpResponse.Body.Close()
	if readErr != nil {
		return fmt.Errorf("failed to read response: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close response body: %w", closeErr)
	}
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unexpected HTTP status: status_code=%d", httpResponse.StatusCode)
	}

	var response jsonrpc.Response
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	if response.Error != nil {
		return fmt.Errorf("K4K3RU returned an error: %w", response.Error)
	}

	var result k4k3ruAccountApp.GetUsageBalanceResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return fmt.Errorf("failed to decode result: %w", err)
	}
	fmt.Printf("account_id=%d balance_ticks=%d\n", result.AccountID, result.BalanceTicks)
	return nil
}

func sign(algorithm string, secretKey string, payload []byte) (string, error) {
	switch algorithm {
	case "hmac-sha256":
		return signature.SignHMACSHA256(secretKey, payload)
	case "ed25519":
		return signature.SignEd25519(secretKey, payload)
	default:
		return "", fmt.Errorf("unsupported signature algorithm: %q", algorithm)
	}
}
```

実行時には、API Credential作成時に取得した値を設定します。

```bash
export K4K3RU_ACCOUNT_ID='123456789'
export K4K3RU_API_KEY='your-api-key'
export K4K3RU_SECRET_KEY='your-secret-key'
export K4K3RU_SIGNATURE_ALGORITHM='ed25519'
go run .
```

`K4K3RU_SIGNATURE_ALGORITHM`には`ed25519`または`hmac-sha256`を指定します。secret keyは、API Credential作成時に返されたunpadded Base64 URL encoding形式の値をそのまま設定してください。

### 署名処理の要点

1. メソッド固有の`params`をJSONへ変換します。
2. 現在時刻のUnix秒と、`signature.GenerateNonce()`で生成したnonceを用意します。
3. `signature.BuildPayload()`で正規化された署名対象を構築します。
4. Credentialのアルゴリズムに対応する関数で署名します。
5. API key、timestamp、nonce、signatureを`jsonrpc.Auth`へ設定します。

`BuildPayload()`へ渡した`method`、`timestamp`、`nonce`、`params`と、リクエストEnvelopeへ設定する値は必ず一致させてください。特に、署名後に`params`を再構築または変更すると署名検証に失敗します。また、nonceはリクエストごとに新しく生成し、API keyやsecret keyをログへ出力しないでください。
