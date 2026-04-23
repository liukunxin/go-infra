package milvus

type Config struct {
	Address  string `yaml:"address" json:"address"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	PoolSize int    `yaml:"poolSize" json:"poolSize"`
}
