package proxy

import (
	"encoding/json"
	"log"
	"os"
)

func str(str string) *string {
	return &str
}

var DefaultConfig = &Config{
	Watch: WatchConfig{
		Files: []string{"./templates/"},
		Build: []CmdConfig{
			{
				Cmd: "go build -o ./tmp/main ./cmd/main.go",
			},
		},
		Exec: []CmdConfig{
			{
				Cmd:       "./tmp/main",
				Condition: str("curl -Is http://localhost:3000/health -o /dev/null"),
			},
		},
	},
	Proxy: ProxyConfig{
		Address: ":8080",
		Targets: map[string]string{
			"/": "http://localhost:3000/",
		},
	},
}

type Config struct {
	Watch WatchConfig `json:"watch"`
	Proxy ProxyConfig `json:"proxy"`
}

type ProxyConfig struct {
	Address string            `json:"address"`
	Targets map[string]string `json:"targets"`
}

type WatchConfig struct {
	Files []string    `json:"files"`
	Build []CmdConfig `json:"build"`
	Exec  []CmdConfig `json:"exec"`
}

type CmdConfig struct {
	Cmd       string  `json:"cmd"`
	Condition *string `json:"condition,omitempty"`
}

func NewConfig(opts ...ProxyConfigOpt) *Config {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

type ProxyConfigOpt func(*Config)

func FromFile(name string) ProxyConfigOpt {
	return func(cfg *Config) {
		f, err := os.Open(name)
		if err != nil {
			log.Fatalf("error opening config file: %s", err)
		}
		defer f.Close()

		if err := json.NewDecoder(f).Decode(cfg); err != nil {
			log.Fatalf("error decoding config file: %s", err)
		}
	}
}
