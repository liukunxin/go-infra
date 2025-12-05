package mysql

import (
	"log"
	"sync"
)

var (
	globalClient *Client
	once         sync.Once
)

// Init 启动时Mysql全局初始化
func Init(cfg Config) {
	once.Do(func() {
		c, err := NewClient(cfg)
		if err != nil {
			log.Fatal(err)
		}
		globalClient = c
	})
}

func GetClient() *Client {
	if globalClient == nil {
		log.Fatal("mysql client not initialized")
	}
	return globalClient
}
