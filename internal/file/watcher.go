package file

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/flohansen/sentinel/internal/color"
	"github.com/flohansen/sentinel/internal/proxy"
	"github.com/fsnotify/fsnotify"
)

type WatchConfig struct {
	Files []string    `json:"files"`
	Build []CmdConfig `json:"build"`
	Exec  []CmdConfig `json:"exec"`
}

type CmdConfig struct {
	Cmd       string  `json:"cmd"`
	Condition *string `json:"condition,omitempty"`
}

type Watcher struct {
	proxy  *proxy.Proxy
	files  []string
	execs  []CmdConfig
	builds []CmdConfig
}

func NewWatcher(proxy *proxy.Proxy, config WatchConfig) *Watcher {
	return &Watcher{
		proxy:  proxy,
		files:  config.Files,
		execs:  config.Exec,
		builds: config.Build,
	}
}

func (w *Watcher) Start() error {
	for _, cmd := range w.builds {
		startCmd(context.Background(), cmd)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating file system watcher: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	for _, cmd := range w.execs {
		startCmd(ctx, cmd)
	}
	defer cancel()

	for _, file := range w.files {
		watcher.Add(file)
	}

	for {
		select {
		case ev := <-watcher.Events:
			if ev.Has(fsnotify.Write) {
				fmt.Printf(color.Green+"detected change in"+color.Reset+" %s\n", ev.Name)
				cancel()

				ctx, cancel = context.WithCancel(context.Background())
				defer cancel()

				for _, cmd := range w.execs {
					fmt.Printf(color.Yellow+"execute"+color.Reset+" %s\n", cmd.Cmd)
					startCmd(ctx, cmd)
				}

				w.proxy.RefreshConnections()
				fmt.Print(color.Yellow + "done" + color.Reset + "\n")
			}
		case err := <-watcher.Errors:
			return fmt.Errorf("error watching file system: %s", err)
		}
	}
}

func startCmd(ctx context.Context, cmd CmdConfig) {
	args := strings.Split(cmd.Cmd, " ")
	c := exec.CommandContext(ctx, args[0], args[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if cmd.Condition != nil {
		if err := c.Start(); err != nil {
			log.Printf("error running command: %s", err)
		}

		for {
			args := strings.Split(*cmd.Condition, " ")
			c := exec.CommandContext(ctx, args[0], args[1:]...)
			if err := c.Run(); err == nil {
				break
			}

			time.Sleep(1 * time.Second)
		}
	} else {
		if err := c.Run(); err != nil {
			log.Printf("error running command: %s", err)
		}
	}
}
