package http

import (
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

type logger struct {
	serialNo int64
}

func (s *logger) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	start := time.Now()

	addr := req.Header.Get("X-Real-IP")
	if addr == "" {
		addr = req.Header.Get("X-Forwarded-For")
		if addr == "" {
			addr = req.RemoteAddr
		}
	}

	curSerial := atomic.AddInt64(&s.serialNo, 1)
	if EnableTrace() {
		slog.Info("request started", "serial", curSerial, "method", req.Method, "path", req.URL.Path, "addr", addr)
	}

	rw := res.(ResponseWriter)
	ctx.Next()

	elapseVal := time.Since(start)
	if EnableTrace() {
		slog.Info("request completed", "serial", curSerial, "status", rw.Status(), "status_text", http.StatusText(rw.Status()), "elapsed", elapseVal)
	} else if elapseVal >= GetElapseThreshold() {
		slog.Warn("slow request", "serial", curSerial, "method", req.Method, "path", req.URL.Path, "addr", addr, "status", rw.Status(), "elapsed", elapseVal)
	}
}
