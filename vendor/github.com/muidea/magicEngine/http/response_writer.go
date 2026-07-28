package http

import (
	"net/http"
)

type ResponseWriter interface {
	http.ResponseWriter
	Status() int
	Written() bool
	Size() int
}

func NewResponseWriter(rw http.ResponseWriter) ResponseWriter {
	return &responseWriter{responseWriter: rw}
}

type responseWriter struct {
	responseWriter http.ResponseWriter
	status         int
	size           int
}

func (rw *responseWriter) Header() http.Header {
	return rw.responseWriter.Header()
}

func (rw *responseWriter) WriteHeader(s int) {
	rw.responseWriter.WriteHeader(s)
	rw.status = s
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.Written() {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.responseWriter.Write(b)
	rw.size += size
	return size, err
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) Size() int {
	return rw.size
}

func (rw *responseWriter) Written() bool {
	return rw.status != 0
}

func (rw *responseWriter) Flush() {
	rw.responseWriter.(http.Flusher).Flush()
}
