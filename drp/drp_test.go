package drp

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func pongServer(s string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(s)) }))
}

func adminAndProxy() (admin, proxy *httptest.Server) {
	p := New()
	h := p.adminMux()

	return httptest.NewServer(h), httptest.NewServer(http.HandlerFunc(p.index))
}

func TestMultipleHosts(t *testing.T) {
	a := pongServer("pong a")
	b := pongServer("pong b")

	admin, proxy := adminAndProxy()

	if rsp, err := updateConfig(admin.URL, &Config{Address: a.URL}); err != nil {
		t.Fatal(err)
	} else if rsp.code != 201 {
		t.Errorf("expected code to be 201, was %d", rsp.code)
	}

	if rsp, err := updateConfig(admin.URL, &Config{Path: "some.host.com", Address: b.URL}); err != nil {
		t.Fatal(err)
	} else if rsp.code != 201 {
		t.Errorf("expected code to be 201, was %d", rsp.code)
	}

	if rsp, err := rspWrapper(http.Get(admin.URL)); err != nil {
		t.Fatal(err)
	} else {
		ex := `{"/":{"address":"` + a.URL + `","path":"/"},"some.host.com/":{"address":"` + b.URL + `","path":"some.host.com/"}}`

		if rsp.body != ex {
			t.Errorf("expected body to be\n%q\nwas\n%q", rsp.body, ex)
		}
	}

	rsp, err := rspWrapper(http.Get(proxy.URL))
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", proxy.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "some.host.com"

	rsp2, err := rspWrapper(http.DefaultClient.Do(req))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Name     string
		Expected interface{}
		Value    interface{}
	}{
		{"rw.Code", 200, rsp.code},
		{"rw.Body", "pong a", rsp.body},
		{"rw2.Code", 200, rsp2.code},
		{"rw2.Body", "pong b", rsp2.body},
	}

	for _, tst := range tests {
		if tst.Expected != tst.Value {
			t.Errorf("expected %s to be %#v, was %#v", tst.Name, tst.Expected, tst.Value)
		}
	}
}

func TestIntegration(t *testing.T) {
	a := pongServer("first server")
	b := pongServer("second server")

	admin, proxy := adminAndProxy()

	rsp, err := rspWrapper(http.Get(admin.URL + "/"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Name     string
		Expected interface{}
		Value    interface{}
	}{
		{"Code", rsp.code, 200},
		{"Body", rsp.body, "{}"},
	}

	for _, tst := range tests {
		if tst.Expected != tst.Value {
			t.Errorf("expected %s to be %#v, was %#v", tst.Name, tst.Expected, tst.Value)
		}
	}

	rsp, err = updateConfig(admin.URL, &Config{Address: a.URL})
	if err != nil {
		t.Fatal(err)
	}

	tests = []struct {
		Name     string
		Expected interface{}
		Value    interface{}
	}{
		{"Code", 201, rsp.code},
		{"Body", `{"address":"` + a.URL + `","path":"/"}`, rsp.body},
	}

	for _, tst := range tests {
		if tst.Expected != tst.Value {
			t.Errorf("expected %s to be %#v, was %#v", tst.Name, tst.Expected, tst.Value)
		}
	}

	rsp, err = rspWrapper(http.Get(proxy.URL))
	if err != nil {
		t.Fatal(err)
	}

	tests = []struct {
		Name     string
		Expected interface{}
		Value    interface{}
	}{
		{"Code", 200, rsp.code},
		{"Body", "first server", rsp.body},
	}

	for _, tst := range tests {
		if tst.Expected != tst.Value {
			t.Errorf("expected %s to be %#v, was %#v", tst.Name, tst.Expected, tst.Value)
		}
	}

	rsp, err = updateConfig(admin.URL, &Config{Address: b.URL})
	if err != nil {
		t.Fatal(err)
	}

	tests = []struct {
		Name     string
		Expected interface{}
		Value    interface{}
	}{
		{"Code", 201, rsp.code},
		{"Body", `{"address":"` + b.URL + `"}`, rsp.body},
	}

	rsp, err = rspWrapper(http.Get(proxy.URL))
	if err != nil {
		t.Fatal(err)
	}

	tests = []struct {
		Name     string
		Expected interface{}
		Value    interface{}
	}{
		{"Code", 200, rsp.code},
		{"Body", "second server", rsp.body},
	}

	for _, tst := range tests {
		if tst.Expected != tst.Value {
			t.Errorf("expected %s to be %#v, was %#v", tst.Name, tst.Expected, tst.Value)
		}
	}
}

// verify how the go default mux handles subdomains
func TestMux(t *testing.T) {
	m := http.NewServeMux()
	m.HandleFunc("domain.de/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("domain.de")) })
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("index"))
	})
	m.HandleFunc("sub.domain.de/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("sub.domain.de")) })

	s := httptest.NewServer(m)

	tests := []struct {
		Path string
		Host string
		Code int
		Body string
	}{
		{"/", "", 200, "index"},
		{"/", "domain.de", 200, "domain.de"},
		{"/", "sub.domain.de", 200, "sub.domain.de"},
	}

	for _, tst := range tests {
		req, err := http.NewRequest("GET", s.URL+tst.Path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if tst.Host != "" {
			req.Host = tst.Host
		}
		rsp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("error doing request: %s", err)
			continue
		}
		if rsp.StatusCode != tst.Code {
			t.Errorf("expected Code to be %v, was %v", tst.Code, rsp.StatusCode)
		}
		b, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			t.Errorf("error reading body: %s", err)
			continue
		}
		if string(b) != tst.Body {
			t.Errorf("expected Code to be %v, was %v", tst.Body, string(b))
		}
	}

}

func updateConfig(adminURL string, c *Config) (*rsp, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return rspWrapper(http.Post(adminURL, "application/json", bytes.NewReader(b)))
}

type rsp struct {
	code int
	body string
}

func rspWrapper(r *http.Response, err error) (*rsp, error) {
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return &rsp{code: r.StatusCode, body: string(b)}, nil
}

func init() {
	logger = log.New(ioutil.Discard, "", 0)
}
