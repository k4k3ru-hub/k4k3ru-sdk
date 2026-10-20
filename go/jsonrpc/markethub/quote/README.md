# MarketHub.Quote.Get

Trade estimate向けの、1方向・1数量に対する市場別見積り契約です。
公開型はこの`quote`パッケージが所有し、HTTPクライアントは
`httpclient.NewClient(...).MarketHub().Quote().Get(ctx, params)`で組み立てます。

**実装段階:** SDKとアプリ側のMarketHub算出・Gateway・Console接続を実装済みです。
SDK公開版への依存更新・Docker反映は別途必要です。SDK追加だけでは稼働APIは切り替わりません。

## 取得元と価格

サーバーはAggregatorに保持したOrderBook／Pool状態と、保持済みの手数料情報を使います。
見積り要求ごとの外部RPC、Tx構築、simulation、発注は行いません。
市場間の状態は必ずしも同時刻ではなく、約定を保証する見積りではありません。

- `netPrice`はQuote/Base単価。売買手数料と、指定数量による価格影響を含みます。
- `fees`は既にnet値へ反映した内訳です。再度差し引きません。gasは含みません。
- `sentAt`はGatewayが送信する時刻（Unix μs）。市場状態の更新時刻ではありません。
- `status`は数量計算の可否を表します。`live`／`stale`の鮮度判定ではありません。

## Request

| 項目 | 必須 | 意味 |
|---|---|---|
| `assetType` | はい | `crypto`。省略時の補完なし |
| `marketType` | はい | `spot`／`perpetual`。`perp`は拒否 |
| `symbol` | はい | canonical pair（例：`SUI/USDC`）。既知シンボル一覧への限定なし |
| `side` | はい | `buy`／`sell` |
| `quantity` | はい | 正の数量。下表の単位に従う |
| `markets` | はい | 1〜64対象。明示的な重複は拒否 |
| `feeAccounts` | いいえ | 対象市場と同じVenue・Networkの取引口座。初期はHyperliquidのみ |

| 市場・方向 | quantityの単位 | SUI/USDCでquantity=10の場合 |
|---|---|---|
| Spot Buy | Quote支払数量 | 10 USDCを支払ってSUIを買う |
| Spot Sell | Base支払数量 | 10 SUIを売る |
| Perpetual Buy | Base建て契約数量 | 10 SUI相当を買う |
| Perpetual Sell | Base建て契約数量 | 10 SUI相当を売る |

Perpetualの数量は証拠金額・レバレッジ・Venue固有のlot数ではありません。
Requestに`kind`や`exact-output`の指定は追加しません。
既存の算定処理へ接続する際、Perpetual Buyは内部でBase数量固定の経路を使います。

子struct:

```go
// finance/market.Quantity: 値 = amount / 10^decimals
type Quantity struct {
    Amount   string `json:"amount"`
    Decimals uint8  `json:"decimals"`
}

// finance/market.MarketTarget: instrument省略時は登録市場から解決
type MarketTarget struct {
    Venue       market.Venue `json:"venue"`
    Network     core.Network `json:"network"`
    Chain       core.Chain   `json:"chain,omitempty"`
    PoolID      string       `json:"poolId,omitempty"`
    VenueSymbol string       `json:"venueSymbol,omitempty"`
}

// quote.FeeAccount
type FeeAccount struct {
    Venue   market.Venue `json:"venue"`
    Network core.Network `json:"network"`
    Address string       `json:"address"`
}
```

`amount`は最大384桁の非負整数文字列、`decimals`は0〜255の明示値です。
入力数量のゼロは拒否します。Tokenの最小単位やVenueの刻みへの対応はサーバーが検証します。
`poolId`と`venueSymbol`は同時指定できません。Networkの暗黙補完はありません。
各対象は複数Poolへ展開でき、解決後の対象も既存Scalpingと同じ最大64市場です。
対象外・未登録市場の自動探索はこのAPIの責務ではありません。

`FeeAccount.address`は取引口座の20-byte hexアドレスで、署名鍵ではありません。
同じVenue・Networkへの複数口座指定や、marketsにない口座指定は拒否します。
省略時は標準料率を使い、指定口座の料率が取得できない場合は標準料率へ切り替えません。

必須項目の省略・null、未知フィールド、重複キー、不正数量はJSON decodeで拒否します。
`Normalize`は入力のsliceをコピーし、MarketType・Side・市場スコープを小文字、Symbolを大文字へそろえます。
instrument IDの大文字小文字は保存します。入力を変更せず、無効値を既定値へ置換しません。

Spot Buy:

```json
{
  "assetType": "crypto",
  "marketType": "spot",
  "symbol": "SUI/USDC",
  "side": "buy",
  "quantity": {"amount": "100", "decimals": 0},
  "markets": [
    {"venue": "cetus", "chain": "sui", "network": "mainnet"},
    {"venue": "turbos", "chain": "sui", "network": "mainnet"}
  ]
}
```

Perpetual Sell（標準料率、10 SUI相当）:

```json
{
  "assetType": "crypto",
  "marketType": "perpetual",
  "symbol": "SUI/USDC",
  "side": "sell",
  "quantity": {"amount": "10", "decimals": 0},
  "markets": [{"venue": "hyperliquid", "network": "mainnet", "venueSymbol": "SUI"}]
}
```

## Result

`marketType`・`symbol`・`side`・`quantity`を返し、`quotes[]`は解決済み市場ごとの値です。
`MarketRef`はMarketTargetと同じフィールドですが、`poolId`／`venueSymbol`のどちらかが必須で、
Poolなら`chain`も必須です。結果の配列順をBestの判定に使用しません。

| status | netPrice | Spot pay / receive | fees | Best比較 |
|---|---|---|---|---|
| `vwap` | 指定全数量の手数料込み単価 | 両方必須 | 算定済み内訳のみ | 対象 |
| `fallback_reference` | 手数料込みの最良気配／Pool参考単価 | 省略 | 省略 | 対象外 |
| `unavailable` | 省略 | 省略 | 省略 | 対象外 |

Perpetualは`vwap`でも`netPayQuantity`／`netReceiveQuantity`を返しません。
契約数量と`netPrice`・`fees`を使います。SpotのPayは入力側、Receiveは受取側のToken数量です。
不明な値をゼロにせず省略します。手数料の内訳だけ不明でも、net値を算定できれば`vwap`を返せます。

```go
type Fees struct {
    Swap  *Fee `json:"swap,omitempty"`  // LP + protocol
    Taker *Fee `json:"taker,omitempty"` // 即時板執行
}
type Fee struct {
    Token    FeeToken        `json:"token"`
    Quantity market.Quantity `json:"quantity"`
}
type FeeToken struct {
    AssetID string `json:"assetId"`
    Symbol  string `json:"symbol"`
}
```

Feeの数量ゼロは既知のゼロ手数料として許可します。Tokenは市場のchain/network、
またはvenue/network名前空間で識別します。同名Symbolだけで同一資産とは判断しません。
`fees`に未知の内訳しかなければ、空オブジェクトではなく項目自体を省略します。

次は形式説明用のPerpetual応答例です。実データではありません。

```json
{
  "marketType": "perpetual",
  "symbol": "SUI/USDC",
  "side": "sell",
  "quantity": {"amount": "10", "decimals": 0},
  "quotes": [{
    "market": {"venue": "hyperliquid", "network": "mainnet", "venueSymbol": "SUI"},
    "status": "vwap",
    "netPrice": "1.999",
    "fees": {"taker": {
      "token": {"assetId": "USDC", "symbol": "USDC"},
      "quantity": {"amount": "10000", "decimals": 6}
    }}
  }],
  "sentAt": 1792454400123000
}
```

HTTPクライアントは結果の必須値・status整合性に加え、要求方向・数量・シンボル・市場スコープを検証します。
数量は浮動小数点へ変換せず比較し、Sui Pool IDは先頭ゼロの表現差を許容します。
自動リトライ・認証情報の自動取得は行いません。

## 料金・利用上限と料率更新

未認証はIP・methodごとに直近60秒で3回無料。認証済みはaccount・methodごとに60回まで、
最初の3回無料・以降1要求100 creditsです。銘柄・市場・数量・APIキーの変更で枠は増えません。
複数市場やunavailableも1要求です。Consoleは無料で同じaccount・methodの枠を共有します。

`feeAccounts`は認証済みAPIだけで指定可能です。未認証の口座指定はエラー、Consoleは標準料率です。
口座指定は最新の要求から5分間、背景更新の対象になります。既存workerが60秒間隔で更新し、
成功から5分以内の料率を使います。要求は取得完了を待たず、未取得・期限切れはunavailableです。
要求の繰返しで料率の有効期限は延長しません。指定口座の失敗を標準料率へ切り替えません。
