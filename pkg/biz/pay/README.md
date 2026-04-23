# Pay - 聚合支付

基于**标准库**封装的聚合支付模块，覆盖微信支付（APIv3）、支付宝（RSA2）、
Apple 内购（App Store Server API）与京东支付，无额外第三方支付 SDK 依赖。
子包拆分便于按需引用；根包 `Hub` 提供统一聚合入口。

## 功能概览

| 能力 | 微信 `wechat` | 支付宝 `alipay` | Apple `applepay` | 京东 `jdpay` |
|------|:---:|:---:|:---:|:---:|
| 下单（JSAPI/H5/PC/APP/扫码） | ✓ | ✓ | — | ✓ |
| 查单 | ✓ | ✓ | ✓ | ✓ |
| 关单 | ✓ | — | — | ✓ |
| 退款 | ✓ | ✓ | — | ✓ |
| 退款查询 | — | — | ✓ | ✓ |
| 异步回调验签 | ✓ | ✓ | ✓(通知解析) | ✓ |
| 交易 JWS 解码/验签 | — | — | ✓ | — |
| 订阅状态查询 | — | — | ✓ | — |
| 延长续订日期 | — | — | ✓ | — |

## 快速开始

```go
import (
    "github.com/liukunxin/go-infra/pkg/biz/pay"
    "github.com/liukunxin/go-infra/pkg/biz/pay/wechat"
    "github.com/liukunxin/go-infra/pkg/biz/pay/alipay"
    "github.com/liukunxin/go-infra/pkg/biz/pay/applepay"
    "github.com/liukunxin/go-infra/pkg/biz/pay/jdpay"
)

hub := pay.NewHub(
    pay.WithWechat(wxCli),       // 可选，nil 时 hub.Wechat() 返回 nil
    pay.WithAlipay(aliCli),
    pay.WithApplePay(appleCli),
    pay.WithJDPay(jdCli),
)

// 渠道路由示例
switch provider {
case pay.ProviderWechat:
    hub.Wechat().CreateJSAPIOrder(...)
case pay.ProviderApplePay:
    hub.ApplePay().GetTransactionInfo(...)
}
```

---

## 微信支付（APIv3）

### 配置

```go
wxCli, err := wechat.NewClient(wechat.Config{
    AppID:               "wx...",
    MchID:               "1230000109",
    CertificateSerialNo: "商户API证书序列号",
    PrivateKeyPEM:       string(商户API私钥PEM),
    APIv3Key:            "32位APIv3密钥",
    PlatformCertPEM:     string(微信支付平台证书PEM),
})
```

### 主要能力

```go
// JSAPI/小程序下单
order, err := wxCli.CreateJSAPIOrder(ctx, wechat.JSAPIOrderRequest{
    Description: "商品描述", OutTradeNo: "单号", NotifyURL: "https://...",
    AmountFen: 100, OpenID: "openid",
})
params, err := wxCli.BuildClientPayParams(order.PrepayID) // 传给前端调起支付

// Native 扫码
nat, err := wxCli.CreateNativeOrder(ctx, wechat.NativeOrderRequest{...})
// nat.CodeURL 生成二维码

// 查单 / 关单 / 退款
q, err   := wxCli.QueryOrderByOutTradeNo(ctx, outTradeNo)
_        = wxCli.CloseOrder(ctx, outTradeNo)
rf, err  := wxCli.Refund(ctx, wechat.RefundRequest{...})

// 支付回调（验签 + 解密一步到位）
payload, err := wxCli.ParseTransactionNotify(r.Header, body)
wechat.WriteNotifyAck(w)
```

---

## 支付宝（RSA2）

### 配置

```go
aliCli, err := alipay.NewClient(alipay.Config{
    AppID:              "2021...",
    PrivateKeyPEM:      string(应用私钥PEM),
    AlipayPublicKeyPEM: string(支付宝公钥PEM),
    IsProduction:       true,
})
```

### 主要能力

```go
orderStr, err := aliCli.TradeAppPay(alipay.AppPayRequest{...})          // APP 支付
pre, err      := aliCli.TradePrecreate(ctx, alipay.PrecreateRequest{...}) // 扫码
q, err        := aliCli.TradeQuery(ctx, alipay.TradeQueryRequest{...})
rf, err       := aliCli.TradeRefund(ctx, alipay.RefundRequest{...})

// 异步通知
v, _   := alipay.ParseNotifyForm(r)
_      = aliCli.VerifyNotify(v)
alipay.WriteNotifyAck(w)
```

---

## Apple 内购（App Store Server API）

> Apple Pay 在此 SDK 中指 **App Store Server API**，用于服务端验证 iOS/macOS 内购交易、
> 管理订阅，不涉及前端支付界面（支付界面由 StoreKit 在客户端完成）。

### 配置

在 [App Store Connect](https://appstoreconnect.apple.com/) → 用户和访问 → 集成 → App Store Connect API
中创建 API 密钥，下载 `.p8` 私钥文件。

```go
appleCli, err := applepay.NewClient(applepay.Config{
    KeyID:     "XXXXXXXXXX",          // App Store Connect API Key ID
    IssuerID:  "xxxxxxxx-xxxx-...",   // Issuer ID（Team UUID）
    BundleID:  "com.example.myapp",
    PrivateKey: string(p8文件内容),   // -----BEGIN PRIVATE KEY----- 格式

    // 可选：提供 Apple 根证书以完整验证 JWS 证书链（推荐生产环境开启）
    // 从 https://www.apple.com/certificateauthority/ 下载 AppleRootCA-G3.cer
    // openssl x509 -inform DER -in AppleRootCA-G3.cer -out root.pem
    AppleRootCertPEM: string(rootPEM),

    IsSandbox: false,
})
```

### 验证内购交易

```go
// 1. iOS 客户端完成购买，将 JWSTransaction 字符串发给服务端
// 2. 服务端调用 GetTransactionInfo 验证
resp, err := appleCli.GetTransactionInfo(ctx, transactionID)
tx, err   := applepay.DecodeJWSTransaction(resp.SignedTransactionInfo)
// 或带证书链验证：
tx, err   := applepay.DecodeJWSTransactionVerified(resp.SignedTransactionInfo, rootCertPEM)

fmt.Println(tx.ProductID, tx.TransactionID, tx.PurchaseDate)
```

### 查询交易历史

```go
// 分页查询
hist, err := appleCli.GetTransactionHistory(ctx, transactionID, "")
for _, jws := range hist.SignedTransactions {
    tx, _ := applepay.DecodeJWSTransaction(jws)
    // 处理 tx...
}

// 一次性获取全部（自动翻页）
all, err := appleCli.GetAllTransactionHistory(ctx, transactionID)
```

### 查询订阅状态

```go
status, err := appleCli.GetAllSubscriptionStatuses(ctx, transactionID)
for _, group := range status.Data {
    for _, s := range group.LastTransactions {
        // s.Status: 1=活跃, 2=过期, 3=账单宽限期, 4=账单重试, 5=撤销, 6=暂停
        tx, _ := applepay.DecodeJWSTransaction(s.SignedTransactionInfo)
        ri, _ := applepay.DecodeJWSRenewalInfo(s.SignedRenewalInfo)
        _ = tx; _ = ri
    }
}
```

### 延长订阅续订日期

```go
// 补偿用户因服务中断损失的时间
result, err := appleCli.ExtendSubscriptionRenewalDate(ctx, originalTransactionID,
    applepay.ExtendRenewalDateRequest{
        ExtendByDays:      7,
        ExtendReasonCode:  3, // 3=服务问题
        RequestIdentifier: uuid.New().String(), // 幂等 ID
    },
)
```

### 退款历史

```go
hist, err := appleCli.GetRefundHistory(ctx, transactionID, "")
for _, jws := range hist.SignedTransactions {
    tx, _ := applepay.DecodeJWSTransaction(jws)
    // tx.RevocationReason: 1=用户主动退款
}
```

### 处理 App Store Server Notifications

```go
// Apple 以 POST 方式推送 JWS 通知到你的服务器
func handleAppleNotify(w http.ResponseWriter, r *http.Request) {
    var body struct {
        SignedPayload string `json:"signedPayload"`
    }
    json.NewDecoder(r.Body).Decode(&body)

    notification, err := applepay.ParseServerNotification(body.SignedPayload, rootCertPEM)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    // notification.NotificationType: SUBSCRIBED / DID_RENEW / EXPIRED / REFUND 等
    // notification.Data.SignedTransactionInfo 可进一步解码
    tx, _ := applepay.DecodeJWSTransaction(notification.Data.SignedTransactionInfo)
    _ = tx
    w.WriteHeader(http.StatusOK)
}
```

---

## 京东支付

### 配置

在京东商家平台申请商户号，获取 AppKey 及 RSA 密钥对（商户私钥用于签名，京东平台公钥用于验签）。

```go
jdCli, err := jdpay.NewClient(jdpay.Config{
    MerchantNo:     "商户编号",
    AppKey:         "appkey",
    PrivateKeyPEM:  string(商户私钥PEM),
    JDPublicKeyPEM: string(京东平台公钥PEM),
    IsSandbox:      false,
})
```

### 统一下单

```go
// H5 支付
resp, err := jdCli.UnifiedOrder(ctx, jdpay.UnifiedOrderRequest{
    TradeNum:    "商户唯一订单号",
    TradeName:   "商品名称",
    Amount:      100, // 分
    TradeType:   jdpay.TradeTypeH5,
    CallbackURL: "https://你的域名/notify/jdpay",
    UserIP:      "用户IP",
})
// resp.PayURL 引导用户跳转支付

// 扫码支付
resp, err := jdCli.UnifiedOrder(ctx, jdpay.UnifiedOrderRequest{
    TradeType: jdpay.TradeTypeQRCode,
    // ...
})
// resp.QRCode 生成二维码

// APP 支付
resp, err := jdCli.UnifiedOrder(ctx, jdpay.UnifiedOrderRequest{
    TradeType: jdpay.TradeTypeAPP,
    // ...
})
// resp.Token 传入京东钱包 SDK
```

### 查单 / 关单 / 退款

```go
// 查单
q, err := jdCli.QueryOrder(ctx, tradeNum)
// q.TradeState: WAIT_BUYER_PAY / TRADE_SUCCESS / TRADE_CLOSED / TRADE_REFUND

// 关单（未付款订单）
_ = jdCli.CloseOrder(ctx, tradeNum)

// 申请退款
rf, err := jdCli.ApplyRefund(ctx, jdpay.ApplyRefundRequest{
    TradeNum:     tradeNum,
    OutRefundNo:  "退款单号（唯一）",
    RefundAmount: 50, // 分
    RefundReason: "协商退款",
})

// 查询退款
rfq, err := jdCli.QueryRefund(ctx, outRefundNo)
// rfq.RefundState: REFUND_SUCCESS / REFUND_FAIL / REFUNDING
```

### 支付回调

```go
func handleJDPayNotify(w http.ResponseWriter, r *http.Request) {
    payload, err := jdCli.ParseAndVerifyNotify(r)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    if payload.TradeState != "TRADE_SUCCESS" {
        jdpay.WriteNotifyAck(w)
        return
    }
    // TODO: 用 payload.TradeNum 查找本地订单，核验金额，幂等更新状态
    jdpay.WriteNotifyAck(w)
}
```

---

## 可选注入 `*http.Client`

所有子包均支持在 Config 中注入自定义 `*http.Client`（如 `pkg/http_client`），
非 nil 时 `HTTPTimeout` 被忽略，以注入实例的 `Timeout`/`Transport` 为准。

```go
shared := http_client.NewClient(http_client.Config{Timeout: 30 * time.Second})
wxCli, _    := wechat.NewClient(wechat.Config{..., HTTPClient: shared.HTTPClient()})
appleCli, _ := applepay.NewClient(applepay.Config{..., HTTPClient: shared.HTTPClient()})
```

---

## 错误处理

| 包 | 哨兵错误 | 说明 |
|----|---------|------|
| `wechat` | `ErrAPI`、`ErrVerifySign`、`ErrInvalidConfig` | 业务错误类型为 `*wechat.APIError` |
| `alipay` | `ErrAPI`、`ErrVerifySign`、`ErrInvalidConfig` | |
| `applepay` | `ErrAPI`、`ErrVerifySign`、`ErrInvalidConfig` | 业务错误类型为 `*applepay.APIError` |
| `jdpay` | `ErrAPI`、`ErrVerifySign`、`ErrInvalidConfig` | 业务错误类型为 `*jdpay.APIError` |

```go
var apiErr *applepay.APIError
if errors.As(err, &apiErr) {
    // apiErr.ErrorCode, apiErr.HTTPStatus
}
if errors.Is(err, jdpay.ErrVerifySign) {
    // 签名验证失败
}
```

---

## 安全建议

- 回调 URL 使用 HTTPS，并校验订单金额、商户单号与用户身份一致性。
- 微信：务必 `ParseTransactionNotify`（验签 + 解密），勿仅解密不验签。
- 支付宝：务必 `VerifyNotify` 后再更新订单状态。
- Apple：生产环境务必配置 `AppleRootCertPEM` 开启证书链验证；重要交易用 `GetTransactionInfo` 服务端二次核验。
- 京东：`ParseAndVerifyNotify` 自动验签，验通后再执行业务逻辑。
- 所有密钥/证书建议来自配置中心或密钥管理服务，勿写入代码库。

---

## 代码路径

```
pkg/biz/pay/
├── pay.go            — Provider 常量
├── hub.go            — Hub 聚合入口
├── wechat/           — 微信支付 APIv3
├── alipay/           — 支付宝 RSA2
├── applepay/         — Apple App Store Server API
│   ├── config.go
│   ├── errors.go
│   ├── client.go     — JWT ES256 生成 + HTTP 请求
│   ├── jws.go        — JWS 解码 + ECDSA 证书链验签
│   ├── transaction.go — 交易查询与历史
│   ├── subscription.go — 订阅状态与续订日期延长
│   ├── refund.go     — 退款历史
│   └── notification.go — 服务端通知解析与测试
└── jdpay/            — 京东支付
    ├── config.go
    ├── errors.go
    ├── sign.go       — RSA-SHA256 签名/验签
    ├── client.go     — HTTP POST + 自动签名/验签
    ├── order.go      — 统一下单、关单
    ├── query.go      — 查单、查退款
    ├── refund.go     — 申请退款
    └── notify.go     — 回调解析与验签
```
