package v8

import "time"

type Config struct {
	Mode         string        `json:"mode" yaml:"mode"`
	Addresses    []string      `json:"addresses" yaml:"addresses"`
	Password     string        `json:"password" yaml:"password"`
	PoolSize     int           `json:"pool_size" yaml:"pool_size"`
	MinIdleConns int           `json:"min_idle_conns" yaml:"min_idle_conns"`
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
}
