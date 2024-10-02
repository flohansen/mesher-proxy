package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/flohansen/sentinel/internal/proxy"
)

var (
	configFile = flag.String("config", ".sentinel.json", "The path to the config file")
)

func main() {
	flag.Parse()

	cfg := proxy.NewConfig(proxy.FromFile(*configFile))
	proxy := proxy.NewProxy(proxy.WithClient(&http.Client{}), proxy.WithConfig(cfg))
	log.Fatalf("error: %s", proxy.Start())
}
