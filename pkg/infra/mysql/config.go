package mysql

import (
	"time"

	"gorm.io/gorm/logger"
)

// Config holds MySQL connection and pool parameters.
// All fields have sensible defaults applied by NewClient.
type Config struct {
	DSN               string          `yaml:"dsn"                  json:"dsn"`                   // full DSN: user:pass@tcp(host:3306)/db?charset=utf8mb4&parseTime=True&loc=Local
	MaxOpenConns      int             `yaml:"max_open_conns"       json:"max_open_conns"`        // default 100
	MaxIdleConns      int             `yaml:"max_idle_conns"       json:"max_idle_conns"`        // default 10
	ConnMaxLifetime   time.Duration   `yaml:"conn_max_lifetime"    json:"conn_max_lifetime"`     // default 1h
	ConnMaxIdleTime   time.Duration   `yaml:"conn_max_idle_time"   json:"conn_max_idle_time"`    // default 10m
	EnablePrepareStmt bool            `yaml:"enable_prepare_stmt"  json:"enable_prepare_stmt"`  // prepared statement cache; recommended for read-heavy workloads
	SkipDefaultTx     bool            `yaml:"skip_default_tx"      json:"skip_default_tx"`      // skip implicit transactions on writes; improves throughput but reduces safety
	ConnRetryTimes    int             `yaml:"conn_retry_times"     json:"conn_retry_times"`     // default 3
	ConnRetryInterval time.Duration   `yaml:"conn_retry_interval"  json:"conn_retry_interval"`  // default 2s
	GormLogLevel      logger.LogLevel `yaml:"gorm_log_level"       json:"gorm_log_level"`
}
