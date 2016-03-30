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

const gzippableMinSize = 1400

var gzipPool = sync.Pool{New: func() interface{} { return gzip.NewWriter(nil) }}

var notGzippableTypes = map[string]struct{}{
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
	w.Header().Add("Vary", "Accept-Encoding")

	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || r.Header.Get("Sec-WebSocket-Key") != "" {
		h.Next.ServeHTTP(w, r)
		return
	}

	cw := &compressWriter{
		ResponseWriter: w,
		gzipWriter:     gzipPool.Get().(*gzip.Writer),
	}
	defer gzipPool.Put(cw.gzipWriter)
	defer cw.close()

	h.Next.ServeHTTP(cw, r)
}

// compressWriter binds the downstream repsonse writing into gzipWriter if the first content is detected as gzippable.
type compressWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
	gzipDetect bool // gzipDetect tells if the gzippable detection has been done.
	gzipUse    bool // gzipUse tells if gzip is used for the response.
	status     int
}

// WriteHeader catches a downstream WriteHeader call and caches the status code.
// The header will be written later, at the first Write call, after the gzipping detection has been done.
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
	if !cw.gzipDetect {
		cl, _ := strconv.Atoi(cw.ResponseWriter.Header().Get("Content-Length"))
		if cl < 1 {
			cl = len(b) // If no Content-Length, take the length of this first chunk.
		}

		ct := cw.ResponseWriter.Header().Get("Content-Type")
		if ct == "" {
			ct = http.DetectContentType(b)
			cw.ResponseWriter.Header().Set("Content-Type", ct)
		}

		if isGzippable(cl, ct) {
			cw.gzipUse = true
			cw.setGzipHeaders()
			cw.gzipWriter.Reset(cw.ResponseWriter)
		}

		cw.writePostponedHeader()
		cw.gzipDetect = true
	}

	if cw.gzipUse {
		return cw.gzipWriter.Write(b)
	}
	return cw.ResponseWriter.Write(b)
}

// close closes the gzip writer if it has been used.
func (cw *compressWriter) close() {
	if cw.gzipUse {
		cw.gzipWriter.Close()
	}
}

// setGzipHeaders sets the Content-Encoding header for a gzip response.
// Because the compressed content will have a new size, it also removes the Content-Length header as it could have been set downstream by another handler.
func (cw *compressWriter) setGzipHeaders() {
	cw.ResponseWriter.Header().Del("Content-Length")
	cw.ResponseWriter.Header().Set("Content-Encoding", "gzip")
}

// isGzippable checks if a content must be compressed following its content length (cl) and content type (ct).
func isGzippable(cl int, ct string) bool {
	if cl < gzippableMinSize || ct == "" {
		return false
	}

	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	ct = strings.ToLower(ct)
	_, ok := notGzippableTypes[ct]
	return !ok
}
