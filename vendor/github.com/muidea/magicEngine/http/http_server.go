package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

type HTTPServer interface {
	Use(handler MiddleWareHandler)
	Bind(routeRegistry RouteRegistry)
	Run()
}

type HTTPServerOption func(*httpServer)

func WithPort(port string) HTTPServerOption {
	return func(s *httpServer) {
		s.listenAddr = fmt.Sprintf(":%s", port)
	}
}

func WithStatic(rootPath, prefixUri, excludeUri string) HTTPServerOption {
	return func(s *httpServer) {
		s.staticOptions = &StaticOptions{
			RootPath:   rootPath,
			PrefixUri:  prefixUri,
			ExcludeUri: excludeUri,
		}
	}
}

func WithStaticEnabled(enabled bool) HTTPServerOption {
	return func(s *httpServer) {
		s.enableStatic = enabled
	}
}

type httpServer struct {
	listenAddr       string
	routeRegistry    RouteRegistry
	middlewareChains MiddleWareChains
	staticOptions    *StaticOptions
	enableStatic     bool
}

func NewHTTPServer(opts ...HTTPServerOption) HTTPServer {
	svr := &httpServer{
		listenAddr:       ":8080",
		middlewareChains: NewMiddleWareChains(),
		staticOptions:    &StaticOptions{RootPath: "./static", PrefixUri: "/static", ExcludeUri: "/api/"},
		enableStatic:     false,
	}

	for _, opt := range opts {
		opt(svr)
	}

	svr.Use(&logger{})
	svr.Use(&recovery{})

	if svr.enableStatic {
		svr.Use(&static{rootPath: Root})
	}

	return svr
}

func (s *httpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	httpContext := context.WithValue(req.Context(), StaticOptionsKey{}, s.staticOptions)
	ctx := NewRequestContext(s.middlewareChains.GetHandlers(), s.routeRegistry, httpContext, res, req)

	ctx.Run()
}

func (s *httpServer) Use(handler MiddleWareHandler) {
	s.middlewareChains.Append(handler)
}

func (s *httpServer) Bind(routeRegistry RouteRegistry) {
	s.routeRegistry = routeRegistry
}

func (s *httpServer) Run() {
	slog.Info("server listening", "addr", s.listenAddr)
	err := http.ListenAndServe(s.listenAddr, s)
	slog.Error("server fatal error", "err", err)
}
