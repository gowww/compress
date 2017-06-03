package compress

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testableContent struct {
	needsGzip bool
	body      []byte
}

var testableContents = []*testableContent{
	{false, []byte("")},
	{false, []byte("foobar")},
	{false, []byte("%PDF-")},
	{false, []byte("<!DOCTYPE HTML>")},
	{true, addGzippableMinSize([]byte(""))},
	{true, addGzippableMinSize([]byte("foobar"))},
	{false, addGzippableMinSize([]byte("%PDF-"))},
	{true, addGzippableMinSize([]byte("<!DOCTYPE HTML>"))},
}

type testCase struct {
	t              *testing.T
	acceptEncoding string
	f              http.HandlerFunc
	test           func(*testableContent, *http.Response) []string
}

func test(tc *testCase) {
	ts := httptest.NewServer(HandleFunc(tc.f))
	defer ts.Close()

	for _, c := range testableContents {
		req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(c.body))
		if err != nil {
			tc.t.Fatal(err)
		}
		req.Header.Set("Accept-Encoding", tc.acceptEncoding)

		res, err := new(http.Client).Do(req)
		if err != nil {
			tc.t.Fatal(err)
		}

		if errs := tc.test(c, res); len(errs) > 0 {
			for _, err := range errs {
				tc.t.Errorf("%v bytes %q: %v", len(c.body), res.Header.Get("Content-Type"), err)
				// tc.t.Log(string(c.body))
			}
		}
	}
}

func TestGzipAcceptance(t *testing.T) {
	test(&testCase{
		t:              t,
		acceptEncoding: "otherThanGzip",
		f: func(w http.ResponseWriter, r *http.Request) {
			io.Copy(w, r.Body)
		},
		test: func(_ *testableContent, res *http.Response) (errs []string) {
			if resce := res.Header.Get("Content-Encoding"); resce != "" {
				errs = append(errs, "gzipped response without client acceptance")
			}
			return
		},
	})
}

func TestResponseAlreadyEncoded(t *testing.T) {
	encoding := "otherThanGzip"
	test(&testCase{
		t:              t,
		acceptEncoding: "gzip",
		f: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Encoding", encoding)
			io.Copy(w, r.Body)
		},
		test: func(c *testableContent, res *http.Response) (errs []string) {
			if resce := res.Header.Get("Content-Encoding"); resce != encoding {
				errs = append(errs, fmt.Sprintf("response already encoded: want %v, got %v", encoding, resce))
			}
			return
		},
	})
}

func TestGzip(t *testing.T) {
	test(&testCase{
		t:              t,
		acceptEncoding: "gzip",
		f: func(w http.ResponseWriter, r *http.Request) {
			io.Copy(w, r.Body)
		},
		test: func(c *testableContent, res *http.Response) (errs []string) {
			if resce := res.Header.Get("Content-Encoding"); c.needsGzip && resce != "gzip" {
				errs = append(errs, "gzip needed")
			} else if !c.needsGzip && resce != "" {
				errs = append(errs, "gzip not needed")
			}
			return
		},
	})
}

func TestGzipWriteHeader(t *testing.T) {
	status := http.StatusTeapot
	test(&testCase{
		t:              t,
		acceptEncoding: "gzip",
		f: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			io.Copy(w, r.Body)
		},
		test: func(c *testableContent, res *http.Response) (errs []string) {
			if res.StatusCode != status {
				errs = append(errs, fmt.Sprintf("write header: want %v, got %v", status, res.StatusCode))
			}
			return
		},
	})
}

func addGzippableMinSize(b []byte) []byte {
	return append(b, make([]byte, gzippableMinSize)...)
}
