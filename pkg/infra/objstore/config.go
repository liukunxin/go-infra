package objstore

import "time"

// Config holds S3-compatible object storage connection parameters.
// Works with any S3-compatible provider (AWS S3, Aliyun OSS, Huawei OBS, KS3, MinIO, etc.)
// by setting the appropriate Endpoint.
type Config struct {
	Endpoint     string `yaml:"endpoint" json:"endpoint"`
	Region       string `yaml:"region" json:"region"`
	AccessKey    string `yaml:"access_key" json:"access_key"`
	SecretKey    string `yaml:"secret_key" json:"secret_key"`
	Bucket       string `yaml:"bucket" json:"bucket"`
	UsePathStyle bool   `yaml:"use_path_style" json:"use_path_style"`
}

// PutOptions carries optional parameters for PutObject.
type PutOptions struct {
	ContentType   string
	ContentLength int64
	Expires       time.Time
	Metadata      map[string]string
}
