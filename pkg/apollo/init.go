package apollo

import (
	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
	"log"
	"os"
)

var (
	client agollo.Client
)

func Init() {
	apolloIP := os.Getenv("apollo_ip")
	if apolloIP == "" {
		log.Fatalf("初始化 Apollo 客户端失败，未找到apolloIP")
	}
	cluster := os.Getenv("apollo_cluster")
	if cluster == "" {
		cluster = "default"
	}
	namespace := os.Getenv("apollo_namespace")
	if namespace == "" {
		namespace = "application"
	}

	// 配置远程 Apollo 服务信息
	appConfig := &config.AppConfig{
		AppID:          "vektor",  // 替换为你的 AppID
		Cluster:        cluster,   // 替换为你的 Cluster 名称
		IP:             apolloIP,  // 远程 Apollo 地址
		NamespaceName:  namespace, // 替换为你的 Namespace
		IsBackupConfig: true,      // 是否启用本地备份
	}
	// 初始化 Apollo 客户端
	apolloClient, err := agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return appConfig, nil
	})
	if err != nil {
		log.Fatalf("初始化 Apollo 客户端失败: %v", err)
	}
	log.Println("成功连接到 Apollo 服务")
	client = apolloClient
}

func GetClient() agollo.Client {
	return client
}
