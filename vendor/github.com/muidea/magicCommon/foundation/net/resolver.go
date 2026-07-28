package net

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http/httptrace"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type DNSResolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
	LookupAddr(ctx context.Context, addr string) (names []string, err error)
}

type Resolver struct {
	Timeout time.Duration

	Resolver DNSResolver

	once  sync.Once
	mu    sync.RWMutex
	cache map[string]*cacheEntry

	OnCacheMiss func()
}

type ResolverRefreshOptions struct {
	ClearUnused      bool
	PersistOnFailure bool
}

type cacheEntry struct {
	rrs  []string
	err  error
	used bool
}

func (r *Resolver) LookupAddr(ctx context.Context, addr string) (names []string, err error) {
	r.once.Do(r.init)
	return r.lookup(ctx, "r"+addr)
}

func (r *Resolver) LookupHost(ctx context.Context, host string) (addrs []string, err error) {
	r.once.Do(r.init)
	return r.lookup(ctx, "h"+host)
}

func (r *Resolver) refreshRecords(clearUnused bool, persistOnFailure bool) {
	r.once.Do(r.init)
	r.mu.RLock()
	update := make([]string, 0, len(r.cache))
	del := make([]string, 0, len(r.cache))
	for key, entry := range r.cache {
		if entry.used {
			update = append(update, key)
		} else if clearUnused {
			del = append(del, key)
		}
	}
	r.mu.RUnlock()

	if len(del) > 0 {
		r.mu.Lock()
		for _, key := range del {
			delete(r.cache, key)
		}
		r.mu.Unlock()
	}

	for _, key := range update {
		_, _ = r.update(context.Background(), key, false, persistOnFailure)
	}
}

func (r *Resolver) Refresh(clearUnused bool) {
	r.refreshRecords(clearUnused, false)
}

func (r *Resolver) RefreshWithOptions(options ResolverRefreshOptions) {
	r.refreshRecords(options.ClearUnused, options.PersistOnFailure)
}

func (r *Resolver) init() {
	r.cache = make(map[string]*cacheEntry)
}

var lookupGroup singleflight.Group

func (r *Resolver) lookup(ctx context.Context, key string) (rrs []string, err error) {
	var found bool
	rrs, err, found = r.load(key)
	if !found {
		if r.OnCacheMiss != nil {
			r.OnCacheMiss()
		}
		rrs, err = r.update(ctx, key, true, false)
	}
	return
}

func (r *Resolver) update(ctx context.Context, key string, used bool, persistOnFailure bool) (rrs []string, err error) {
	lookupFunc, err := r.lookupFunc(ctx, key)
	if err != nil {
		return nil, err
	}
	c := lookupGroup.DoChan(key, lookupFunc)
	select {
	case <-ctx.Done():
		err = ctx.Err()
		if errors.Is(err, context.DeadlineExceeded) {
			lookupGroup.Forget(key)
		}
	case res := <-c:
		if res.Shared {
			// We had concurrent lookups, check if the cache is already updated
			// by a friend.
			var found bool
			rrs, err, found = r.load(key)
			if found {
				return
			}
		}
		err = res.Err
		if err == nil {
			rrs, _ = res.Val.([]string)
		}

		if err != nil && persistOnFailure {
			var found bool
			rrs, err, found = r.load(key)
			if found {
				return
			}
		}

		r.mu.Lock()
		r.storeLocked(key, rrs, used, err)
		r.mu.Unlock()
	}
	return
}

func (r *Resolver) lookupFunc(ctx context.Context, key string) (func() (interface{}, error), error) {
	if len(key) == 0 {
		return nil, errors.New("lookupFunc with empty key")
	}

	var resolver DNSResolver = defaultResolver
	if r.Resolver != nil {
		resolver = r.Resolver
	}

	switch key[0] {
	case 'h':
		return func() (interface{}, error) {
			ctx, cancel := r.prepareCtx(ctx)
			defer cancel()

			return resolver.LookupHost(ctx, key[1:])
		}, nil
	case 'r':
		return func() (interface{}, error) {
			ctx, cancel := r.prepareCtx(ctx)
			defer cancel()

			return resolver.LookupAddr(ctx, key[1:])
		}, nil
	default:
		return nil, fmt.Errorf("lookupFunc invalid key type: %s", key)
	}
}

func (r *Resolver) prepareCtx(origContext context.Context) (ctx context.Context, cancel context.CancelFunc) {
	ctx = context.Background()
	if r.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
	} else {
		cancel = func() {
			// nothing
		}
	}

	if trace := httptrace.ContextClientTrace(origContext); trace != nil {
		derivedTrace := &httptrace.ClientTrace{
			DNSStart: trace.DNSStart,
			DNSDone:  trace.DNSDone,
		}

		ctx = httptrace.WithClientTrace(ctx, derivedTrace)
	}

	return
}

func (r *Resolver) load(key string) (rrs []string, err error, found bool) {
	r.mu.RLock()
	var entry *cacheEntry
	entry, found = r.cache[key]
	if !found {
		r.mu.RUnlock()
		return
	}
	rrs = entry.rrs
	err = entry.err
	used := entry.used
	r.mu.RUnlock()
	if !used {
		r.mu.Lock()
		entry.used = true
		r.mu.Unlock()
	}
	return rrs, err, true
}

func (r *Resolver) storeLocked(key string, rrs []string, used bool, err error) {
	if entry, found := r.cache[key]; found {
		// Update existing entry in place
		entry.rrs = rrs
		entry.err = err
		entry.used = used
		return
	}
	r.cache[key] = &cacheEntry{
		rrs:  rrs,
		err:  err,
		used: used,
	}
}

var defaultResolver = &defaultResolverWithTrace{
	ipVersion: "ip",
}

func NewResolverOnlyV4() DNSResolver {
	return &defaultResolverWithTrace{
		ipVersion: "ip4",
	}
}

func NewResolverOnlyV6() DNSResolver {
	return &defaultResolverWithTrace{
		ipVersion: "ip6",
	}
}

type defaultResolverWithTrace struct {
	ipVersion string
}

func (d *defaultResolverWithTrace) LookupHost(ctx context.Context, host string) (addrs []string, err error) {
	ipVersion := d.ipVersion
	if ipVersion != "ip" && ipVersion != "ip4" && ipVersion != "ip6" {
		ipVersion = "ip"
	}

	rawIPs, err := net.DefaultResolver.LookupIP(ctx, ipVersion, host)
	if err != nil {
		return nil, err
	}

	cookedIPs := make([]string, len(rawIPs))

	for i, v := range rawIPs {
		cookedIPs[i] = v.String()
	}

	return cookedIPs, nil
}

func (d *defaultResolverWithTrace) LookupAddr(ctx context.Context, addr string) (names []string, err error) {
	return net.DefaultResolver.LookupAddr(ctx, addr)
}
