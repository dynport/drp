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

func TestIntegration(t *testing.T) {
	p := New()
	h := p.adminMux()

	admin := httptest.NewServer(h)

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

	s := pongServer("first server")

	b, err := json.Marshal(&Config{Address: s.URL})
	if err != nil {
		t.Fatal(err)
	}
	rsp, err = rspWrapper(http.Post(admin.URL, "application/json", bytes.NewReader(b)))
	if err != nil {
		t.Fatal(err)
	}
	tests = []struct {
		Name     string
		Expected interface{}
		Value    interface{}
	}{
		{"Code", 201, rsp.code},
		{"Body", `{"address":"` + s.URL + `"}`, rsp.body},
	}

	for _, tst := range tests {
		if tst.Expected != tst.Value {
			t.Errorf("expected %s to be %#v, was %#v", tst.Name, tst.Expected, tst.Value)
		}
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rw := httptest.NewRecorder()
	p.index(rw, req)

	b, err = ioutil.ReadAll(rw.Body)
	if err != nil {
		t.Fatal(err)
	}

	tests = []struct {
		Name     string
		Expected interface{}
		Value    interface{}
	}{
		{"Code", 200, rw.Code},
		{"Body", "first server", string(b)},
	}

	for _, tst := range tests {
		if tst.Expected != tst.Value {
			t.Errorf("expected %s to be %#v, was %#v", tst.Name, tst.Expected, tst.Value)
		}
	}

	s = pongServer("second server")

	b, err = json.Marshal(&Config{Address: s.URL})
	if err != nil {
		t.Fatal(err)
	}
	rsp, err = rspWrapper(http.Post(admin.URL, "application/json", bytes.NewReader(b)))
	if err != nil {
		t.Fatal(err)
	}
	tests = []struct {
		Name     string
		Expected interface{}
		Value    interface{}
	}{
		{"Code", 201, rsp.code},
		{"Body", `{"address":"` + s.URL + `"}`, rsp.body},
	}

	rw = httptest.NewRecorder()
	p.index(rw, req)

	b, err = ioutil.ReadAll(rw.Body)
	if err != nil {
		t.Fatal(err)
	}

	tests = []struct {
		Name     string
		Expected interface{}
		Value    interface{}
	}{
		{"Code", 200, rw.Code},
		{"Body", "second server", string(b)},
	}

	for _, tst := range tests {
		if tst.Expected != tst.Value {
			t.Errorf("expected %s to be %#v, was %#v", tst.Name, tst.Expected, tst.Value)
		}
	}

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

func init() {
	logger = log.New(ioutil.Discard, "", 0)
}
