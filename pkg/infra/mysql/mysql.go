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

// Client wraps a gorm.DB with connection-pool lifecycle management.
type Client struct {
	cfg   Config
	DB    *gorm.DB
	sqlDB *sql.DB
}

// NewClient creates and validates a MySQL client.
// It applies connection-pool defaults, retries the initial connection, and runs a ping.
func NewClient(cfg Config) (*Client, error) {
	if cfg.DSN == "" {
		return nil, errors.New("mysql: DSN must not be empty")
	}
	if cfg.ConnRetryTimes <= 0 {
		cfg.ConnRetryTimes = 3
	}
	if cfg.ConnRetryInterval <= 0 {
		cfg.ConnRetryInterval = 2 * time.Second
	}
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 100
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 10
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = time.Hour
	}
	if cfg.ConnMaxIdleTime <= 0 {
		cfg.ConnMaxIdleTime = 10 * time.Minute
	}

	gormCfg := &gorm.Config{
		PrepareStmt:            cfg.EnablePrepareStmt,
		SkipDefaultTransaction: cfg.SkipDefaultTx,
		Logger:                 logger.Default.LogMode(cfg.GormLogLevel),
	}

	var db *gorm.DB
	var err error
	for i := 0; i < cfg.ConnRetryTimes; i++ {
		db, err = gorm.Open(mysql.Open(cfg.DSN), gormCfg)
		if err == nil {
			break
		}
		if i < cfg.ConnRetryTimes-1 {
			log.Printf("mysql: open failed (attempt %d/%d): %v, retry in %s",
				i+1, cfg.ConnRetryTimes, err, cfg.ConnRetryInterval)
			time.Sleep(cfg.ConnRetryInterval)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("mysql: open gorm: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("mysql: get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	c := &Client{cfg: cfg, DB: db, sqlDB: sqlDB}

	if err := c.Ping(context.Background()); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("mysql: ping after open: %w", err)
	}

	return c, nil
}

// Ping checks connectivity with a 3-second timeout.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.sqlDB == nil {
		return errors.New("mysql: client not initialized")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return c.sqlDB.PingContext(ctx)
}

// Close releases all connections in the pool.
func (c *Client) Close() error {
	if c == nil || c.sqlDB == nil {
		return nil
	}
	return c.sqlDB.Close()
}

// WithTransaction runs fn inside a database transaction.
// Commits on nil return, rolls back on any error.
func (c *Client) WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	if c == nil || c.DB == nil {
		return errors.New("mysql: client not initialized")
	}
	return c.DB.WithContext(ctx).Transaction(fn)
}

// ExecContext executes a non-query statement and returns the number of rows affected.
func (c *Client) ExecContext(ctx context.Context, sqlStr string, args ...interface{}) (int64, error) {
	if c == nil || c.DB == nil {
		return 0, errors.New("mysql: client not initialized")
	}
	res := c.DB.WithContext(ctx).Exec(sqlStr, args...)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// RawQuery executes a raw SELECT and scans results into dest (must be a pointer or slice pointer).
func (c *Client) RawQuery(ctx context.Context, dest interface{}, sqlStr string, args ...interface{}) error {
	if c == nil || c.DB == nil {
		return errors.New("mysql: client not initialized")
	}
	return c.DB.WithContext(ctx).Raw(sqlStr, args...).Scan(dest).Error
}

// GetGormDB returns a gorm.DB scoped to ctx for chained calls.
func (c *Client) GetGormDB(ctx context.Context) *gorm.DB {
	return c.DB.WithContext(ctx)
}
