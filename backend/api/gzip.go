package api

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipped compresses the response when the client accepts gzip. It is meant
// for large responses only, such as a game result with every player's track.
func gzipped(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding")
		if !acceptsGzip(r.Header.Get("Accept-Encoding")) {
			next(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(w)
		defer writer.Close()
		next(gzipResponseWriter{ResponseWriter: w, body: writer}, r)
	}
}

// acceptsGzip reports whether an Accept-Encoding header allows gzip, which "gzip;q=0" forbids.
func acceptsGzip(header string) bool {
	for _, part := range strings.Split(header, ",") {
		coding, params, _ := strings.Cut(part, ";")
		if strings.EqualFold(strings.TrimSpace(coding), "gzip") {
			q := strings.ReplaceAll(strings.TrimSpace(params), " ", "")
			return q != "q=0" && q != "q=0.0" && q != "q=0.00" && q != "q=0.000"
		}
	}
	return false
}

type gzipResponseWriter struct {
	http.ResponseWriter
	body io.Writer
}

func (w gzipResponseWriter) Write(data []byte) (int, error) { return w.body.Write(data) }
