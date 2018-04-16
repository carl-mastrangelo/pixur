package handlers

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

var _ http.Handler = &compressionHandler{}

type compressionHandler struct {
	next http.Handler
}

func (h *compressionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var gw io.Writer
	for _, enc := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		if strings.TrimSpace(enc) == "gzip" {
			var err error
			gw, err = gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				httpError(w, err)
				return
			}
			break
		}
	}
	crw := &compressingResponseWriter{delegate: w, writer: gw}
	defer crw.Close()
	h.next.ServeHTTP(crw, r)
}

var _ http.ResponseWriter = &compressingResponseWriter{}
var _ http.Flusher = &compressingResponseWriter{}
var _ http.Pusher = &compressingResponseWriter{}

type compressingResponseWriter struct {
	delegate http.ResponseWriter
	writer   io.Writer
	whcalled bool
}

func (rw *compressingResponseWriter) Header() http.Header {
	return rw.delegate.Header()
}

func (rw *compressingResponseWriter) Write(data []byte) (int, error) {
	if !rw.whcalled {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.writer.Write(data)
}

/*

	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "text/html; charset=utf-8")
	}
*/

func (rw *compressingResponseWriter) WriteHeader(code int) {
	if !rw.whcalled {
		rw.whcalled = true
		header := rw.Header()
		if header.Get("Content-Encoding") == "" && rw.writer != nil {
			header.Set("Content-Encoding", "gzip")
		} else {
			rw.writer = rw.delegate
		}

	}
	rw.delegate.WriteHeader(code)
}

func (rw *compressingResponseWriter) Flush() {
	type errflusher interface {
		Flush() error
	}
	switch f := rw.writer.(type) {
	case http.Flusher:
		f.Flush()
	case errflusher:
		if err := f.Flush(); err != nil {
			httpError(rw, err)
			return
		}
	}
}

func (rw *compressingResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := rw.delegate.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// Close should be called after the ServeHTTP method returns, per net/http:
// "A ResponseWriter may not be used after the Handler.ServeHTTP method has returned."
func (rw *compressingResponseWriter) Close() error {
	if closer, ok := rw.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
