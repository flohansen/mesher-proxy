package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/flohansen/sentinel/internal/color"
	"github.com/flohansen/sentinel/internal/file"
	"github.com/flohansen/sentinel/internal/proxy"
)

type App struct {
	version string
}

func NewApp(version string) *App {
	return &App{version}
}

func (app *App) PrintVersion(w io.Writer) {
	w.Write([]byte(fmt.Sprintf("sentinel v%s\n", app.version)))
}

func (app *App) PrintHelp(w io.Writer) {
	app.PrintVersion(w)
	for _, line := range []string{
		"",
		"Usage: sentinel <command>",
		"",
		"Commands:",
		"  init     Create default configuration file",
		"  run      Run sentinel proxy",
		"  version  Print the binary version",
	} {
		w.Write([]byte(line + "\n"))
	}
}

func (app *App) Run(ctx context.Context, config Config) error {
	proxy := proxy.NewProxy(proxy.WithClient(&http.Client{}), proxy.WithConfig(config.Proxy))
	watcher := file.NewWatcher(config.Watch.Files)

	errs := make(chan error)
	defer close(errs)

	go func() {
		if err := watcher.Start(ctx); err != nil {
			errs <- fmt.Errorf("watcher error: %s", err)
		}

		log.Println("watcher closed")
	}()

	go func() {
		if err := proxy.Start(ctx); err != nil {
			errs <- fmt.Errorf("proxy error: %s", err)
		}

		log.Println("proxy closed")
	}()

	cmdContext, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, cmd := range config.Watch.Build {
		startCmd(cmdContext, cmd)
	}

	for _, cmd := range config.Watch.Exec {
		startCmd(cmdContext, cmd)
	}

	for file := range watcher.FileChanges() {
		fmt.Printf(color.Green+"detected change in"+color.Reset+" %s\n", file)
		cancel()

		cmdContext, cancel = context.WithCancel(ctx)
		defer cancel()

		for _, cmd := range config.Watch.Exec {
			fmt.Printf(color.Yellow+"execute"+color.Reset+" %s\n", cmd.Cmd)
			startCmd(cmdContext, cmd)
		}

		proxy.RefreshConnections()
		fmt.Print(color.Yellow + "done" + color.Reset + "\n")
	}

	log.Println("file chan closed")
	return nil
}

func (a *App) Init() error {
	f, err := os.Create(".sentinel.json")
	if err != nil {
		return fmt.Errorf("error opening file: %s", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(DefaultConfig); err != nil {
		return fmt.Errorf("error encoding config: %s", err)
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
