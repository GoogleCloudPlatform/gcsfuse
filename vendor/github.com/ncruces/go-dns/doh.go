package dns

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

// NewDoHResolver creates a DNS over HTTPS resolver.
// The uri may be an URI Template.
func NewDoHResolver(uri string, options ...DoHOption) (*net.Resolver, error) {
	// parse the uri template into a url
	uri, err := parseURITemplate(uri)
	if err != nil {
		return nil, err
	}
	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	port := url.Port()
	if port == "" {
		port = url.Scheme
	}

	// apply options
	var opts dohOpts
	for _, o := range options {
		o.apply(&opts)
	}

	// resolve server network addresses
	if len(opts.addrs) == 0 {
		ips, err := OpportunisticResolver.LookupIPAddr(context.Background(), url.Hostname())
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

	// setup the http transport
	if opts.transport == nil {
		opts.transport = &http.Transport{
			MaxIdleConns:        http.DefaultMaxIdleConnsPerHost,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			ForceAttemptHTTP2:   true,
		}
	} else {
		opts.transport = opts.transport.Clone()
	}

	// setup the http client
	client := http.Client{
		Transport: opts.transport,
	}

	// create the resolver
	var resolver = net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			conn := &dnsConn{}
			conn.roundTrip = dohRoundTrip(uri, &client)
			return conn, nil
		},
	}

	// setup dialer
	var index atomic.Uint32
	opts.transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		var d net.Dialer
		i := index.Load()
		conn, err := d.DialContext(ctx, network, opts.addrs[i])
		if err != nil {
			index.CompareAndSwap(i, (i+1)%uint32(len(opts.addrs)))
		}
		return conn, err
	}

	// setup caching
	if opts.cache {
		resolver.Dial = NewCachingDialer(resolver.Dial, opts.cacheOpts...)
	}

	return &resolver, nil
}

// A DoHOption customizes the DNS over HTTPS resolver.
type DoHOption interface {
	apply(*dohOpts)
}

type dohOpts struct {
	transport *http.Transport
	addrs     []string
	cache     bool
	cacheOpts []CacheOption
}

type (
	dohTransport http.Transport
	dohAddresses []string
	dohCache     []CacheOption
)

func (o *dohTransport) apply(t *dohOpts) { t.transport = (*http.Transport)(o) }
func (o dohAddresses) apply(t *dohOpts)  { t.addrs = ([]string)(o) }
func (o dohCache) apply(t *dohOpts)      { t.cache = true; t.cacheOpts = ([]CacheOption)(o) }

// DoHTransport sets the http.Transport used by the resolver.
func DoHTransport(transport *http.Transport) DoHOption { return (*dohTransport)(transport) }

// DoHAddresses sets the network addresses of the resolver.
// These should be IP addresses, or network addresses of the form "IP:port".
// This avoids having to resolve the resolver's addresses, improving performance and privacy.
func DoHAddresses(addresses ...string) DoHOption { return dohAddresses(addresses) }

// DoHCache adds caching to the resolver, with the given options.
func DoHCache(options ...CacheOption) DoHOption { return dohCache(options) }

func dohRoundTrip(uri string, client *http.Client) roundTripper {
	return func(ctx context.Context, msg string) (string, error) {
		// prepare request
		req, err := http.NewRequestWithContext(ctx,
			http.MethodPost, uri, strings.NewReader(msg))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/dns-message")

		// send request
		res, err := client.Do(req)
		if err != nil {
			return "", err
		}

		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return "", errors.New(http.StatusText(res.StatusCode))
		}

		// read response
		var str strings.Builder
		_, err = io.Copy(&str, res.Body)
		if err != nil {
			return "", err
		}
		return str.String(), nil
	}
}

func parseURITemplate(uri string) (string, error) {
	var str strings.Builder
	var exp bool

	for i := 0; i < len(uri); i++ {
		switch c := uri[i]; c {
		case '{':
			if exp {
				return "", errors.New("uri: invalid syntax")
			}
			exp = true
		case '}':
			if !exp {
				return "", errors.New("uri: invalid syntax")
			}
			exp = false
		default:
			if !exp {
				str.WriteByte(c)
			}
		}
	}

	return str.String(), nil
}
