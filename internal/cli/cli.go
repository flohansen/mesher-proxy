package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/flohansen/sentinel/internal/proxy"
)

func Init() error {
	f, err := os.Create(".sentinel.json")
	if err != nil {
		return fmt.Errorf("error opening file: %s", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(proxy.DefaultConfig); err != nil {
		return fmt.Errorf("error encoding config: %s", err)
	}

	return nil
}
