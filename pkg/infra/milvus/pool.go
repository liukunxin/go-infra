package milvus

import (
	"context"
	"errors"
	"time"

	"github.com/liukunxin/go-infra/pkg/base/log"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
)

// Pool 定义连接池结构
type Pool struct {
	clients    chan client.Client
	maxSize    int
	createFunc func() (client.Client, error)
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

	for i := 0; i < maxSize; i++ {
		milvusClient, err := createFunc()
		if err != nil {
			return nil, err
		}
		p.clients <- milvusClient
	}

	return p, nil
}

// Get 获取一个连接。
func (p *Pool) Get() (client.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case c := <-p.clients:
		return c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Put 归还健康连接到池中。
func (p *Pool) Put(milvusClient client.Client) {
	select {
	case p.clients <- milvusClient:
	default:
		_ = milvusClient.Close()
	}
}

// Discard 丢弃一个不可用的连接，关闭它并创建新连接补充到池中。
// 当调用方发现连接出错时应调用此方法代替 Put。
func (p *Pool) Discard(broken client.Client) {
	_ = broken.Close()
	newC, err := p.createFunc()
	if err != nil {
		log.New().Warnf("[milvus] discard: failed to recreate connection: %v", err)
		return
	}
	p.Put(newC)
}

// Close 关闭连接池
func (p *Pool) Close() {
	close(p.clients)
	for milvusClient := range p.clients {
		err := milvusClient.Close()
		if err != nil {
			log.New().Errorf("[milvus_err]||err=%s", err.Error())
		}
	}
}

func GetClient(ctx context.Context) (client.Client, error) {
	milvusClient, err := milvusPool.Get()
	if err != nil {
		log.WithContext(ctx).WithFields(map[string]interface{}{
			"err": err.Error(),
		}).Error("[milvus_err]||Failed to get Milvus client")
		return nil, err
	}
	return milvusClient, nil
}

func ReturnClient(c client.Client) {
	milvusPool.Put(c)
}

// DiscardClient 丢弃一个不可用的连接并自动补充新连接到池中。
// 当调用方发现 Milvus 操作出错（如连接断开）时调用此方法代替 ReturnClient。
func DiscardClient(c client.Client) {
	milvusPool.Discard(c)
}
