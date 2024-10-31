package proxy

type Config struct {
	Address string            `json:"address"`
	Targets map[string]string `json:"targets"`
}
