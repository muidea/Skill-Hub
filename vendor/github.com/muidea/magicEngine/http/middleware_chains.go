package http

import (
	"context"
	"net/http"
	"sync"
)

type RouteHandleFunc func(context.Context, http.ResponseWriter, *http.Request)
type MiddleWareHandleFunc = func(RequestContext, http.ResponseWriter, *http.Request)

// MiddleWareHandler 中间件处理器
type MiddleWareHandler interface {
	MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request)
}

type anonymousMiddleWareHandler struct {
	handleFunc MiddleWareHandleFunc
}

func (s *anonymousMiddleWareHandler) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	s.handleFunc(ctx, res, req)
}

// MiddleWareChains 处理器链
type MiddleWareChains interface {
	Append(handler MiddleWareHandler)

	GetHandlers() []MiddleWareHandleFunc
}

type chainsImpl struct {
	handlers    []MiddleWareHandleFunc
	handlesLock sync.RWMutex
}

// NewMiddleWareChains 新建MiddleWareChains
func NewMiddleWareChains() MiddleWareChains {
	return &chainsImpl{handlers: []MiddleWareHandleFunc{}}
}

func (s *chainsImpl) GetHandlers() []MiddleWareHandleFunc {
	s.handlesLock.RLock()
	defer s.handlesLock.RUnlock()

	return s.handlers[:]
}

func (s *chainsImpl) Append(handler MiddleWareHandler) {
	s.handlesLock.Lock()
	defer s.handlesLock.Unlock()

	s.handlers = append(s.handlers, handler.MiddleWareHandle)
}
