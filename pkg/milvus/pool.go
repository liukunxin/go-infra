package milvus

import (
	"backend/go-infra/pkg/log"
	"context"
	"errors"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"sync"
	"time"
)

// Pool 定义连接池结构
type Pool struct {
	mu         sync.Mutex
	clients    chan client.Client            // 存储连接的通道
	maxSize    int                           // 最大连接数
	createFunc func() (client.Client, error) // 创建新连接的方法
}

// NewPool 创建一个新的连接池
func NewPool(maxSize int, createFunc func() (client.Client, error)) (*Pool, error) {
	if maxSize <= 0 {
		return nil, errors.New("maxSize must be greater than 0")
	}

	p := &Pool{
		clients:    make(chan client.Client, maxSize),
		maxSize:    maxSize,
		createFunc: createFunc,
	}

	// 预初始化连接池
	for i := 0; i < maxSize; i++ {
		milvusClient, err := createFunc()
		if err != nil {
			return nil, err
		}
		p.clients <- milvusClient
	}

	return p, nil
}

// Get 获取一个连接
func (p *Pool) Get() (client.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	select {
	case milvusClient := <-p.clients:
		// 从池中获取连接
		return milvusClient, nil
	case <-ctx.Done():
		// 获取超时
		return nil, ctx.Err()
	}
}

// Put 归还连接到池中
func (p *Pool) Put(milvusClient client.Client) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 如果池已满，关闭连接
	if len(p.clients) >= p.maxSize {
		err := milvusClient.Close()
		if err != nil {
			log.New().Error("[milvus_err]||err=%s" + err.Error())
		}
		return
	}

	// 放回池中
	select {
	case p.clients <- milvusClient:
	default:
		// 如果通道已满，关闭连接
		err := milvusClient.Close()
		if err != nil {
			log.New().Error("[milvus_err]||err=%s" + err.Error())
		}
	}
}

// Close 关闭连接池
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.clients)
	for milvusClient := range p.clients {
		err := milvusClient.Close()
		if err != nil {
			log.New().Error("[milvus_err]||err=%s" + err.Error())
		}
	}
}

func GetClient(ctx context.Context) (client.Client, error) {
	milvusClient, err := milvusPool.Get()
	if err != nil {
		log.WithContext(ctx).Error("[milvus_err]||Failed to get Milvus client", map[string]interface{}{
			"err": err.Error(),
		})
		return nil, err
	}
	return milvusClient, err
}

func ReturnClient(c client.Client) {
	defer milvusPool.Put(c)
}
