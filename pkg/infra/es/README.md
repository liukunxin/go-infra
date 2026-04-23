# Elasticsearch 客户端

基于 [go-elasticsearch/v8](https://github.com/elastic/go-elasticsearch) 的 ES 封装，提供文档 CRUD、索引管理、DSL 查询和批量写入，内置重试与连接池。

---

## 快速上手

### 1. 初始化（程序启动时调用一次）

```go
import "github.com/yourorg/go-infra/pkg/infra/es"

func main() {
    cfg := es.Config{
        Addresses:    []string{"http://127.0.0.1:9200"},
        Username:     "elastic",   // 可留空
        Password:     "password",  // 可留空
        MaxRetries:   3,
        RetryBackoff: 100 * time.Millisecond,
    }

    if err := es.Init(cfg); err != nil {
        log.Fatalf("es init: %v", err)
    }
}
```

**启用 HTTPS + 自定义 CA 证书**

```go
caCert, _ := os.ReadFile("/path/to/ca.crt")
cfg := es.Config{
    Addresses: []string{"https://es-host:9200"},
    EnableTLS: true,
    CACert:    caCert,  // PEM 格式
}
```

### 2. 获取客户端

```go
client := es.GetClient()   // 未初始化时 panic
```

---

## 索引管理

### 创建索引（指定 mapping）

```go
err := es.GetClient().CreateIndex(ctx, "articles", map[string]interface{}{
    "mappings": map[string]interface{}{
        "properties": map[string]interface{}{
            "title":      map[string]interface{}{"type": "text"},
            "created_at": map[string]interface{}{"type": "date"},
        },
    },
})
```

### 判断索引是否存在

```go
exists, err := es.GetClient().IndexExists(ctx, "articles")
```

### 更新 Mapping（新增字段）

```go
err := es.GetClient().PutMapping(ctx, "articles", map[string]interface{}{
    "properties": map[string]interface{}{
        "author": map[string]interface{}{"type": "keyword"},
    },
})
```

### 删除索引

```go
err := es.GetClient().DeleteIndex(ctx, "articles")
```

---

## 文档 CRUD

### 写入文档

```go
type Article struct {
    Title   string `json:"title"`
    Content string `json:"content"`
}

doc := Article{Title: "Go语言实战", Content: "..."}

// id 为空时 ES 自动生成
err := es.GetClient().IndexDoc(ctx, "articles", "doc-001", doc)
```

### 读取文档

```go
var article Article
err := es.GetClient().GetDoc(ctx, "articles", "doc-001", &article)
if errors.Is(err, es.ErrNotFound) {
    // 文档不存在
}
```

### 局部更新文档（只更新指定字段）

```go
err := es.GetClient().UpdateDoc(ctx, "articles", "doc-001",
    map[string]interface{}{"title": "新标题"})
```

### 删除文档

```go
err := es.GetClient().DeleteDoc(ctx, "articles", "doc-001")
```

---

## 搜索（DSL 查询）

```go
query := map[string]interface{}{
    "query": map[string]interface{}{
        "match": map[string]interface{}{
            "title": "Go语言",
        },
    },
    "sort": []interface{}{
        map[string]interface{}{"created_at": map[string]interface{}{"order": "desc"}},
    },
    "size": 10,
    "from": 0,
}

result, err := es.GetClient().Search(ctx, []string{"articles"}, query)
if err != nil {
    return err
}

fmt.Println("总命中:", result.Total)
for _, hit := range result.Hits {
    var article Article
    _ = json.Unmarshal(hit.Source, &article)
    fmt.Printf("[%s] %s\n", hit.ID, article.Title)
}
```

---

## 批量写入（Bulk）

比循环单条写入快 **5-10 倍**，适合导入大量数据。

```go
actions := []es.BulkAction{
    {Action: es.BulkIndex, Index: "articles", ID: "1", Doc: Article{Title: "文章1"}},
    {Action: es.BulkIndex, Index: "articles", ID: "2", Doc: Article{Title: "文章2"}},
    {Action: es.BulkUpdate, Index: "articles", ID: "1", Doc: map[string]interface{}{"title": "新标题"}},
    {Action: es.BulkDelete, Index: "articles", ID: "3"},
}

result, err := es.GetClient().Bulk(ctx, actions)
if err != nil {
    return err
}
if result.Errors {
    // 部分文档失败，遍历 result.Items 查看详情
    for _, item := range result.Items {
        fmt.Println(string(item))
    }
}
```

---

## 错误处理

```go
_, err := es.GetClient().GetDoc(ctx, "articles", "no-such-id", &doc)
switch {
case errors.Is(err, es.ErrNotFound):
    // 文档不存在，正常情况

case err != nil:
    var respErr *es.ResponseError
    if errors.As(err, &respErr) {
        fmt.Printf("ES 返回 HTTP %d: %s\n", respErr.StatusCode, respErr.RawBody)
    }
}
```

---

## 使用原始客户端

封装方法不满足需求时，可获取底层 `*elasticsearch.Client`：

```go
raw := es.GetClient().ESClient()
res, err := raw.Indices.Refresh(raw.Indices.Refresh.WithIndex("articles"))
```

---

## 配置说明

| 字段            | 说明                                  | 默认值  |
|-----------------|--------------------------------------|---------|
| `Addresses`     | ES 节点地址列表（必填）               | —       |
| `Username`      | HTTP Basic Auth 用户名               | ""      |
| `Password`      | HTTP Basic Auth 密码                 | ""      |
| `APIKey`        | API Key 认证（优先于用户名密码）      | ""      |
| `MaxRetries`    | 失败重试次数                          | 3       |
| `RetryBackoff`  | 首次重试等待时间（指数增长）          | 100ms   |
| `EnableTLS`     | 是否启用 TLS                          | false   |
| `CACert`        | PEM 格式 CA 证书（TLS 时可选）        | nil     |
