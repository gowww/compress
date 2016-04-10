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

// gzippableMinSize is the minimal size (in bytes) a content needs to have to be gzipped.
//
// A TCP packet is normally 1500 bytes long.
// So if the response plus the TCP headers already fits into a single packet, there will be no gain from gzip.
const gzippableMinSize = 1400

// notGzippableTypes is a custom list of media types referring to a compressed content.
// Gzip will not be applied to any of these content types.
//
// For performance, only the most common officials (and future officials) are listed.
//
// All official media types: http://www.iana.org/assignments/media-types/media-types.xhtml
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

var gzipPool = sync.Pool{New: func() interface{} { return gzip.NewWriter(nil) }}

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

// writePostponedHeader writes the response header when a cached status code exists.
func (cw *compressWriter) writePostponedHeader() {
	if cw.status > 0 {
		cw.ResponseWriter.WriteHeader(cw.status)
	}
}

// Write sets the compressing headers and calls the gzip writer, but only if the Content-Type header defines a compressible content.
// Otherwise, it calls the original Write method.
func (cw *compressWriter) Write(b []byte) (int, error) {
	if !cw.gzipDetect {
		// Check content is not already encoded.
		if cw.ResponseWriter.Header().Get("Content-Encoding") != "" {
			goto NoGzipUse
		}

		// Check content has sufficient length.
		if cl, _ := strconv.Atoi(cw.ResponseWriter.Header().Get("Content-Length")); cl <= 0 {
			// If no Content-Length, take the length of this first chunk.
			if len(b) < gzippableMinSize {
				goto NoGzipUse
			}
		}

		// Check content is of gzippable type.
		if ct := cw.ResponseWriter.Header().Get("Content-Type"); ct == "" {
			ct = http.DetectContentType(b)
			cw.ResponseWriter.Header().Set("Content-Type", ct)

			if i := strings.IndexByte(ct, ';'); i >= 0 {
				ct = ct[:i]
			}
			ct = strings.ToLower(ct)

			if _, ok := notGzippableTypes[ct]; ok {
				goto NoGzipUse
			}
		}

		cw.ResponseWriter.Header().Del("Content-Length") // Because the compressed content will have a new length.
		cw.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		cw.gzipWriter.Reset(cw.ResponseWriter)
		cw.gzipUse = true

	NoGzipUse:
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
	if !cw.gzipDetect {
		cw.writePostponedHeader()
	}

	if cw.gzipUse {
		cw.gzipWriter.Close()
	}
}
