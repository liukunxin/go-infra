package http_client

import (
	"log"
	"sync"
)

// 对外暴露
// 初始化后，直接GetClient即可
var (
	globalHttpClient *Client
	once             sync.Once
)

func Init(cfg Config) {
	once.Do(func() {
		globalHttpClient = NewClient(cfg)
	})
}

func GetHttpClient() *Client {
	if globalHttpClient == nil {
		log.Fatal("http client not initialized")
	}
	return globalHttpClient
}
