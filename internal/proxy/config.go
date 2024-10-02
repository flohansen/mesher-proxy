package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
)

type URL url.URL

func (u URL) MarshalJSON() ([]byte, error) {
	b := []byte(fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path))
	return b, nil
}

func (u *URL) UnmarshalJSON(b []byte) error {
	var urlString string
	if err := json.Unmarshal(b, &urlString); err != nil {
		return err
	}

	url, err := url.Parse(urlString)
	if err != nil {
		return err
	}

	*u = URL(*url)
	return nil
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
	Files []string `json:"files"`
	Build []string `json:"build"`
	Exec  []string `json:"exec"`
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
