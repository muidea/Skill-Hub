package http

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type ProxyErrorHandler func(http.ResponseWriter, *http.Request, error)

// newReverseProxy 创建一个新的反向代理，将请求转发到指定的目标URL
// 它会合并目标URL和传入请求中的查询参数
func newReverseProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery, _ := url.ParseQuery(target.RawQuery)
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		reqQuery, _ := url.ParseQuery(req.URL.RawQuery)
		for k, v := range targetQuery {
			reqQuery.Set(k, v[0])
		}
		req.URL.RawQuery = reqQuery.Encode()
	}
	return &httputil.ReverseProxy{Director: director}
}

// proxyRoute 表示一个代理路由，用于将请求转发到目标URL
type proxyRoute struct {
	uriPattern string
	method     string
	targetURL  string
	rewriteURL bool
	targetURI  *url.URL
	proxy      *httputil.ReverseProxy
	parseErr   error
}

// Pattern 返回此代理路由的URI模式
func (s *proxyRoute) Pattern() string {
	return s.uriPattern
}

// Method 返回此代理路由的HTTP方法
func (s *proxyRoute) Method() string {
	return s.method
}

// Handler 返回路由处理函数
func (s *proxyRoute) Handler() RouteHandleFunc {
	return s.proxyFun
}

func cloneQueryValues(values url.Values) url.Values {
	if values == nil {
		return url.Values{}
	}

	ret := make(url.Values, len(values))
	for key, items := range values {
		ret[key] = append([]string(nil), items...)
	}
	return ret
}

func applyProxyTarget(targetURI *url.URL, req *http.Request) *url.URL {
	target := &url.URL{}
	if targetURI != nil {
		*target = *targetURI
	}

	targetQuery := cloneQueryValues(target.Query())
	reqQuery := req.URL.Query()

	dynamicTAG := req.Header.Get(DynamicTag)
	dynamicValue := req.Header.Get(DynamicValue)
	if dynamicTAG != "" && dynamicValue != "" {
		target.Path = strings.ReplaceAll(target.Path, dynamicTAG, dynamicValue)
	}

	for k, v := range reqQuery {
		targetQuery.Set(k, v[0])
	}
	target.RawQuery = targetQuery.Encode()
	return target
}

func (s *proxyRoute) applyTarget(req *http.Request) *url.URL {
	return applyProxyTarget(s.targetURI, req)
}

// ProxyHTTP proxies the current request to a dynamically resolved target URL.
func ProxyHTTP(res http.ResponseWriter, req *http.Request, targetURL string, errorHandler ProxyErrorHandler) error {
	targetURI, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	target := applyProxyTarget(targetURI, req)
	if target.Hostname() == "" {
		http.Redirect(res, req, target.String(), http.StatusSeeOther)
		return nil
	}

	proxy := &httputil.ReverseProxy{
		Director: func(proxyReq *http.Request) {
			proxyReq.URL.Scheme = target.Scheme
			proxyReq.URL.Host = target.Host
			proxyReq.Host = target.Host
			proxyReq.URL.Path = target.Path
			proxyReq.URL.RawQuery = target.RawQuery
		},
	}
	proxy.ErrorHandler = func(res http.ResponseWriter, req *http.Request, err error) {
		if errorHandler != nil {
			errorHandler(res, req, err)
			return
		}
		res.WriteHeader(http.StatusInternalServerError)
		_, _ = res.Write([]byte(err.Error()))
	}
	proxy.ServeHTTP(res, req)
	return nil
}

// proxyFun 是实际处理请求转发的函数
func (s *proxyRoute) proxyFun(_ context.Context, res http.ResponseWriter, req *http.Request) {
	if s.parseErr != nil {
		slog.Error("illegal proxy target URL", "url", s.targetURL, "err", ErrInvalidProxyTarget)
		return
	}

	targetUri := s.applyTarget(req)

	// 如果目标URL没有主机名，则执行重定向
	if targetUri.Hostname() == "" {
		http.Redirect(res, req, targetUri.String(), http.StatusSeeOther)
		return
	}

	// errorHandler 处理代理转发过程中的错误
	errorHandler := func(res http.ResponseWriter, req *http.Request, err error) {
		res.WriteHeader(http.StatusInternalServerError)
		_, _ = res.Write([]byte(err.Error()))
	}

	if s.proxy == nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.proxy.ErrorHandler = errorHandler
	s.proxy.ServeHTTP(res, req)
}

// CreateProxyRoute 创建代理路由
func CreateProxyRoute(uriPattern, method, targetURL string, rewriteURL bool) Route {
	route := &proxyRoute{uriPattern: uriPattern, method: method, targetURL: targetURL, rewriteURL: rewriteURL}
	targetURI, err := url.Parse(targetURL)
	if err != nil {
		route.parseErr = err
		return route
	}
	route.targetURI = targetURI
	if targetURI.Hostname() == "" {
		return route
	}

	if rewriteURL {
		route.proxy = &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				target := route.applyTarget(req)
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.URL.Path = target.Path
				req.URL.RawQuery = target.RawQuery
			},
		}
		return route
	}

	route.proxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			target := route.applyTarget(req)
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = target.Path
			req.URL.RawQuery = target.RawQuery
		},
	}
	return route
}
