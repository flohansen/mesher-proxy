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

func StartWatcher(proxy *proxy.Proxy, config WatchConfig) error {
	for _, cmd := range config.Build {
		startCmd(context.Background(), cmd)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating file system watcher: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	for _, cmd := range config.Exec {
		startCmd(ctx, cmd)
	}

	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				if ev.Has(fsnotify.Write) {
					fmt.Printf(color.Green+"detected change in"+color.Reset+" %s\n", ev.Name)
					cancel()

					ctx, cancel = context.WithCancel(context.Background())
					for _, cmd := range config.Exec {
						fmt.Printf(color.Yellow+"execute"+color.Reset+" %s\n", cmd.Cmd)
						startCmd(ctx, cmd)
					}

					proxy.RefreshConnections()
					fmt.Print(color.Yellow + "done" + color.Reset + "\n")
				}
			case err := <-watcher.Errors:
				log.Fatalf("error watching file system: %s", err)
			}
		}
	}()

	for _, file := range config.Files {
		watcher.Add(file)
	}

	return nil
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
