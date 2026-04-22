# Pay - 微信 / 支付宝支付

基于 **标准库** 封装的微信支付 **APIv3** 与支付宝 **RSA2（OpenAPI）**，无额外第三方支付 SDK 依赖。子包拆分便于按需引用；`pkg/pay` 根包提供聚合入口 `Hub`。

## 功能概览

| 能力 | 微信 `pkg/pay/wechat` | 支付宝 `pkg/pay/alipay` |
|------|----------------------|-------------------------|
| 小程序 / 公众号 JSAPI 下单 + 调起参数 | `CreateJSAPIOrder`、`BuildClientPayParams` | — |
| Native 扫码下单 | `CreateNativeOrder` | `TradePrecreate`（当面付预创建） |
| APP 调起串 | — | `TradeAppPay` |
| 查单 / 关单 | `QueryOrderByOutTradeNo`、`CloseOrder` | `TradeQuery` |
| 退款 | `Refund` | `TradeRefund` |
| 支付回调 | `ParseTransactionNotify`、`WriteNotifyAck` | `ParseNotifyForm`、`VerifyNotify`、`WriteNotifyAck` |

**说明**：本模块覆盖直连商户常见路径；服务商模式、合单、分账、合单关单等需自行扩展或二次封装。

## 依赖与导入

```go
import (
    "github.com/liukunxin/go-infra/pkg/pay"
    "github.com/liukunxin/go-infra/pkg/pay/alipay"
    "github.com/liukunxin/go-infra/pkg/pay/wechat"
)
```

聚合注入（单侧可为 `nil`）：

```go
hub := pay.NewHub(wxCli, aliCli)
```

## 微信支付（APIv3）

### 配置

在商户平台配置 APIv3 密钥、申请商户 API 证书，并下载**微信支付平台证书**（用于回调验签）。

```go
wxCli, err := wechat.NewClient(wechat.Config{
    AppID:               "wx...",
    MchID:               "1230000109",
    CertificateSerialNo: "商户API证书序列号",
    PrivateKeyPEM:       string(商户API私钥PEM),
    APIv3Key:            "32位APIv3密钥32位APIv3密钥12", // 须正好 32 个字符
    PlatformCertPEM:     string(微信支付平台证书PEM),
    HTTPTimeout:         15 * time.Second,
})
```

### JSAPI / 小程序

1. 服务端下单（金额单位：**分**）：

```go
order, err := wxCli.CreateJSAPIOrder(ctx, wechat.JSAPIOrderRequest{
    Description: "商品描述",
    OutTradeNo:  "商户侧单号",
    NotifyURL:   "https://你的域名/notify/wechat",
    AmountFen:   1,
    OpenID:      "用户openid",
})
```

2. 生成前端调起参数（`wx.requestPayment` / 小程序支付）：

```go
params, err := wxCli.BuildClientPayParams(order.PrepayID)
// 将 params 序列化给前端使用
```

### Native（扫码）

```go
nat, err := wxCli.CreateNativeOrder(ctx, wechat.NativeOrderRequest{
    Description: "商品描述",
    OutTradeNo:  "商户侧单号",
    NotifyURL:   "https://你的域名/notify/wechat",
    AmountFen:   1,
})
// nat.CodeURL 生成二维码
```

### 查单、关单、退款

```go
q, err := wxCli.QueryOrderByOutTradeNo(ctx, outTradeNo)
err = wxCli.CloseOrder(ctx, outTradeNo)
rf, err := wxCli.Refund(ctx, wechat.RefundRequest{
    OutTradeNo:  outTradeNo,
    OutRefundNo: "退款单号",
    Reason:      "", // 空则不下发 reason 字段
    RefundFen:   1,
    TotalFen:    1,
})
```

### 支付回调

1. 读取 **原始 Body**（若使用框架中间件，需保证 Body 未被提前消费且与验签使用的一致）。
2. 调用 `ParseTransactionNotify`：内部会 **校验 `Wechatpay-*` 头签名**（依赖 `PlatformCertPEM`）、**校验时间戳在 ±5 分钟内**（降低重放风险）、并 **AES-GCM 解密** `resource`。
3. 业务幂等更新订单后，调用 `wechat.WriteNotifyAck` 返回 **HTTP 204**。

```go
func handleWechatNotify(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil { /* 记录日志 */ return }
    payload, err := wxCli.ParseTransactionNotify(r.Header, body)
    if err != nil { w.WriteHeader(http.StatusBadRequest); return }
    // TODO: 校验 payload.TradeState、金额与 out_trade_no，幂等更新订单
    wechat.WriteNotifyAck(w)
}
```

**已知限制**：当前按**单份**平台证书验签；证书轮换时需更新 `PlatformCertPEM`，或自行扩展为按 `Wechatpay-Serial` 选择对应公钥。

## 支付宝（RSA2）

### 配置

在开放平台创建应用，配置「应用私钥」与「支付宝公钥」（用于验签异步通知与同步响应）。

```go
aliCli, err := alipay.NewClient(alipay.Config{
    AppID:               "2021...",
    PrivateKeyPEM:       string(应用私钥PEM),
    AlipayPublicKeyPEM:  string(支付宝公钥PEM),
    IsProduction:        true,
    HTTPTimeout:         15 * time.Second,
    Charset:             "utf-8",
})
```

### APP 支付

生成 **orderStr**，交给客户端支付宝 SDK 调起：

```go
orderStr, err := aliCli.TradeAppPay(alipay.AppPayRequest{
    Subject:     "商品标题",
    OutTradeNo:  "商户侧单号",
    TotalAmount: "0.01", // 元，字符串
    NotifyURL:   "https://你的域名/notify/alipay",
})
```

### 当面付预创建（扫码）

```go
pre, err := aliCli.TradePrecreate(ctx, alipay.PrecreateRequest{
    Subject:     "商品标题",
    OutTradeNo:  "商户侧单号",
    TotalAmount: "0.01",
    NotifyURL:   "https://你的域名/notify/alipay",
})
// pre.QRCode 生成二维码
```

### 查单、退款

```go
q, err := aliCli.TradeQuery(ctx, alipay.TradeQueryRequest{OutTradeNo: outTradeNo})
rf, err := aliCli.TradeRefund(ctx, alipay.RefundRequest{
    OutTradeNo:   outTradeNo,
    RefundAmount: "0.01",
    RefundReason: "协商退款",
    OutRequestNo: "退款请求号",
})
```

### 异步通知

1. `alipay.ParseNotifyForm(r)` 解析表单。
2. `aliCli.VerifyNotify(values)` 验签。
3. 业务处理成功后 `alipay.WriteNotifyAck` 返回纯文本 **`success`**（小写）。

```go
func handleAlipayNotify(w http.ResponseWriter, r *http.Request) {
    v, err := alipay.ParseNotifyForm(r)
    if err != nil { return }
    if err := aliCli.VerifyNotify(v); err != nil { return }
    // TODO: 解析 trade_status、out_trade_no、total_amount 等，幂等更新订单
    alipay.WriteNotifyAck(w)
}
```

## 可选注入 `*http.Client`

默认：微信 / 支付宝各自使用**独立**的 `*http.Client{Timeout: HTTPTimeout}`。

若要与全库连接池、超时策略一致（例如使用 `pkg/http_client`），在 `wechat.Config`、`alipay.Config` 中设置 **`HTTPClient`**（非 `nil`）即可；此时 **`HTTPTimeout` 字段会被忽略**，以注入实例的 `Timeout`、`Transport` 为准。

与 `http_client` 配合示例（需已 `http_client.Init`，或直接使用 `NewClient` 返回值）：

```go
import (
    "github.com/liukunxin/go-infra/pkg/http_client"
    "github.com/liukunxin/go-infra/pkg/pay/wechat"
)

shared := http_client.NewClient(http_client.Config{Timeout: 30 * time.Second})
wxCli, err := wechat.NewClient(wechat.Config{
    // ... AppID、密钥等 ...
    HTTPClient: shared.HTTPClient(),
})
```

若进程内已通过 `http_client.Init` 持有全局客户端：

```go
wxCli, err := wechat.NewClient(wechat.Config{
    // ...
    HTTPClient: http_client.GetHttpClient().HTTPClient(),
})
```

## HTTP 与 Context

- 微信：`CreateJSAPIOrder`、`CreateNativeOrder`、`QueryOrderByOutTradeNo`、`CloseOrder`、`Refund` 均将 **`context` 传入底层 HTTP 请求**，支持超时与取消；GET 请求不再附带无意义的 `Content-Type: application/json`。
- 支付宝：`TradePrecreate`、`TradeQuery`、`TradeRefund` 同样传入 **`context`**；`TradeAppPay` 仅为本地签名，无 HTTP 调用。
- 注入的 `*http.Client` 同样遵守上述 `context` 行为（由标准库 `Do` 处理）。

## 错误处理

- 微信：API 业务错误解析为 `*wechat.APIError`，可与 `errors.Is(err, wechat.ErrAPI)` 配合判断。
- 支付宝：验签失败为 `alipay.ErrVerifySign`；网关返回非预期结构会包装为 `alipay.ErrAPI`。

## 安全建议

- 回调 URL 使用 HTTPS；校验订单金额、商户单号与用户身份的一致性。
- 微信：务必启用 `ParseTransactionNotify` 完整流程（验签 + 解密），勿仅解密不验签。
- 支付宝：务必 `VerifyNotify` 后再更新订单状态。
- 生产环境密钥与证书建议来自配置中心或密钥管理服务，勿写入代码库。

## 相关代码路径

- `pkg/pay/pay.go` — 包说明
- `pkg/pay/hub.go` — `Hub` 聚合
- `pkg/pay/wechat/` — 微信实现
- `pkg/pay/alipay/` — 支付宝实现
