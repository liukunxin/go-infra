package milvus

import (
	"context"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"log"
)

var milvusPool *Pool

func InitializePool(config *Config) {
	var err error

	milvusPool, err = NewPool(config.PoolSize, func() (client.Client, error) {
		// 配置 Milvus 客户端
		clientConfig := client.Config{
			Address:  config.Address,
			Username: config.Username,
			Password: config.Password,
		}

		// 创建 Milvus 客户端
		milvusClient, err := client.NewClient(context.Background(), clientConfig)
		if err != nil {
			log.Printf("Failed to create Milvus client: %v", err)
			return nil, err
		}

		return milvusClient, nil
	})

	if err != nil {
		log.Fatalf("Failed to initialize Milvus connection pool: %v", err)
	}

	log.Println("成功初始化 Milvus 客户端连接池")
}
