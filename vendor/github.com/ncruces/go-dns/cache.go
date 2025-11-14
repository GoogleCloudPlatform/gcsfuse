package dns

import (
	"context"
	"math"
	"net"
	"sync"
	"time"
)

// NewCachingResolver creates a caching [net.Resolver] that uses parent to resolve names.
func NewCachingResolver(parent *net.Resolver, options ...CacheOption) *net.Resolver {
	if parent == nil {
		parent = &net.Resolver{}
	}

	return &net.Resolver{
		PreferGo:     true,
		StrictErrors: parent.StrictErrors,
		Dial:         NewCachingDialer(parent.Dial, options...),
	}
}

// NewCachingDialer adds caching to a [net.Resolver.Dial] function.
func NewCachingDialer(parent DialFunc, options ...CacheOption) DialFunc {
	var cache = cache{dial: parent, negative: true}
	for _, o := range options {
		o.apply(&cache)
	}
	if cache.maxEntries == 0 {
		cache.maxEntries = DefaultMaxCacheEntries
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		conn := &dnsConn{}
		conn.roundTrip = cachingRoundTrip(&cache, network, address)
		return conn, nil
	}
}

const DefaultMaxCacheEntries = 150

// A CacheOption customizes the resolver cache.
type CacheOption interface {
	apply(*cache)
}

type maxEntriesOption int
type maxTTLOption time.Duration
type minTTLOption time.Duration
type negativeCacheOption bool

func (o maxEntriesOption) apply(c *cache)    { c.maxEntries = int(o) }
func (o maxTTLOption) apply(c *cache)        { c.maxTTL = time.Duration(o) }
func (o minTTLOption) apply(c *cache)        { c.minTTL = time.Duration(o) }
func (o negativeCacheOption) apply(c *cache) { c.negative = bool(o) }

// MaxCacheEntries sets the maximum number of entries to cache.
// If zero, [DefaultMaxCacheEntries] is used; negative means no limit.
func MaxCacheEntries(n int) CacheOption { return maxEntriesOption(n) }

// MaxCacheTTL sets the maximum time-to-live for entries in the cache.
func MaxCacheTTL(d time.Duration) CacheOption { return maxTTLOption(d) }

// MinCacheTTL sets the minimum time-to-live for entries in the cache.
func MinCacheTTL(d time.Duration) CacheOption { return minTTLOption(d) }

// NegativeCache sets whether to cache negative responses.
func NegativeCache(b bool) CacheOption { return negativeCacheOption(b) }

type cache struct {
	sync.RWMutex

	dial    DialFunc
	entries map[string]cacheEntry

	maxEntries int
	maxTTL     time.Duration
	minTTL     time.Duration
	negative   bool
}

type cacheEntry struct {
	deadline time.Time
	value    string
}

func (c *cache) put(req string, res string) {
	// ignore uncacheable/unparseable answers
	if invalid(req, res) {
		return
	}

	// ignore errors (if requested)
	if nameError(res) && !c.negative {
		return
	}

	// ignore uncacheable/unparseable answers
	ttl := getTTL(res)
	if ttl <= 0 {
		return
	}

	// adjust TTL
	if ttl < c.minTTL {
		ttl = c.minTTL
	}
	// maxTTL overrides minTTL
	if ttl > c.maxTTL && c.maxTTL != 0 {
		ttl = c.maxTTL
	}

	c.Lock()
	defer c.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]cacheEntry)
	}

	// do some cache evition
	var tested, evicted int
	for k, e := range c.entries {
		if time.Until(e.deadline) <= 0 {
			// delete expired entry
			delete(c.entries, k)
			evicted++
		}
		tested++

		if tested < 8 {
			continue
		}
		if evicted == 0 && c.maxEntries > 0 && len(c.entries) >= c.maxEntries {
			// delete at least one entry
			delete(c.entries, k)
		}
		break
	}

	// remove message IDs
	c.entries[req[2:]] = cacheEntry{
		deadline: time.Now().Add(ttl),
		value:    res[2:],
	}
}

func (c *cache) get(req string) (res string) {
	// ignore invalid messages
	if len(req) < 12 {
		return ""
	}
	if req[2] >= 0x7f {
		return ""
	}

	c.RLock()
	defer c.RUnlock()

	if c.entries == nil {
		return ""
	}

	// remove message ID
	entry, ok := c.entries[req[2:]]
	if ok && time.Until(entry.deadline) > 0 {
		// prepend correct ID
		return req[:2] + entry.value
	}
	return ""
}

func invalid(req string, res string) bool {
	if len(req) < 12 || len(res) < 12 { // header size
		return true
	}
	if req[0] != res[0] || req[1] != res[1] { // IDs match
		return true
	}
	if req[2] >= 0x7f || res[2] < 0x7f { // query, response
		return true
	}
	if req[2]&0x7a != 0 || res[2]&0x7a != 0 { // standard query, not truncated
		return true
	}
	if res[3]&0xf != 0 && res[3]&0xf != 3 { // no error, or name error
		return true
	}
	return false
}

func nameError(res string) bool {
	return res[3]&0xf == 3
}

func getTTL(msg string) time.Duration {
	ttl := math.MaxInt32

	qdcount := getUint16(msg[4:])
	ancount := getUint16(msg[6:])
	nscount := getUint16(msg[8:])
	arcount := getUint16(msg[10:])
	rdcount := ancount + nscount + arcount

	msg = msg[12:] // skip header

	// skip questions
	for i := 0; i < qdcount; i++ {
		name := getNameLen(msg)
		if name < 0 || name+4 > len(msg) {
			return -1
		}
		msg = msg[name+4:]
	}

	// parse records
	for i := 0; i < rdcount; i++ {
		name := getNameLen(msg)
		if name < 0 || name+10 > len(msg) {
			return -1
		}
		rtyp := getUint16(msg[name+0:])
		rttl := getUint32(msg[name+4:])
		rlen := getUint16(msg[name+8:])
		if name+10+rlen > len(msg) {
			return -1
		}
		// skip EDNS OPT since it doesn't have a TTL
		if rtyp != 41 && rttl < ttl {
			ttl = rttl
		}
		msg = msg[name+10+rlen:]
	}

	return time.Duration(ttl) * time.Second
}

func getNameLen(msg string) int {
	i := 0
	for i < len(msg) {
		if msg[i] == 0 {
			// end of name
			i += 1
			break
		}
		if msg[i] >= 0xc0 {
			// compressed name
			i += 2
			break
		}
		if msg[i] >= 0x40 {
			// reserved
			return -1
		}
		i += int(msg[i] + 1)
	}
	return i
}

func getUint16(s string) int {
	return int(s[1]) | int(s[0])<<8
}

func getUint32(s string) int {
	return int(s[3]) | int(s[2])<<8 | int(s[1])<<16 | int(s[0])<<24
}

func cachingRoundTrip(cache *cache, network, address string) roundTripper {
	return func(ctx context.Context, req string) (res string, err error) {
		// check cache
		if res := cache.get(req); res != "" {
			return res, nil
		}

		// dial connection
		var conn net.Conn
		if cache.dial != nil {
			conn, err = cache.dial(ctx, network, address)
		} else {
			var d net.Dialer
			conn, err = d.DialContext(ctx, network, address)
		}
		if err != nil {
			return "", err
		}

		ctx, cancel := context.WithCancel(ctx)
		go func() {
			<-ctx.Done()
			conn.Close()
		}()
		defer cancel()

		if t, ok := ctx.Deadline(); ok {
			err = conn.SetDeadline(t)
			if err != nil {
				return "", err
			}
		}

		// send request
		err = writeMessage(conn, req)
		if err != nil {
			return "", err
		}

		// read response
		res, err = readMessage(conn)
		if err != nil {
			return "", err
		}

		// cache response
		cache.put(req, res)
		return res, nil
	}
}
