# Milvus - 向量数据库客户端

Milvus 向量数据库客户端封装，提供连接池管理，优化高并发场景。

## 📖 功能特性

- ✅ 自定义连接池实现
- ✅ 并发安全
- ✅ 连接复用
- ✅ 超时控制（优化为5秒）
- ✅ 自动初始化
- ✅ 简单易用

## 🚀 快速开始

### 基础用法

```go
package main

import (
    "context"
    "log"
    "github.com/liukunxin/go-infra/pkg/milvus"
    "github.com/milvus-io/milvus-sdk-go/v2/entity"
)

func main() {
    // 1. 初始化 Milvus
    milvus.Init(&milvus.Config{
        Address:  "localhost:19530",
        Username: "root",
        Password: "Milvus",
        PoolSize: 10,  // 连接池大小
    })

    // 2. 获取客户端
    ctx := context.Background()
    client, err := milvus.GetClient(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer milvus.ReturnClient(client)

    // 3. 使用客户端
    collections, err := client.ListCollections(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, col := range collections {
        log.Printf("Collection: %s\n", col.Name)
    }
}
```

## 📋 配置选项

### Config 结构

```go
type Config struct {
    Address  string  // Milvus 服务地址
    Username string  // 用户名
    Password string  // 密码
    PoolSize int     // 连接池大小
}
```

### 推荐配置

```go
// 开发环境
&milvus.Config{
    Address:  "localhost:19530",
    Username: "root",
    Password: "Milvus",
    PoolSize: 5,
}

// 生产环境（低并发）
&milvus.Config{
    Address:  "milvus.prod.com:19530",
    Username: "app_user",
    Password: "strong_password",
    PoolSize: 10,
}

// 生产环境（高并发）
&milvus.Config{
    Address:  "milvus.prod.com:19530",
    Username: "app_user",
    Password: "strong_password",
    PoolSize: 50,  // 根据并发量调整
}
```

## 💡 使用示例

### 示例1：创建 Collection

```go
func createCollection(ctx context.Context) error {
    client, err := milvus.GetClient(ctx)
    if err != nil {
        return err
    }
    defer milvus.ReturnClient(client)

    // 定义 Schema
    schema := &entity.Schema{
        CollectionName: "article_vectors",
        Description:    "文章向量索引",
        Fields: []*entity.Field{
            {
                Name:       "id",
                DataType:   entity.FieldTypeInt64,
                PrimaryKey: true,
                AutoID:     true,
            },
            {
                Name:     "article_id",
                DataType: entity.FieldTypeInt64,
            },
            {
                Name:     "embedding",
                DataType: entity.FieldTypeFloatVector,
                TypeParams: map[string]string{
                    "dim": "768",  // 向量维度
                },
            },
        },
    }

    // 创建 Collection
    return client.CreateCollection(ctx, schema, 2)  // 2个分片
}
```

### 示例2：插入向量

```go
func insertVectors(ctx context.Context, articles []Article) error {
    client, err := milvus.GetClient(ctx)
    if err != nil {
        return err
    }
    defer milvus.ReturnClient(client)

    // 准备数据
    articleIDs := make([]int64, len(articles))
    embeddings := make([][]float32, len(articles))
    
    for i, article := range articles {
        articleIDs[i] = article.ID
        embeddings[i] = article.Embedding  // 768维向量
    }

    // 插入数据
    _, err = client.Insert(ctx, 
        "article_vectors",
        "",  // partition name
        entity.NewColumnInt64("article_id", articleIDs),
        entity.NewColumnFloatVector("embedding", 768, embeddings),
    )
    
    return err
}
```

### 示例3：向量搜索

```go
func searchSimilarArticles(ctx context.Context, queryVector []float32, topK int) ([]int64, error) {
    client, err := milvus.GetClient(ctx)
    if err != nil {
        return nil, err
    }
    defer milvus.ReturnClient(client)

    // 搜索参数
    sp, _ := entity.NewIndexFlatSearchParam()
    
    // 执行搜索
    results, err := client.Search(
        ctx,
        "article_vectors",           // collection name
        []string{},                  // partition names
        "",                          // expression
        []string{"article_id"},      // output fields
        []entity.Vector{             // query vectors
            entity.FloatVector(queryVector),
        },
        "embedding",                 // vector field name
        entity.L2,                   // metric type
        topK,                        // top K
        sp,                          // search params
    )
    
    if err != nil {
        return nil, err
    }

    // 提取结果
    var articleIDs []int64
    for _, result := range results {
        for _, field := range result.Fields {
            if field.Name() == "article_id" {
                ids := field.(*entity.ColumnInt64).Data()
                articleIDs = append(articleIDs, ids...)
            }
        }
    }
    
    return articleIDs, nil
}
```

### 示例4：创建索引

```go
func createIndex(ctx context.Context) error {
    client, err := milvus.GetClient(ctx)
    if err != nil {
        return err
    }
    defer milvus.ReturnClient(client)

    // IVF_FLAT 索引
    idx, err := entity.NewIndexIvfFlat(entity.L2, 128)
    if err != nil {
        return err
    }

    // 创建索引
    return client.CreateIndex(
        ctx,
        "article_vectors",  // collection name
        "embedding",        // field name
        idx,               // index
        false,             // async
    )
}
```

### 示例5：查询数据

```go
func queryArticles(ctx context.Context, articleIDs []int64) ([]Article, error) {
    client, err := milvus.GetClient(ctx)
    if err != nil {
        return nil, err
    }
    defer milvus.ReturnClient(client)

    // 构建查询表达式
    expr := fmt.Sprintf("article_id in [%s]", strings.Join(idsToStrings(articleIDs), ","))

    // 执行查询
    results, err := client.Query(
        ctx,
        "article_vectors",
        []string{},  // partition names
        expr,
        []string{"article_id", "embedding"},
    )
    
    if err != nil {
        return nil, err
    }

    // 解析结果
    var articles []Article
    for _, field := range results {
        // 处理返回的字段...
    }
    
    return articles, nil
}
```

### 示例6：删除数据

```go
func deleteArticles(ctx context.Context, articleIDs []int64) error {
    client, err := milvus.GetClient(ctx)
    if err != nil {
        return err
    }
    defer milvus.ReturnClient(client)

    // 构建删除表达式
    expr := fmt.Sprintf("article_id in [%s]", strings.Join(idsToStrings(articleIDs), ","))

    // 执行删除
    return client.Delete(
        ctx,
        "article_vectors",
        "",    // partition name
        expr,
    )
}
```

### 示例7：完整的向量搜索服务

```go
type VectorService struct {
    // 其他依赖...
}

// 添加文章向量
func (s *VectorService) AddArticle(ctx context.Context, article *Article) error {
    // 1. 生成向量（假设已有embedding服务）
    embedding, err := s.embeddingService.GetEmbedding(article.Content)
    if err != nil {
        return err
    }

    // 2. 获取 Milvus 客户端
    client, err := milvus.GetClient(ctx)
    if err != nil {
        return err
    }
    defer milvus.ReturnClient(client)

    // 3. 插入向量
    _, err = client.Insert(ctx,
        "article_vectors",
        "",
        entity.NewColumnInt64("article_id", []int64{article.ID}),
        entity.NewColumnFloatVector("embedding", 768, [][]float32{embedding}),
    )

    return err
}

// 搜索相似文章
func (s *VectorService) SearchSimilar(ctx context.Context, queryText string, topK int) ([]*Article, error) {
    // 1. 生成查询向量
    queryVector, err := s.embeddingService.GetEmbedding(queryText)
    if err != nil {
        return nil, err
    }

    // 2. 向量搜索
    articleIDs, err := searchSimilarArticles(ctx, queryVector, topK)
    if err != nil {
        return nil, err
    }

    // 3. 从数据库获取文章详情
    articles, err := s.articleRepo.GetByIDs(ctx, articleIDs)
    return articles, err
}
```

## 🎯 最佳实践

### 1. 及时归还连接

```go
// ✅ 好的做法 - 使用 defer 确保归还
func doSomething(ctx context.Context) error {
    client, err := milvus.GetClient(ctx)
    if err != nil {
        return err
    }
    defer milvus.ReturnClient(client)  // 确保归还
    
    // 使用 client...
    return nil
}

// ❌ 不好的做法 - 忘记归还
func doSomething(ctx context.Context) error {
    client, _ := milvus.GetClient(ctx)
    // 使用 client...
    // 忘记归还，导致连接泄漏！
    return nil
}
```

### 2. 合理设置连接池大小

```go
// 根据并发量设置
并发量 10-50   → PoolSize: 10
并发量 50-200  → PoolSize: 20
并发量 200-500 → PoolSize: 50
并发量 500+    → PoolSize: 100
```

### 3. 批量操作

```go
// ✅ 好的做法 - 批量插入
func batchInsert(ctx context.Context, articles []Article) error {
    // 一次插入多条
    return insertVectors(ctx, articles)
}

// ❌ 不好的做法 - 逐条插入
func insertOneByOne(ctx context.Context, articles []Article) error {
    for _, article := range articles {
        insertVectors(ctx, []Article{article})  // 多次网络请求！
    }
    return nil
}
```

### 4. 错误处理

```go
func safeOperation(ctx context.Context) error {
    client, err := milvus.GetClient(ctx)
    if err != nil {
        if err == context.DeadlineExceeded {
            log.Error("获取连接超时")
            return err
        }
        log.Errorf("获取连接失败: %v", err)
        return err
    }
    defer milvus.ReturnClient(client)
    
    // 使用 client...
    return nil
}
```

### 5. 向量维度统一

```go
// ✅ 定义常量统一管理
const (
    EmbeddingDim = 768  // BERT/RoBERTa
    // EmbeddingDim = 1536  // OpenAI Ada-002
)

// 创建 Collection 时使用
TypeParams: map[string]string{
    "dim": fmt.Sprintf("%d", EmbeddingDim),
}
```

## 📊 性能优化

### 索引选择

```go
// 小数据集（< 100万）- FLAT
idx, _ := entity.NewIndexFlat(entity.L2)

// 中等数据集（100万 - 1000万）- IVF_FLAT
idx, _ := entity.NewIndexIvfFlat(entity.L2, 2048)

// 大数据集（> 1000万）- IVF_PQ
idx, _ := entity.NewIndexIvfPQ(entity.L2, 2048, 16, 8)

// 高精度场景 - HNSW
idx, _ := entity.NewIndexHNSW(entity.L2, 16, 200)
```

### 批量操作

```go
// 批量插入，每批1000-10000条
const batchSize = 5000

for i := 0; i < len(articles); i += batchSize {
    end := i + batchSize
    if end > len(articles) {
        end = len(articles)
    }
    insertVectors(ctx, articles[i:end])
}
```

### 搜索优化

```go
// 1. 设置合理的 nprobe（IVF 索引）
sp, _ := entity.NewIndexIvfFlatSearchParam(64)  // nprobe=64

// 2. 限制返回字段
[]string{"article_id"}  // 只返回ID

// 3. 使用分区
client.Search(ctx, collectionName, []string{"partition_2024"}, ...)
```

## ⚠️ 注意事项

1. **连接必须归还** - 使用 defer 确保归还
2. **连接池大小要合理** - 避免过大或过小
3. **向量维度要一致** - Collection 定义和数据要匹配
4. **超时已优化为5秒** - 足够大多数场景使用
5. **并发安全** - 连接池已做并发保护

## 🔗 相关模块

- [Log](log.md) - 操作日志记录
- [Trace](trace.md) - 操作追踪
- [Metrics](metrics.md) - 性能监控

## 📖 推荐阅读

- [Milvus 官方文档](https://milvus.io/docs/)
- [Milvus SDK Go](https://github.com/milvus-io/milvus-sdk-go)
- [向量索引选择指南](https://milvus.io/docs/index.md)
