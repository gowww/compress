package compress

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func testInterfaces(hf http.HandlerFunc, useServer bool) {
	log.SetOutput(ioutil.Discard)
	log.SetFlags(0)
	defer log.SetOutput(os.Stderr)

	h := HandleFunc(hf)

	if useServer {
		ts := httptest.NewServer(h)
		defer ts.Close()
		req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
		req.Header.Set("Accept-Encoding", "gzip")
		http.DefaultClient.Do(req)
	} else {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}

var handlerFuncCloseNotify = func(w http.ResponseWriter, r *http.Request) {
	n, ok := w.(http.CloseNotifier)
	if ok {
		cn := n.CloseNotify()
		go func() {
			<-cn
		}()
	}
}

func TestCloseNotify(t *testing.T) {
	testInterfaces(handlerFuncCloseNotify, true)
}

func TestNoCloseNotify(t *testing.T) {
	testInterfaces(handlerFuncCloseNotify, false)
}

func TestFlush(t *testing.T) {
	testInterfaces(func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		if ok {
			f.Flush()
		}
	}, true)
}

var handlerFuncHijack = func(w http.ResponseWriter, r *http.Request) {
	h, ok := w.(http.Hijacker)
	if ok {
		conn, _, err := h.Hijack()
		if err == nil {
			conn.Close()
		}
	}
}

func TestHijack(t *testing.T) {
	testInterfaces(handlerFuncHijack, true)
}

func TestNoHijack(t *testing.T) {
	testInterfaces(handlerFuncHijack, false)
}

var handlerFuncPush = func(w http.ResponseWriter, r *http.Request) {
	p, ok := w.(http.Pusher)
	if ok {
		p.Push("/", nil)
	}
}

func TestPush(t *testing.T) {
	// TODO: Set HTTP/2 test server.
}

func TestNoPush(t *testing.T) {
	testInterfaces(handlerFuncPush, false)
}
