package config

type Config struct {
	UpstreamConfig UpstreamConfig `json:"upstream"`
	Filters        Filters        `json:"filters"`
}

type UpstreamConfig struct {
	Target string `json:"target"`
}

type Filters []struct {
	Path string `json:"path"`
}
