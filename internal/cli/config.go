package cli

import (
	"encoding/json"
	"log"
	"os"

	"github.com/flohansen/sentinel/internal/proxy"
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
	Proxy: proxy.Config{
		Address: ":8080",
		Targets: map[string]string{
			"/": "http://localhost:3000/",
		},
	},
}

type Config struct {
	Version string
	Watch   WatchConfig  `json:"watch"`
	Proxy   proxy.Config `json:"proxy"`
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

func NewConfigFromFile(name string) Config {
	f, err := os.Open(name)
	if err != nil {
		log.Fatalf("error opening config file: %s", err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatalf("error decoding config file: %s", err)
	}

	return cfg
}
