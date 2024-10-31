package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/flohansen/sentinel/internal/cli"
)

var version string

var (
	configFile = flag.String("config", ".sentinel.json", "The path to the config file")
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer stop()

	flag.Parse()
	app := cli.NewApp(version)

	if len(os.Args) == 1 {
		app.PrintHelp(os.Stdout)
		return
	}

	switch os.Args[1] {
	case "run":
		config := cli.NewConfigFromFile(*configFile)
		if err := app.Run(ctx, config); err != nil {
			log.Fatalf("run error: %s", err)
		}
	case "version":
		app.PrintVersion(os.Stdout)
	case "init":
		if err := app.Init(); err != nil {
			log.Fatalf("init error: %s", err)
		}
	default:
		log.Fatalf("unknown argument '%s'", os.Args[1])
	}
}
