package drp

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
)

var logger = log.New(os.Stderr, "", 0)

// Run runs a proxy with the default parameters
func Run() error {
	p := New()
	err := p.Start()
	if err != nil {
		return err
	}
	return p.Wait()
}

func New() *Proxy {
	return &Proxy{}
}

type Proxy struct {
	mutex   sync.Mutex
	px      http.Handler
	wg      *sync.WaitGroup
	configs map[string]*Config
}

func (p *Proxy) Start() error {
	p.wg = &sync.WaitGroup{}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		err := p.startAdmin()
		if err != nil {
			logger.Printf("error starting admin. err=%q", err)
		}
	}()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		err := p.startLB()
		if err != nil {
			logger.Printf("error starting lb. err=%q", err)
		}
	}()
	return nil
}

func (p *Proxy) startLB() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	if !strings.Contains(port, ":") {
		port = "0.0.0.0:" + port
	}
	return http.ListenAndServe(port, http.HandlerFunc(p.index))
}

func (p *Proxy) startAdmin() error {
	port := os.Getenv("ADMIN_PORT")
	if port == "" {
		port = "8001"
	}
	if !strings.Contains(port, ":") {
		port = "0.0.0.0:" + port
	}
	return http.ListenAndServe(port, p.adminMux())
}

func (p *Proxy) Wait() error {
	p.wg.Wait()
	return nil
}

func (p *Proxy) adminMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.adminHandler)
	return mux
}

func (p *Proxy) updateConfig(cfg *Config) error {
	if cfg.Address == "" {
		return errors.New("address must be set")
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.configs == nil {
		p.configs = map[string]*Config{}
	}
	if !strings.HasSuffix(cfg.Path, "/") {
		cfg.Path += "/"
	}
	p.configs[cfg.Path] = cfg

	mux := http.NewServeMux()

	for path, cfg := range p.configs {
		u, err := url.Parse(cfg.Address)
		if err != nil {
			return err
		}
		mux.Handle(path, httputil.NewSingleHostReverseProxy(u))
	}
	p.px = mux
	return nil
}

func (p *Proxy) proxy() http.Handler {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.px
}

func (p *Proxy) Configs() ([]byte, error) {
	return json.Marshal(p.configs)
}

func (p *Proxy) adminHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		p.getConfigHandler(w, r)
	case "POST":
		p.updateConfigHandler(w, r)
	default:
		code := http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(code), code)
	}
}

func (p *Proxy) getConfigHandler(w http.ResponseWriter, r *http.Request) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.configs == nil {
		p.configs = map[string]*Config{}
	}
	writeJSON(w, http.StatusOK, p.configs)
}

func writeJSON(w http.ResponseWriter, code int, i interface{}) {
	b, err := json.Marshal(i)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)

}

func (p *Proxy) updateConfigHandler(w http.ResponseWriter, r *http.Request) {
	var cfg *Config
	err := func() error {
		err := json.NewDecoder(r.Body).Decode(&cfg)
		if err != nil {
			return err
		}
		if err := p.updateConfig(cfg); err != nil {
			return err
		}
		writeJSON(w, http.StatusCreated, cfg)
		return nil
	}()
	if err != nil {
		logger.Printf("err=%q", err)
		http.Error(w, "Not Acceptable: "+err.Error(), http.StatusNotAcceptable)
		return
	}
}

func (p *Proxy) index(w http.ResponseWriter, r *http.Request) {
	p.proxy().ServeHTTP(w, r)
}
