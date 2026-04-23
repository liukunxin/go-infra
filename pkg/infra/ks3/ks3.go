package ks3

import (
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/aws/credentials"
	"github.com/ks3sdklib/aws-sdk-go/service/s3"
	"sync"
)

var (
	client *s3.S3
	once   sync.Once
)

func Init(config *Config) {
	once.Do(func() {
		cre := credentials.NewStaticCredentials(config.Ak, config.Sk, "")
		// 创建S3Client，更多配置项请查看Go-SDK初始化文档
		ks3client := s3.New(&aws.Config{
			Credentials: cre,             // 访问凭证
			Region:      config.Region,   // 填写您的Region
			Endpoint:    config.Endpoint, // 填写您的Endpoint
		})
		client = ks3client
	})
}

// GetClient 获取全局 KS3 客户端
func GetClient() *s3.S3 {
	return client
}
