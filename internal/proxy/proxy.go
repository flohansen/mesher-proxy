//go:generate mockgen -package mocks -source proxy.go -destination mocks/proxy.go

package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

const (
	script = "<script>new WebSocket('/internal/reload').addEventListener('message', () => location.reload())</script>"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Target struct {
	Path string
	URL  *url.URL
}

type Proxy struct {
	addr     string
	client   HttpClient
	services map[string]Target
	u        *websocket.Upgrader
	conns    map[*websocket.Conn]struct{}
	files    []string
	execs    []CmdConfig
	builds   []CmdConfig
}

func NewProxy(opts ...ProxyOpt) *Proxy {
	p := &Proxy{
		addr:     ":8080",
		client:   http.DefaultClient,
		services: make(map[string]Target),
		conns:    make(map[*websocket.Conn]struct{}),
		u: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Proxy) Start() error {
	for _, cmd := range p.builds {
		startCmd(context.Background(), cmd)
	}
	if err := p.startWatcher(p.files, p.execs); err != nil {
		return fmt.Errorf("error starting file watcher: %s", err)
	}

	return http.ListenAndServe(p.addr, p)
}

func (p *Proxy) getUrl(r *http.Request) (Target, bool) {
	for path, target := range p.services {
		if strings.HasPrefix(r.URL.Path, path) {
			return target, true
		}
	}

	return Target{}, false
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/internal/reload" {
		p.handleWebsocket(w, r)
		return
	}

	target, ok := p.getUrl(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	r.Host = target.URL.Host
	r.URL.Host = target.URL.Host
	r.URL.Scheme = target.URL.Scheme
	r.RequestURI = ""

	path, _ := strings.CutPrefix(r.URL.Path, target.Path)
	r.URL.Path = fmt.Sprintf("%s%s", target.URL.Path, path)

	res, err := p.client.Do(r)
	if err != nil {
		log.Printf("error while sending proxy request: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for name, values := range res.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	if strings.HasPrefix(res.Header.Get("Content-Type"), "text/html") {
		htmlContent, err := io.ReadAll(res.Body)
		if err != nil {
			log.Printf("error while reading response: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := strings.Replace(string(htmlContent), "</body>", script+"</body>", 1)
		w.Header().Set("Content-Length", strconv.Itoa(len(response)))
		if _, err := w.Write([]byte(response)); err != nil {
			log.Printf("error while responding: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		io.Copy(w, res.Body)
	}
}

func (p *Proxy) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := p.u.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error creating websocket connection: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	p.conns[conn] = struct{}{}
}

func (p *Proxy) startWatcher(files []string, cmds []CmdConfig) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating file system watcher: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	for _, cmd := range cmds {
		startCmd(ctx, cmd)
	}

	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				if ev.Has(fsnotify.Write) {
					log.Print(ev.Name)
					cancel()

					ctx, cancel = context.WithCancel(context.Background())
					for _, cmd := range cmds {
						startCmd(ctx, cmd)
					}

					for conn := range p.conns {
						conn.WriteMessage(websocket.BinaryMessage, []byte(""))
					}
				}
			case err := <-watcher.Errors:
				log.Fatalf("error watching file system: %s", err)
			}
		}
	}()

	for _, file := range files {
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

type ProxyOpt func(*Proxy)

func WithClient(client HttpClient) ProxyOpt {
	return func(p *Proxy) {
		p.client = client
	}
}

func WithAddr(addr string) ProxyOpt {
	return func(p *Proxy) {
		p.addr = addr
	}
}

func WithTarget(path string, target string) ProxyOpt {
	return func(p *Proxy) {
		url, err := url.Parse(target)
		if err != nil {
			panic(err)
		}

		p.services[path] = Target{
			Path: path,
			URL:  url,
		}
	}
}

func WithFiles(files ...string) ProxyOpt {
	return func(p *Proxy) {
		p.files = append(p.files, files...)
	}
}

func WithBuildCmds(cmds ...CmdConfig) ProxyOpt {
	return func(p *Proxy) {
		p.builds = append(p.builds, cmds...)
	}
}

func WithCmds(cmds ...CmdConfig) ProxyOpt {
	return func(p *Proxy) {
		p.execs = append(p.execs, cmds...)
	}
}

func WithConfig(cfg *Config) ProxyOpt {
	return func(p *Proxy) {
		WithFiles(cfg.Watch.Files...)(p)
		WithBuildCmds(cfg.Watch.Build...)(p)
		WithCmds(cfg.Watch.Exec...)(p)
		WithAddr(cfg.Proxy.Address)(p)

		for path, url := range cfg.Proxy.Targets {
			WithTarget(path, url)(p)
		}
	}
}
