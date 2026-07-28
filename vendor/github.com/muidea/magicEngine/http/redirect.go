package http

import (
	"context"
	"net/http"
)

type redirectRoute struct {
	uriPattern      string
	method          string
	redirectPattern string
}

func (s *redirectRoute) Method() string {
	return s.method
}

func (s *redirectRoute) Pattern() string {
	return s.uriPattern
}

func (s *redirectRoute) Handler() RouteHandleFunc {
	return func(ctx context.Context, res http.ResponseWriter, req *http.Request) {
		http.Redirect(res, req, s.redirectPattern, http.StatusSeeOther)
	}
}

func CreateRedirectRoute(uriPattern, method, redirectPattern string) Route {
	return &redirectRoute{uriPattern: uriPattern, method: method, redirectPattern: redirectPattern}
}
