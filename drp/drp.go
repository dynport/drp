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
	mutex sync.Mutex
	px    http.HandlerFunc
	cfg   *Config
	wg    *sync.WaitGroup
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
	return http.ListenAndServe(port, p.lbMux())
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

func (b *Proxy) lbMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", b.index)
	return mux
}

func (p *Proxy) adminMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.adminHandler)
	return mux
}

type Config struct {
	Address  string          `json:"address,omitempty"`
	Path     string          `json:"path,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

func (p *Proxy) setProxy(cfg *Config) error {
	u, err := url.Parse(cfg.Address)
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.cfg = cfg
	p.px = httputil.NewSingleHostReverseProxy(u).ServeHTTP
	return nil
}

func (p *Proxy) config() *Config {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.cfg == nil {
		return &Config{}
	}
	return p.cfg
}

func (p *Proxy) proxy() http.HandlerFunc {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.px
}

func (p *Proxy) adminHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		cfg := p.config()
		b, err := json.Marshal(cfg)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(b)
		return
	} else if r.Method == "POST" {
		var cfg *Config
		err := func() error {
			err := json.NewDecoder(r.Body).Decode(&cfg)
			if err != nil {
				return err
			}
			if cfg.Address == "" {
				return errors.New("address must be set")
			}
			if err := p.setProxy(cfg); err != nil {
				return err
			}
			b, err := json.Marshal(cfg)
			if err != nil {
				return err
			}
			w.WriteHeader(http.StatusCreated)
			w.Write(b)
			return nil
		}()
		if err != nil {
			logger.Printf("err=%q", err)
			http.Error(w, "Not Acceptable: "+err.Error(), http.StatusNotAcceptable)
			return
		}
	}
	return
}

func (p *Proxy) index(w http.ResponseWriter, r *http.Request) {
	p.proxy()(w, r)
}
