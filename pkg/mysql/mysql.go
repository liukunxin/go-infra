// gorm_mysql_sdk.go
// 简洁高性能的 GORM MySQL SDK
// 特性：
// - 支持连接池参数（MaxOpenConns、MaxIdleConns、ConnMaxLifetime、ConnMaxIdleTime）
// - 启用 prepared statements 缓存、禁用默认事务以提升性能（可配置）
// - 带重试的连接初始化和健康检查
// - 事务封装（WithTransaction）和常用包装方法
// - 支持自定义 logger 和钩子

package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config SDK 配置
type Config struct {
	DSN               string        // 完整 DSN，如: user:pass@tcp(host:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	MaxOpenConns      int           // 最大打开连接数
	MaxIdleConns      int           // 最大空闲连接数
	ConnMaxLifetime   time.Duration // 连接最大存活时长
	ConnMaxIdleTime   time.Duration // 连接最大空闲时长 (Go 1.15+)
	EnablePrepareStmt bool          // 是否启用 prepared statement cache (gorm.Config)。默认 true 推荐用于读多写少场景
	SkipDefaultTx     bool          // 是否跳过默认事务（可提高写入性能，但风险更高）
	ConnRetryTimes    int           // 连接初始化重试次数
	ConnRetryInterval time.Duration // 重试间隔
	GormLogLevel      logger.LogLevel
}

// Client SDK 客户端
type Client struct {
	cfg   Config
	DB    *gorm.DB
	sqlDB *sql.DB
}

// NewClient 创建并初始化 SDK
func NewClient(cfg Config) (*Client, error) {
	if cfg.DSN == "" {
		return nil, errors.New("DSN 不能为空")
	}
	// 默认值
	if cfg.ConnRetryTimes <= 0 {
		cfg.ConnRetryTimes = 3
	}
	if cfg.ConnRetryInterval <= 0 {
		cfg.ConnRetryInterval = 2 * time.Second
	}

	gormCfg := &gorm.Config{
		PrepareStmt:            cfg.EnablePrepareStmt,
		SkipDefaultTransaction: cfg.SkipDefaultTx,
		Logger:                 logger.Default.LogMode(cfg.GormLogLevel),
	}

	var db *gorm.DB
	var err error
	// 重试连接
	for i := 0; i < cfg.ConnRetryTimes; i++ {
		db, err = gorm.Open(mysql.Open(cfg.DSN), gormCfg)
		if err == nil {
			break
		}
		if i < cfg.ConnRetryTimes-1 {
			log.Printf("gorm open failed (attempt %d/%d): %v, retry after %s", i+1, cfg.ConnRetryTimes, err, cfg.ConnRetryInterval)
			time.Sleep(cfg.ConnRetryInterval)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open gorm: %w", err)
	}

	// 获取底层 sql.DB 操作连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}

	// 连接池配置
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	client := &Client{
		cfg:   cfg,
		DB:    db,
		sqlDB: sqlDB,
	}

	// 简单的一次性健康检查
	if err := client.Ping(context.Background()); err != nil {
		// 连接成功但 ping 失败时，尝试关闭并返回错误
		_ = client.Close()
		return nil, fmt.Errorf("ping failed after open: %w", err)
	}

	return client, nil
}

// Ping 健康检查
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.sqlDB == nil {
		return errors.New("client 未初始化")
	}
	// 使用带超时的 context
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return c.sqlDB.PingContext(ctx)
}

// Close 关闭底层连接池
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	if c.sqlDB == nil {
		return nil
	}
	return c.sqlDB.Close()
}

// WithTransaction 事务包装，自动提交或回滚
func (c *Client) WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	if c == nil || c.DB == nil {
		return errors.New("client 未初始化")
	}
	return c.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// ExecContext 执行非查询 SQL（例如更新、删除、insert）并返回 RowsAffected
func (c *Client) ExecContext(ctx context.Context, sqlStr string, args ...interface{}) (int64, error) {
	res := c.DB.WithContext(ctx).Exec(sqlStr, args...)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// RawQuery 执行原生查询并扫描结果到 dest（dest 必须是指针或切片的指针）
func (c *Client) RawQuery(ctx context.Context, dest interface{}, sqlStr string, args ...interface{}) error {
	res := c.DB.WithContext(ctx).Raw(sqlStr, args...).Scan(dest)
	return res.Error
}

// GetGormDB 返回带有 context 的 gorm.DB，便于链式调用
func (c *Client) GetGormDB(ctx context.Context) *gorm.DB {
	return c.DB.WithContext(ctx)
}

/* 使用示例：

package main

import (
	"context"
	"log"
	"time"

	"your/module/path/gormmysqlsdk"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := gormmysqlsdk.Config{
		DSN:               "user:password@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
		MaxOpenConns:      50,
		MaxIdleConns:      25,
		ConnMaxLifetime:   time.Hour,
		ConnMaxIdleTime:   10 * time.Minute,
		EnablePrepareStmt: true,
		SkipDefaultTx:     true,
		ConnRetryTimes:    3,
		ConnRetryInterval: 2 * time.Second,
		GormLogLevel:      logger.Silent,
	}

	client, err := gormmysqlsdk.NewClient(cfg)
	if err != nil {
		log.Fatalf("init db failed: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	// 事务示例
	err = client.WithTransaction(ctx, func(tx *gorm.DB) error {
		// tx.Create(&User{...})
		return nil
	})
	if err != nil {
		log.Printf("tx err: %v", err)
	}
}
*/

// 性能与调优建议：
// 1) 如果是高并发读场景：开启 PrepareStmt 能减少 SQL 解析成本；同时将读操作尽量设计为只读事务或无事务。
// 2) 若写操作频繁，请谨慎使用 SkipDefaultTx（跳过默认事务）会提高写入吞吐，但丧失事务隔离带来的保证。
// 3) 合理设置 MaxOpenConns / MaxIdleConns 与每个连接的最大生命周期，避免频繁建立拆除连接。
// 4) 若有大量短时间 burst 连接，考虑前端使用连接池或排队限流来平滑流量。
// 5) 在需要更高性能时可以考虑：读写分离（proxy + 主从），使用连接复用及连接持久化机制。

// 扩展点建议：
// - 集成 Prometheus 指标：记录 db_stats（OpenConnections, InUse, Idle, WaitCount, WaitDuration）
// - 支持多 DB 实例注册与获取（通过名字注册多个 *Client）
// - 支持动态调整连接池参数（热更新）
