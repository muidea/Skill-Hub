package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// 基本HTTP行为定义
const (
	HEAD    = "HEAD"
	GET     = "GET"
	POST    = "POST"
	PUT     = "PUT"
	DELETE  = "DELETE"
	OPTIONS = "OPTIONS"
)

const (
	DynamicTag           = "X-Mp-Engine-Dynamic-Tag"
	DynamicValue         = "X-Mp-Engine-Dynamic-Value"
	DynamicRawUriPattern = "X-Mp-Engine-Dynamic-Raw-Uri-Pattern"
)

type RawUriPattern struct{}

// Route 路由接口
type Route interface {
	// Method 路由行为GET/PUT/POST/DELETE
	Method() string
	// Pattern uri路由规则, 以'/'开始
	Pattern() string
	// Handler 路由处理器
	Handler() RouteHandleFunc
}

// RouteRegistry 路由器对象
type RouteRegistry interface {
	// SetApiVersion 设置ApiVersion
	SetApiVersion(version string)
	// GetApiVersion 查询ApiVersion
	GetApiVersion() string
	// AddRoute 增加路由
	AddRoute(rt Route, filters ...MiddleWareHandler)
	// RemoveRoute 清除路由
	RemoveRoute(rt Route)
	// AddHandler 增加Handler
	AddHandler(uriPattern, method string, handler RouteHandleFunc, filters ...MiddleWareHandleFunc)
	// RemoveHandler 清除Handler
	RemoveHandler(uriPattern, method string)
	// Handle 分发一条请求
	Handle(ctx context.Context, res http.ResponseWriter, req *http.Request)
	// ExistRoute 路由是否存在
	ExistRoute(rt Route) bool
	// ExistHandler Handler是否存在
	ExistHandler(uriPattern, method string) bool
}

type rtItem struct {
	uriPattern string
	method     string
	handler    RouteHandleFunc
}

func (s *rtItem) Pattern() string {
	return s.uriPattern
}

func (s *rtItem) Method() string {
	return s.method
}

func (s *rtItem) Handler() RouteHandleFunc {
	return s.handler
}

// CreateRoute create Route
func CreateRoute(uriPattern, method string, handler func(context.Context, http.ResponseWriter, *http.Request)) Route {
	return &rtItem{uriPattern: uriPattern, method: method, handler: handler}
}

// PatternFilter route filter
type PatternFilter struct {
	regex *regexp.Regexp
}

var routeReg1 = regexp.MustCompile(`:[^/#?()\.\\]+`)
var routeReg2 = regexp.MustCompile(`\*\*`)

// NewPatternFilter new route filter
func NewPatternFilter(routeUriPattern string) *PatternFilter {
	filter := &PatternFilter{}
	pattern := routeReg1.ReplaceAllStringFunc(routeUriPattern, func(m string) string {
		return fmt.Sprintf(`(?P<%s>[^/#?]+)`, m[1:])
	})
	var index int
	pattern = routeReg2.ReplaceAllStringFunc(pattern, func(m string) string {
		index++
		return fmt.Sprintf(`(?P<_%d>[^#?]*)`, index)
	})
	pattern += `\/?`
	filter.regex = regexp.MustCompile(pattern)

	return filter
}

func (s *PatternFilter) Match(uriPath string) bool {
	matches := s.regex.FindStringSubmatch(uriPath)
	if len(matches) > 0 && matches[0] == uriPath {
		return true
	}

	return false
}

// 路由对象
type routeItem struct {
	versionPrefix  string
	route          Route
	middlewareList []MiddleWareHandler
	patternFilter  *PatternFilter
}

func (s *routeItem) equalRoute(versionPrefix string, rt Route) bool {
	if s.versionPrefix != versionPrefix {
		return false
	}

	return s.route.Pattern() == rt.Pattern()
}

func (s *routeItem) equalPattern(versionPrefix string, uriPattern string) bool {
	if s.versionPrefix != versionPrefix {
		return false
	}

	return s.route.Pattern() == uriPattern
}

func (s *routeItem) match(uriPath string) bool {
	return s.patternFilter.Match(uriPath)
}

func newRouteItem(versionPrefix string, rt Route, filters ...MiddleWareHandler) *routeItem {
	item := &routeItem{versionPrefix: versionPrefix, route: rt}
	item.middlewareList = append(item.middlewareList, filters...)
	rtPattern := rt.Pattern()
	if versionPrefix != "" {
		rtPattern = fmt.Sprintf("%s%s", versionPrefix, rtPattern)
	}
	item.patternFilter = NewPatternFilter(rtPattern)

	// log.Infof("[%s]:%s", rt.Method(), rtPattern)

	return item
}

type routeItemSlice []*routeItem

type routeRegistry struct {
	currentApiVersion string
	routes            map[string]*routeItemSlice
	routesLock        sync.RWMutex
}

// NewRouteRegistry 新建Route registry
func NewRouteRegistry() RouteRegistry {
	return &routeRegistry{routes: make(map[string]*routeItemSlice)}
}

func (s *routeRegistry) SetApiVersion(version string) {
	s.currentApiVersion = version
}

func (s *routeRegistry) GetApiVersion() string {
	return s.currentApiVersion
}

func (s *routeRegistry) AddRoute(rt Route, filters ...MiddleWareHandler) {
	curApiVersion := s.currentApiVersion

	slog.Info("addRoute", "apiVersion", s.currentApiVersion, "pattern", rt.Pattern(), "method", rt.Method())
	s.routesLock.Lock()
	defer s.routesLock.Unlock()

	routeSlice, ok := s.routes[rt.Method()]
	if ok {
		s.checkDuplicateRoute(routeSlice, curApiVersion, rt)
		item := newRouteItem(curApiVersion, rt, filters...)
		*routeSlice = append(*routeSlice, item)
		return
	}

	item := newRouteItem(curApiVersion, rt, filters...)
	routeSlice = &routeItemSlice{}
	*routeSlice = append(*routeSlice, item)
	s.routes[rt.Method()] = routeSlice
}

func (s *routeRegistry) checkDuplicateRoute(routeSlice *routeItemSlice, curApiVersion string, rt Route) {
	for _, val := range *routeSlice {
		if val.equalRoute(curApiVersion, rt) {
			msg := fmt.Sprintf("duplicate route!, apiVersion:%s, pattern:%s, method:%s", curApiVersion, rt.Pattern(), rt.Method())
			panicInfo(msg)
		}
	}
}

func (s *routeRegistry) RemoveRoute(rt Route) {
	s.removeRouteImpl(rt.Pattern(), rt.Method())
}

func (s *routeRegistry) AddHandler(uriPattern, method string,
	handler RouteHandleFunc,
	filters ...MiddleWareHandleFunc) {
	rt := CreateRoute(uriPattern, method, handler)

	middleWareList := make([]MiddleWareHandler, len(filters))
	for idx := range filters {
		middleWareList[idx] = &anonymousMiddleWareHandler{
			handleFunc: filters[idx],
		}
	}

	s.AddRoute(rt, middleWareList...)
}

func (s *routeRegistry) RemoveHandler(uriPattern, method string) {
	s.removeRouteImpl(uriPattern, method)
}

func (s *routeRegistry) removeRouteImpl(uriPattern, method string) {
	curApiVersion := s.currentApiVersion
	slog.Info("removeRoute", "apiVersion", s.currentApiVersion, "pattern", uriPattern, "method", method)

	s.routesLock.Lock()
	defer s.routesLock.Unlock()

	routeSlice, ok := s.routes[method]
	if !ok {
		msg := fmt.Sprintf("no found route!, pattern:%s, method:%s", uriPattern, method)
		panicInfo(msg)
	}

	newRoutes := routeItemSlice{}
	for idx, val := range *routeSlice {
		if val.equalPattern(curApiVersion, uriPattern) {
			if idx > 0 {
				newRoutes = append(newRoutes, (*routeSlice)[0:idx]...)
			}

			idx++
			if idx < len(*routeSlice) {
				newRoutes = append(newRoutes, (*routeSlice)[idx:]...)
			}

			break
		}
	}

	s.routes[method] = &newRoutes
}

func (s *routeRegistry) Handle(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	var routeSlice routeItemSlice
	func() {
		s.routesLock.RLock()
		defer s.routesLock.RUnlock()

		slice, ok := s.routes[strings.ToUpper(req.Method)]
		if ok {
			routeSlice = (*slice)[:]
		}
	}()

	// set default content-type = "application/json; charset=utf-8"
	//res.Header().Set("Content-Type", "application/json; charset=utf-8")
	var routeCtx RequestContext
	for _, val := range routeSlice {
		if val.match(req.URL.Path) {
			routeCtx = NewRouteContext(ctx, val.middlewareList, val.route, res, req)
			break
		}
	}

	if routeCtx != nil {
		routeCtx.Run()
		return
	}

	http.NotFound(res, req)
	//http.Redirect(res, req, "/404.html", http.StatusMovedPermanently)
}

func (s *routeRegistry) ExistRoute(rt Route) bool {
	return s.ExistHandler(rt.Pattern(), rt.Method())
}

func (s *routeRegistry) ExistHandler(uriPattern, method string) bool {
	s.routesLock.Lock()
	defer s.routesLock.Unlock()

	routeSlice, ok := s.routes[method]
	if !ok {
		return false
	}

	return s.findRoute(routeSlice, s.currentApiVersion, uriPattern) != nil
}

func (s *routeRegistry) findRoute(routeSlice *routeItemSlice, curApiVersion string, uriPattern string) *routeItem {
	for _, val := range *routeSlice {
		if val.equalPattern(curApiVersion, uriPattern) {
			return val
		}
	}
	return nil
}
