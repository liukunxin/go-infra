package v8

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

// ClusterClient 集群模式客户端
var clusterClient *redis.ClusterClient

// SingleClient 单节点模式客户端
var singleClient *redis.Client

// InitRedisCluster 初始化 Redis 集群客户端
func InitRedisCluster(redisConf *Config) {
	ctx := context.Background()
	switch redisConf.Mode {
	case "single":
		singleClient = redis.NewClient(&redis.Options{
			Addr:         redisConf.Addresses[0],                             // 集群地址列表
			Password:     redisConf.Password,                                 // 密码
			PoolSize:     redisConf.PoolSize,                                 // 连接池大小
			MinIdleConns: redisConf.MinIdleConns,                             // 最小空闲连接数
			IdleTimeout:  time.Duration(redisConf.IdleTimeout) * time.Second, // 空闲连接超时
		})
		// 测试连接是否成功
		_, err := singleClient.Ping(ctx).Result()
		if err != nil {
			log.Fatalf("Unable to connect to Redis Single Cluster: %v", err)
		}
	case "cluster":
		clusterClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        redisConf.Addresses,                                // 集群地址列表
			Password:     redisConf.Password,                                 // 密码
			PoolSize:     redisConf.PoolSize,                                 // 连接池大小
			MinIdleConns: redisConf.MinIdleConns,                             // 最小空闲连接数
			IdleTimeout:  time.Duration(redisConf.IdleTimeout) * time.Second, // 空闲连接超时
		})
		// 测试连接是否成功
		_, err := clusterClient.Ping(ctx).Result()
		if err != nil {
			log.Fatalf("Unable to connect to Redis Cluster: %v", err)
		}
	default:
		log.Fatalf("Unable find redis deploy mode")
	}
	log.Println("Connected to Redis Cluster successfully!")
}

func GetRedisClient() redis.UniversalClient {
	if clusterClient != nil {
		return clusterClient
	}
	if singleClient != nil {
		return singleClient
	}
	return nil
}
