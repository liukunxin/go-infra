# ObjStore - 统一对象存储

基于 [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) 的 S3 兼容对象存储封装。

## API 一览

| 场景 | 方法 |
|------|------|
| 服务端上传 | `PutObject(ctx, bucket, key, body, PutOptions)` |
| 服务端下载 | `GetObject` |
| 服务端元信息 | `HeadObject` |
| 服务端删除 | `DeleteObject` |
| 客户端上传 | `PresignPut` → `PresignedRequest` |
| 客户端下载（私有） | `PresignGet` → `PresignedRequest` |
| 公有直链 | `ObjectURL`（对象须 `public-read`） |

## PresignedRequest

```go
type PresignedRequest struct {
    URL             string
    Method          string            // GET 或 PUT
    RequiredHeaders map[string]string // 客户端必须原样携带
}
```

## 预签名上传（含 ACL）

```go
req, err := client.PresignPut(ctx, "", key, time.Hour, objstore.PresignPutOptions{
    ACL: "public-read",
})
// PUT req.URL + req.RequiredHeaders（含 x-amz-acl）
```

## 预签名下载（私有对象）

```go
req, err := client.PresignGet(ctx, "", key, 10*time.Minute)
// GET req.URL
```

## 公有直链

```go
url := client.ObjectURL("", key)
```

## 配置

```go
cfg := &objstore.Config{Endpoint: "https://ks3-cn-beijing.ksyun.com", ...}
cfg.Normalize()
client, err := objstore.NewClient(cfg)
```

## ACL 常量

`ObjectACLPublicRead`、`ObjectACLPrivate`、`ObjectACLBucketOwnerFullControl` 等。

`NormalizeObjectACL` / `IsPublicObjectACL` 用于业务层判断。

## 错误

`errors.Is(err, objstore.ErrInvalidArgument)`
