package net

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

var resolver *Resolver
var initOnce sync.Once

const (
	defaultHTTPClientTimeout         = 15 * time.Second
	defaultHTTPDialTimeout           = 5 * time.Second
	defaultHTTPKeepAlive             = 30 * time.Second
	defaultHTTPResponseHeaderTimeout = 10 * time.Second
	defaultHTTPTLSHandshakeTimeout   = 5 * time.Second
	defaultHTTPExpectContinueTimeout = 1 * time.Second
	defaultHTTPIdleConnTimeout       = 90 * time.Second
)

func init() {
	initOnce.Do(func() {
		resolver = &Resolver{}

		options := ResolverRefreshOptions{}
		options.ClearUnused = true
		options.PersistOnFailure = false
		resolver.RefreshWithOptions(options)

		go func() {
			t := time.NewTicker(1 * time.Minute)
			defer t.Stop()
			for range t.C {
				resolver.Refresh(true)
			}
		}()
	})
}

func NewDNSCacheHttpClient() *http.Client {
	dialContext := func(ctx context.Context, network, addr string) (conn net.Conn, err error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := resolver.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			dialer := net.Dialer{
				Timeout:   defaultHTTPDialTimeout,
				KeepAlive: defaultHTTPKeepAlive,
			}
			conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
			if err == nil {
				break
			}
		}
		return
	}

	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{}
	}

	transport := baseTransport.Clone()
	transport.DialContext = dialContext
	transport.ResponseHeaderTimeout = defaultHTTPResponseHeaderTimeout
	transport.TLSHandshakeTimeout = defaultHTTPTLSHandshakeTimeout
	transport.ExpectContinueTimeout = defaultHTTPExpectContinueTimeout
	transport.IdleConnTimeout = defaultHTTPIdleConnTimeout
	return &http.Client{
		Timeout:   defaultHTTPClientTimeout,
		Transport: transport,
	}
}
