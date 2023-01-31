package storageutil

import (
	"io"
	"net"
	"net/url"

	"google.golang.org/api/googleapi"
)

func ShouldRetry(err error) (b bool) {
	// HTTP 50x errors.
	if typed, ok := err.(*googleapi.Error); ok {
		if typed.Code >= 500 && typed.Code < 600 {
			b = true
			return
		}
	}

	// HTTP 429 errors (GCS uses these for rate limiting).
	if typed, ok := err.(*googleapi.Error); ok {
		if typed.Code == 429 {
			b = true
			return
		}
	}

	// HTTP 401 errors - Invalid Credentials
	// This is a work-around to fix the rare issue, we get while executing the
	// long ml-based test on gcsfuse. The better fix would be to expire the
	// cached token a little earlier which is used to access the GCS resources
	// through API.

	// TODO: please replace this implementation with the correction one after the
	// resolution of this issue: https://github.com/golang/oauth2/issues/623
	if typed, ok := err.(*googleapi.Error); ok {
		if typed.Code == 401 {
			b = true
			return
		}
	}

	// Network errors, which tend to show up transiently when doing lots of
	// operations in parallel. For example:
	//
	//     dial tcp 74.125.203.95:443: too many open files
	//
	if _, ok := err.(*net.OpError); ok {
		b = true
		return
	}

	// The HTTP package returns ErrUnexpectedEOF in several places. This seems to
	// come up when the server terminates the connection in the middle of an
	// object read.
	if err == io.ErrUnexpectedEOF {
		b = true
		return
	}

	// The HTTP library also appears to leak EOF errors from... somewhere in its
	// guts as URL errors sometimes.
	if urlErr, ok := err.(*url.Error); ok {
		if urlErr.Err == io.EOF {
			b = true
			return
		}
	}

	// Sometimes the HTTP package helpfully encapsulates the real error in a URL
	// error.
	if urlErr, ok := err.(*url.Error); ok {
		b = ShouldRetry(urlErr.Err)
		return
	}

	return
}
