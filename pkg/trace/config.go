package trace

type Config struct {
	ServiceName *string  `yaml:"service_name,omitempty" json:"service_name,omitempty" toml:"service_name,omitempty"`
	SampleRatio *float64 `yaml:"sample_ratio,omitempty" json:"sample_ratio,omitempty" toml:"sample_ratio,omitempty"`
}
