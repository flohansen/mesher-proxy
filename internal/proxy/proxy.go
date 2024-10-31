//go:generate mockgen -package mocks -source proxy.go -destination mocks/proxy.go

package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/flohansen/sentinel/internal/color"
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
	return http.ListenAndServe(p.addr, p)
}

func (p *Proxy) RefreshConnections() {
	for conn := range p.conns {
		fmt.Printf(color.Yellow+"refresh"+color.Reset+" %s\n", conn.RemoteAddr())
		conn.WriteMessage(websocket.BinaryMessage, []byte(""))
	}
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
	defer conn.Close()

	p.conns[conn] = struct{}{}

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	delete(p.conns, conn)
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

func WithConfig(cfg Config) ProxyOpt {
	return func(p *Proxy) {
		WithAddr(cfg.Address)(p)

		for path, url := range cfg.Targets {
			WithTarget(path, url)(p)
		}
	}
}
