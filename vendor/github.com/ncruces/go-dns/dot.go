package dns

import (
	"context"
	"crypto/tls"
	"net"
	"sync/atomic"
)

// NewDoTResolver creates a DNS over TLS resolver.
// The server can be an IP address, a host name, or a network address of the form "host:port".
func NewDoTResolver(server string, options ...DoTOption) (*net.Resolver, error) {
	// look for a custom port
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		port = "853"
	} else {
		server = host
	}

	// apply options
	var opts dotOpts
	for _, o := range options {
		o.apply(&opts)
	}

	// resolve server network addresses
	if len(opts.addrs) == 0 {
		ips, err := OpportunisticResolver.LookupIPAddr(context.Background(), server)
		if err != nil {
			return nil, err
		}
		opts.addrs = make([]string, len(ips))
		for i, ip := range ips {
			opts.addrs[i] = net.JoinHostPort(ip.String(), port)
		}
	} else {
		for i, a := range opts.addrs {
			if net.ParseIP(a) != nil {
				opts.addrs[i] = net.JoinHostPort(a, port)
			}
		}
	}

	// setup TLS config
	if opts.config == nil {
		opts.config = &tls.Config{
			ClientSessionCache: tls.NewLRUClientSessionCache(len(opts.addrs)),
		}
	} else {
		opts.config = opts.config.Clone()
	}
	if opts.config.ServerName == "" {
		opts.config.ServerName = server
	}

	// setup the dialFunc
	if opts.dialFunc == nil {
		var d net.Dialer
		opts.dialFunc = d.DialContext
	}

	// create the resolver
	var resolver = net.Resolver{PreferGo: true}

	// setup dialer
	var index atomic.Uint32
	resolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		i := index.Load()
		conn, err := opts.dialFunc(ctx, "tcp", opts.addrs[i])
		if err != nil {
			index.CompareAndSwap(i, (i+1)%uint32(len(opts.addrs)))
			return nil, err
		}
		return tls.Client(conn, opts.config), nil
	}

	// setup caching
	if opts.cache {
		resolver.Dial = NewCachingDialer(resolver.Dial, opts.cacheOpts...)
	}

	return &resolver, nil
}

// A DoTOption customizes the DNS over TLS resolver.
type DoTOption interface {
	apply(*dotOpts)
}

type dotOpts struct {
	config    *tls.Config
	addrs     []string
	cache     bool
	cacheOpts []CacheOption
	dialFunc  DialFunc
}

type (
	dotConfig    tls.Config
	dotAddresses []string
	dotCache     []CacheOption
	dotDialFunc  DialFunc
)

func (o *dotConfig) apply(t *dotOpts)   { t.config = (*tls.Config)(o) }
func (o dotAddresses) apply(t *dotOpts) { t.addrs = ([]string)(o) }
func (o dotCache) apply(t *dotOpts)     { t.cache = true; t.cacheOpts = ([]CacheOption)(o) }
func (o dotDialFunc) apply(t *dotOpts)  { t.dialFunc = (DialFunc)(o) }

// DoTConfig sets the tls.Config used by the resolver.
func DoTConfig(config *tls.Config) DoTOption { return (*dotConfig)(config) }

// DoTAddresses sets the network addresses of the resolver.
// These should be IP addresses, or network addresses of the form "IP:port".
// This avoids having to resolve the resolver's addresses, improving performance and privacy.
func DoTAddresses(addresses ...string) DoTOption { return dotAddresses(addresses) }

// DoTCache adds caching to the resolver, with the given options.
func DoTCache(options ...CacheOption) DoTOption { return dotCache(options) }

// DoTDialFunc sets the DialFunc used by the resolver.
// By default [net.Dialer.DialContext] is used.
func DoTDialFunc(f DialFunc) DoTOption { return dotDialFunc(f) }
