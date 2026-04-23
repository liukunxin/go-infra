package ks3

type Config struct {
	Region   string `yaml:"region" json:"region"`
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	Ak       string `yaml:"ak" json:"ak"`
	Sk       string `yaml:"sk" json:"sk"`
}
