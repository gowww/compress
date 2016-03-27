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
	type testCase struct {
		needsGzip bool
		content   []byte
	}
	var cc []*testCase
	for _, c := range []struct {
		gzippableType bool
		contentPrefix []byte
	}{
		{false, []byte("%PDF-")},
		{true, []byte("<!DOCTYPE HTML>")},
	} {
		// Small content
		cc = append(cc, &testCase{false, c.contentPrefix})
		// Big content
		b := append(c.contentPrefix, make([]byte, gzippableMinSize)...)
		cc = append(cc, &testCase{c.gzippableType, b})
	}

	ts := httptest.NewServer(HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeHeader {
			w.WriteHeader(http.StatusOK)
		}
		io.Copy(w, r.Body)
	}))
	defer ts.Close()

	for _, c := range cc {
		req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(c.content))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Accept-Encoding", "gzip")

		res, err := new(http.Client).Do(req)
		if err != nil {
			t.Fatal(err)
		}

		ce := res.Header.Get("Content-Encoding")
		if c.needsGzip && ce != "gzip" {
			t.Errorf("%v bytes %q needs gzip", len(c.content), res.Header.Get("Content-Type"))
		} else if !c.needsGzip && ce != "" {
			t.Errorf("%v bytes %q doesn't need gzip", len(c.content), res.Header.Get("Content-Type"))
		}
	}
}
