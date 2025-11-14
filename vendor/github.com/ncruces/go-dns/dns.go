// Package dns provides [net.Resolver] instances implementing caching,
// opportunistic encryption, and DNS over TLS/HTTPS.
//
// To replace the [net.DefaultResolver] with a caching DNS over HTTPS instance
// using the Google Public DNS resolver:
//
//	net.DefaultResolver = dns.NewDoHResolver(
//		"https://dns.google/dns-query",
//		dns.DoHCache())
package dns

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"
)

// OpportunisticResolver opportunistically tries encrypted DNS over TLS
// using the local resolver.
var OpportunisticResolver = &net.Resolver{
	Dial:     opportunisticDial,
	PreferGo: true,
}

func opportunisticDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, _ := net.SplitHostPort(address)
	if (port == "53" || port == "domain") && notBadServer(address) {
		deadline, ok := ctx.Deadline()
		if ok && deadline.After(time.Now().Add(2*time.Second)) {
			var d net.Dialer
			d.Timeout = time.Second
			tlsAddr := net.JoinHostPort(host, "853")
			tlsConf := tls.Config{InsecureSkipVerify: true}
			conn, _ := tls.DialWithDialer(&d, "tcp", tlsAddr, &tlsConf)
			if conn != nil {
				return conn, nil
			}
			addBadServer(address)
		}
	}

	var d net.Dialer
	return d.DialContext(ctx, network, address)
}

var badServers struct {
	sync.Mutex
	next int
	list [4]string
}

func notBadServer(address string) bool {
	badServers.Lock()
	defer badServers.Unlock()
	for _, a := range badServers.list {
		if a == address {
			return false
		}
	}
	return true
}

func addBadServer(address string) {
	badServers.Lock()
	defer badServers.Unlock()
	for _, a := range badServers.list {
		if a == address {
			return
		}
	}
	badServers.list[badServers.next] = address
	badServers.next = (badServers.next + 1) % len(badServers.list)
}

// DialFunc is a [net.Resolver.Dial] function.
type DialFunc func(ctx context.Context, network, address string) (net.Conn, error)
