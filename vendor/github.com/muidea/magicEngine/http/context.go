package http

import (
	"context"
	"net/http"
)

type RequestContext interface {
	Update(ctx context.Context)
	Context() context.Context
	Value(key any) any
	Next()
	Written() bool
	Run()
}

type baseContext struct {
	rw    ResponseWriter
	req   *http.Request
	index int
}

func (c *baseContext) Written() bool {
	return c.rw.Written()
}

func (c *baseContext) incrementIndex() {
	c.index++
}

func (c *baseContext) getIndex() int {
	return c.index
}

type requestContext struct {
	baseContext
	middlewareChainsFuncs []MiddleWareHandleFunc
	routeRegistry         RouteRegistry
	context               context.Context
}

// NewRequestContext 新建Context
func NewRequestContext(middlewareChains []MiddleWareHandleFunc, routeRegistry RouteRegistry, ctx context.Context, res http.ResponseWriter, req *http.Request) RequestContext {
	return &requestContext{
		baseContext:           baseContext{rw: NewResponseWriter(res), req: req, index: 0},
		middlewareChainsFuncs: middlewareChains,
		routeRegistry:         routeRegistry,
		context:               ctx,
	}
}

func (c *requestContext) Update(ctx context.Context) {
	c.context = ctx
}

func (c *requestContext) Context() context.Context {
	return c.context
}

func (c *requestContext) Value(key any) any {
	return c.context.Value(key)
}

func (c *requestContext) Next() {
	c.incrementIndex()
	c.Run()
}

func (c *requestContext) Written() bool {
	return c.baseContext.Written()
}

func (c *requestContext) Run() {
	totalSize := len(c.middlewareChainsFuncs)
	for c.baseContext.index < totalSize {
		c.middlewareChainsFuncs[c.baseContext.index](c, c.baseContext.rw, c.baseContext.req)

		c.baseContext.index++
		if c.Written() {
			return
		}
	}

	if !c.Written() && c.routeRegistry != nil {
		c.routeRegistry.Handle(c.Context(), c.baseContext.rw.(http.ResponseWriter), c.baseContext.req)
		if !c.Written() {
			http.Error(c.baseContext.rw, "", http.StatusNoContent)
		}
	} else {
		http.NotFound(c.baseContext.rw, c.baseContext.req)
	}
}

type routeContext struct {
	baseContext
	middlewareChainsHandler []MiddleWareHandler
	route                   Route
	context                 context.Context
}

// NewRouteContext 新建Context
func NewRouteContext(reqCtx context.Context, chainsHandler []MiddleWareHandler, route Route, res http.ResponseWriter, req *http.Request) RequestContext {
	return &routeContext{
		baseContext:             baseContext{rw: res.(ResponseWriter), req: req, index: 0},
		middlewareChainsHandler: chainsHandler,
		route:                   route,
		context:                 reqCtx,
	}
}

func (c *routeContext) Update(ctx context.Context) {
	c.context = ctx
}

func (c *routeContext) Context() context.Context {
	return c.context
}

func (c *routeContext) Value(key any) any {
	return c.context.Value(key)
}

func (c *routeContext) Next() {
	c.incrementIndex()
	c.Run()
}

func (c *routeContext) Written() bool {
	return c.baseContext.Written()
}

func (c *routeContext) Run() {
	totalSize := len(c.middlewareChainsHandler)
	for c.baseContext.index < totalSize {
		c.middlewareChainsHandler[c.baseContext.index].MiddleWareHandle(c, c.baseContext.rw, c.baseContext.req)
		c.baseContext.index++
		if c.Written() {
			return
		}
	}

	if !c.Written() {
		funHandle := c.route.Handler()
		funHandle(c.Context(), c.baseContext.rw, c.baseContext.req)
	}

	if !c.Written() {
		http.Error(c.baseContext.rw, "", http.StatusNoContent)
	}
}
