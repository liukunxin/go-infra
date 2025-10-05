package trace

type Config struct {
	ServiceName *string  `yaml:"serviceName,omitempty" json:"serviceName,omitempty" toml:"serviceName,omitempty"`
	SampleRatio *float64 `yaml:"sampleRatio,omitempty" json:"sampleRatio,omitempty" toml:"sampleRatio,omitempty"`
}
