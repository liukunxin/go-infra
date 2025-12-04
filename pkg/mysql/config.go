package mysql

import (
	"gorm.io/gorm/logger"
	"time"
)

// Config SDK 配置
type Config struct {
	DSN               string          `yaml:"dsn"`                                            // 完整 DSN，如: user:pass@tcp(host:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	MaxOpenConns      int             `yaml:"max_open_conns"`                                 // 最大打开连接数
	MaxIdleConns      int             `yaml:"max_idle_conns"`                                 // 最大空闲连接数
	ConnMaxLifetime   time.Duration   `yaml:"conn_max_lifetime"`                              // 连接最大存活时长
	ConnMaxIdleTime   time.Duration   `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`   // 连接最大空闲时长 (Go 1.15+)
	EnablePrepareStmt bool            `json:"enable_prepare_stmt" yaml:"enable_prepare_stmt"` // 是否启用 prepared statement cache (gorm.Config)。默认 true 推荐用于读多写少场景
	SkipDefaultTx     bool            `json:"skip_default_tx" yaml:"skip_default_tx"`         // 是否跳过默认事务（可提高写入性能，但风险更高）
	ConnRetryTimes    int             `json:"conn_retry_times" yaml:"conn_retry_times"`       // 连接初始化重试次数
	ConnRetryInterval time.Duration   `json:"conn_retry_interval" yaml:"conn_retry_interval"` // 重试间隔
	GormLogLevel      logger.LogLevel `json:"gorm_log_level" yaml:"gorm_log_level"`
}
