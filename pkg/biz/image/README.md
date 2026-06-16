# pkg/biz/image

图片存储与防盗链访问封装，基于 S3 兼容对象存储实现，对业务层完全屏蔽底层存储细节。

---

## 核心设计

```
上传图片
  ↓
业务层（image.Service）
  ↓  生成 imageID + objectKey，存入对象存储（私有 ACL）
ObjStore（pkg/infra/objstore）
  ↓
返回 imageID 给业务

访问图片
  ↓
调用方：BuildAccessToken(imageID, viewerID)
  ↓  生成 HMAC-SHA256 签名 Token（含有效期 + 用户绑定）
业务 URL：https://api.example.com/image/v?t={token}
  ↓
HTTP Handler 调用：ResolveAccessToken(ctx, token, viewerID)
  ↓  验证签名 + 有效期 + 用户匹配
生成预签名 URL（短期，默认 5 分钟）
  ↓
302 Redirect → 对象存储临时 URL
```

**防盗链机制**：
- 图片以私有 ACL 存储，不可直接访问
- 业务 URL 中携带 HMAC-SHA256 签名 Token，服务端验证
- Token 支持绑定用户 ID（私有模式），防止 URL 分享后被他人使用
- Token 与预签名 URL 均有有效期，即使泄露影响范围有限
- 预签名 URL 仅在服务端内部生成，对外不可见

---

## 快速开始

### 1. 初始化

```go
import (
    "github.com/liukunxin/go-infra/pkg/infra/objstore"
    "github.com/liukunxin/go-infra/pkg/biz/image"
    "time"
)

// 初始化对象存储客户端（全局一次，支持 KS3/OSS/OBS/MinIO 等）
objstore.Init(&objstore.Config{
    Region:    "cn-beijing",
    Endpoint:  "ks3-cn-beijing.ksyuncs.com",
    AccessKey: "your-access-key",
    SecretKey: "your-secret-key",
    Bucket:    "my-image-bucket",
})

// 创建图片服务
cfg := image.Config{
    Bucket:          "my-image-bucket",
    SignSecret:      "your-32-byte-random-secret-string",
    DefaultTokenTTL: 30 * time.Minute,
    PresignTTL:      5 * time.Minute,
    // 可选：限制允许的图片类型
    AllowedMimeTypes: []string{"image/jpeg", "image/png", "image/webp"},
}

svc, err := image.NewService(cfg, myMetaStore) // myMetaStore 实现 image.MetaStorage
```

### 2. 上传图片

```go
func HandleUpload(w http.ResponseWriter, r *http.Request) {
    file, header, _ := r.FormFile("image")
    defer file.Close()

    img, err := svc.Upload(r.Context(), &image.UploadRequest{
        Body:     file,
        MimeType: header.Header.Get("Content-Type"),
        Size:     header.Size,
        OwnerID:  currentUserID,
        MaxBytes: 5 << 20, // 5 MiB 限制
    })
    if err != nil {
        // 处理 ErrInvalidMimeType / ErrFileTooLarge
    }
    // 将 img.ID 存入数据库，供后续引用
}
```

### 3. 生成访问 URL

```go
// 私有模式：Token 绑定用户，只有 viewerID 可访问
token, err := svc.BuildAccessToken(img.ID, viewerID)
accessURL := "https://api.example.com/image/v?t=" + token

// 公开模式：任何人持有 Token 均可访问（适用于公开图片）
token, err := svc.BuildAccessToken(img.ID, "")

// 自定义有效期
token, err := svc.BuildAccessToken(img.ID, viewerID, image.WithTokenTTL(1*time.Hour))
```

### 4. 处理查看请求

```go
func HandleView(w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("t")
    viewerID := getCurrentUserID(r) // 从 JWT/Session 中获取当前用户 ID

    presignedURL, err := svc.ResolveAccessToken(r.Context(), token, viewerID)
    switch {
    case errors.Is(err, image.ErrTokenExpired):
        http.Error(w, "link expired", http.StatusForbidden)
    case errors.Is(err, image.ErrTokenInvalid):
        http.Error(w, "invalid token", http.StatusForbidden)
    case errors.Is(err, image.ErrTokenUnauthorized):
        http.Error(w, "unauthorized", http.StatusForbidden)
    case err != nil:
        http.Error(w, "internal error", http.StatusInternalServerError)
    default:
        http.Redirect(w, r, presignedURL, http.StatusFound)
    }
}
```

### 5. 删除图片

```go
err := svc.Delete(ctx, imageID)
```

---

## 实现 MetaStorage 接口

```go
type ImageRecord struct {
    gorm.Model
    BizID     string `gorm:"uniqueIndex"`
    MimeType  string
    Size      int64
    OwnerID   string
}

type GormMetaStore struct{ db *gorm.DB }

func (g *GormMetaStore) Save(ctx context.Context, img *image.Image) error {
    return g.db.WithContext(ctx).Create(&ImageRecord{
        BizID:    img.ID,
        MimeType: img.MimeType,
        Size:     img.Size,
        OwnerID:  img.OwnerID,
    }).Error
}

func (g *GormMetaStore) Get(ctx context.Context, id string) (*image.Image, error) {
    var r ImageRecord
    if err := g.db.WithContext(ctx).Where("biz_id = ?", id).First(&r).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, image.ErrNotFound
        }
        return nil, err
    }
    return &image.Image{ID: r.BizID, MimeType: r.MimeType, Size: r.Size, OwnerID: r.OwnerID}, nil
}

func (g *GormMetaStore) Delete(ctx context.Context, id string) error {
    return g.db.WithContext(ctx).Where("biz_id = ?", id).Delete(&ImageRecord{}).Error
}
```

---

## 切换存储供应商

底层使用 `pkg/infra/objstore`（S3 兼容协议），切换供应商只需修改 objstore 初始化配置的 Endpoint：

```go
// 阿里 OSS
objstore.Init(&objstore.Config{
    Endpoint:  "https://oss-cn-hangzhou.aliyuncs.com",
    AccessKey: "...",
    SecretKey: "...",
    Bucket:    "my-bucket",
})

// 华为 OBS
objstore.Init(&objstore.Config{
    Endpoint:  "https://obs.cn-north-4.myhuaweicloud.com",
    AccessKey: "...",
    SecretKey: "...",
    Bucket:    "my-bucket",
})
```

如需在测试中注入自定义 client（如连接 MinIO），使用 `WithClient` 选项：

```go
client, _ := objstore.NewClient(&objstore.Config{Endpoint: "http://localhost:9000", ...})
svc, _ := image.NewService(cfg, nil, image.WithClient(client))
```

---

## 配置说明

| 字段 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `Bucket` | 是 | — | 对象存储 Bucket 名称 |
| `SignSecret` | 是 | — | HMAC 签名密钥，建议 32+ 字节随机串 |
| `KeyPrefix` | 否 | `"images"` | 对象 key 前缀，格式：`{prefix}/{imageID}` |
| `DefaultTokenTTL` | 否 | `30m` | 访问 Token 默认有效期 |
| `PresignTTL` | 否 | `5m` | 预签名 URL 有效期，建议短于 TokenTTL |
| `AllowedMimeTypes` | 否 | 所有 `image/*` | MIME 类型白名单 |
