package compress

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGzipAcceptance(t *testing.T) {
	b := []byte("<!DOCTYPE HTML>")
	b = append(b, make([]byte, gzippableMinSize)...)
	w := httptest.NewRecorder()

	HandleFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(b)
	}).ServeHTTP(w, new(http.Request)) // New request without "Accept-Encoding: gzip" header.

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Errorf("gzipped response without client acceptance")
	}
}

func TestGzip(t *testing.T) {
	testGzip(t, false)
}

func TestGzipWriteHeader(t *testing.T) {
	testGzip(t, true)
}

func testGzip(t *testing.T, writeHeader bool) {
	cases := []*struct {
		needsGzip bool
		content   []byte
	}{
		{false, []byte("")},
		{false, []byte("x")},
		{false, []byte("%PDF-")},
		{false, []byte("<!DOCTYPE HTML>")},
		{true, addGzippableMinSize([]byte(""))},
		{true, addGzippableMinSize([]byte("x"))},
		{false, addGzippableMinSize([]byte("%PDF-"))},
		{true, addGzippableMinSize([]byte("<!DOCTYPE HTML>"))},
	}

	status := http.StatusOK
	if writeHeader {
		status = http.StatusTeapot
	}

	ts := httptest.NewServer(HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeHeader {
			w.WriteHeader(status)
		}
		io.Copy(w, r.Body)
	}))
	defer ts.Close()

	for _, c := range cases {
		req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(c.content))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Accept-Encoding", "gzip")

		res, err := new(http.Client).Do(req)
		if err != nil {
			t.Fatal(err)
		}

		if writeHeader && res.StatusCode != status {
			t.Errorf("%v bytes %q: write header call: want %v, got %v", len(c.content), res.Header.Get("Content-Type"), status, res.StatusCode)
		}

		ce := res.Header.Get("Content-Encoding")
		if c.needsGzip && ce != "gzip" {
			t.Errorf("%v bytes %q needs gzip", len(c.content), res.Header.Get("Content-Type"))
		} else if !c.needsGzip && ce != "" {
			t.Errorf("%v bytes %q doesn't need gzip", len(c.content), res.Header.Get("Content-Type"))
		}
	}
}

func addGzippableMinSize(b []byte) []byte {
	return append(b, make([]byte, gzippableMinSize)...)
}
