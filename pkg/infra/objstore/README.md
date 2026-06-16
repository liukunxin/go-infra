# ObjStore - 统一对象存储

基于 [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) 的 S3 兼容对象存储封装，一套代码支持所有 S3 兼容厂商。

## 支持的厂商

| 厂商 | Endpoint 示例 | UsePathStyle |
|------|---------------|--------------|
| 金山云 KS3 | `ks3-cn-beijing.ksyuncs.com` | false |
| 阿里云 OSS | `oss-cn-hangzhou.aliyuncs.com` | false |
| 华为云 OBS | `obs.cn-north-4.myhuaweicloud.com` | false |
| 腾讯云 COS | `cos.ap-guangzhou.myqcloud.com` | false |
| AWS S3 | `s3.amazonaws.com` | false |
| MinIO | `minio.example.com:9000` | true |

## 快速上手

### 配置

```yaml
objstore:
  endpoint: "ks3-cn-beijing.ksyuncs.com"
  region: "cn-beijing"
  access_key: "your-ak"
  secret_key: "your-sk"
  bucket: "my-bucket"
  use_path_style: false
```

### 初始化

```go
import "github.com/liukunxin/go-infra/pkg/infra/objstore"

if err := objstore.Init(&cfg.ObjStore); err != nil {
    log.Fatalf("objstore init: %v", err)
}
```

### 使用

```go
client := objstore.GetClient()

// 上传
err := client.PutObject(ctx, "", "path/to/file.pdf", reader, objstore.PutOptions{
    ContentType: "application/pdf",
})

// 下载
body, err := client.GetObject(ctx, "", "path/to/file.pdf")
defer body.Close()

// 删除
err := client.DeleteObject(ctx, "", "path/to/file.pdf")

// 生成预签名下载 URL（10分钟有效）
url, err := client.PresignGetURL(ctx, "", "path/to/file.pdf", 10*time.Minute)

// 生成预签名上传 URL（1小时有效）
url, err := client.PresignPutURL(ctx, "", "path/to/file.pdf", time.Hour)
```

> bucket 参数传空字符串时使用 Config 中配置的默认 bucket。

### 多实例

```go
// 需要连接多个存储时，直接创建独立 Client
client, err := objstore.NewClient(&objstore.Config{
    Endpoint:  "obs.cn-north-4.myhuaweicloud.com",
    Region:    "cn-north-4",
    AccessKey: "...",
    SecretKey: "...",
    Bucket:    "another-bucket",
})
```

## 从 ks3 包迁移

| 旧（ks3） | 新（objstore） |
|-----------|----------------|
| `ks3.Init(cfg)` | `objstore.Init(cfg)` |
| `ks3.GetObject(ctx, bucket, key)` | `client.GetObject(ctx, bucket, key)` |
| `ks3.PutObject(ctx, bucket, key, body, opts)` | `client.PutObject(ctx, bucket, key, body, opts)` |
| `ks3.DeleteObject(ctx, bucket, key)` | `client.DeleteObject(ctx, bucket, key)` |
| `ks3.PresignGetURL(ctx, bucket, key, ttl)` | `client.PresignGetURL(ctx, bucket, key, ttl)` |
| `ks3.PresignPutURL(ctx, bucket, key, ttl)` | `client.PresignPutURL(ctx, bucket, key, ttl)` |

主要变化：操作方法从包级函数改为 Client 方法，支持多实例场景。
