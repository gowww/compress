/*
Package compress provides a clever gzip compressing handler.

It takes care to not handle small contents, or contents that are already compressed (like JPEG, MPEG or PDF).
Trying to gzip them not only wastes CPU but can potentially increase the response size.

Make sure to include this handler above any other handler that alter the response body.
*/
package compress

import (
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var (
	gzipPool = sync.Pool{New: func() interface{} {
		return gzip.NewWriter(nil)
	}}

	gzippableMinSize = 150

	notGzippableTypes = map[string]struct{}{
		"application/font-woff": {},
		"application/gzip":      {},
		"application/pdf":       {},
		"application/zip":       {},
		"audio/mp4":             {},
		"audio/mpeg":            {},
		"audio/webm":            {},
		"image/gif":             {},
		"image/jpeg":            {},
		"image/png":             {},
		"image/webp":            {},
		"video/h264":            {},
		"video/mp4":             {},
		"video/mpeg":            {},
		"video/ogg":             {},
		"video/vp8":             {},
		"video/webm":            {},
	}
)

// An Handler provides a clever gzip compressing handler.
type Handler struct {
	Next http.Handler
}

// Handle returns a Handler wrapping another http.Handler.
func Handle(h http.Handler) *Handler {
	return &Handler{h}
}

// HandleFunc returns a Handler wrapping an http.HandlerFunc.
func HandleFunc(f http.HandlerFunc) *Handler {
	return Handle(f)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || r.Header.Get("Sec-WebSocket-Key") != "" {
		h.Next.ServeHTTP(w, r)
		return
	}

	cw := &compressWriter{
		ResponseWriter: w,
		gziWriter:      gzipPool.Get().(*gzip.Writer),
	}
	defer gzipPool.Put(cw.gziWriter)
	defer cw.close()

	h.Next.ServeHTTP(cw, r)
}

// compressWriter binds the downstream repsonse writing into gziWriter if the first content is detected as gzip compressible.
// gzipUse keeps this detection result:
//	-1	detected but not used
// 	0	not detected yet
// 	1	detected and used
type compressWriter struct {
	http.ResponseWriter
	gziWriter *gzip.Writer
	gzipUse   int
	status    int
}

// WriteHeader catches a downstream WriteHeader call and caches the status code.
// The header will be written later, on the first Write call and after all the headers has been correctly set.
func (cw *compressWriter) WriteHeader(status int) {
	cw.status = status
}

// writePostponedHeader writes the response header with the cached status code.
func (cw *compressWriter) writePostponedHeader() {
	if cw.status == 0 {
		cw.status = http.StatusOK
	}
	cw.ResponseWriter.WriteHeader(cw.status)
}

// Write sets the compressing headers and calls the gzip writer, but only if the Content-Type header defines a compressible content.
// Otherwise, it calls the original Write method.
func (cw *compressWriter) Write(b []byte) (int, error) {
	if cw.gzipUse == 0 {
		ct := cw.ResponseWriter.Header().Get("Content-Type")
		if ct == "" {
			ct = http.DetectContentType(b)
			cw.ResponseWriter.Header().Set("Content-Type", ct)
		}

		cl, _ := strconv.Atoi(cw.ResponseWriter.Header().Get("Content-Length"))
		if cl < 1 {
			cl = len(b) // If no Content-Length, take the length of this first chunk.
		}

		if isGzippable(ct, cl) {
			cw.gzipUse = 1
			cw.setGzipHeaders()
			cw.gziWriter.Reset(cw.ResponseWriter)
		} else {
			cw.gzipUse = -1
		}

		cw.writePostponedHeader()
	}

	if cw.gzipUse == 1 {
		return cw.gziWriter.Write(b)
	}
	return cw.ResponseWriter.Write(b)
}

// close closes the gzip writer if it has been used.
func (cw *compressWriter) close() {
	if cw.gzipUse == 1 {
		cw.gziWriter.Close()
	}
}

// setGzipHeaders sets the Content-Encoding header for a gzip response.
// Because the compressed content will have a new size, it also removes the Content-Length header as it could have been set downstream by another handler.
func (cw *compressWriter) setGzipHeaders() {
	cw.ResponseWriter.Header().Del("Content-Length")
	cw.ResponseWriter.Header().Set("Content-Encoding", "gzip")
}

// isGzippable checks if a content must be compressed following its content length (cl) and content type (ct).
func isGzippable(ct string, cl int) bool {
	if cl < gzippableMinSize || ct == "" {
		return false
	}

	_, ok := notGzippableTypes[strings.ToLower(strings.SplitN(ct, ";", 2)[0])]
	return !ok
}
