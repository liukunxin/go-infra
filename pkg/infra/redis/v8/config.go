package v8

import "time"

// Config holds Redis connection parameters.
// Mode must be "single" or "cluster".
// All Duration fields use standard Go duration notation (e.g. 30s, 5m).
type Config struct {
	Mode         string        `json:"mode"           yaml:"mode"`            // "single" or "cluster"
	Addresses    []string      `json:"addresses"      yaml:"addresses"`        // single: one address; cluster: all seed nodes
	Password     string        `json:"password"       yaml:"password"`
	PoolSize     int           `json:"pool_size"      yaml:"pool_size"`        // connections per node, default 10
	MinIdleConns int           `json:"min_idle_conns" yaml:"min_idle_conns"`   // default 0
	IdleTimeout  time.Duration `json:"idle_timeout"   yaml:"idle_timeout"`     // e.g. "5m"; 0 = no timeout
}
