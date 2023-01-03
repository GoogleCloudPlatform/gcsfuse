package storage

import "net/http"

type userAgentRoundTripper struct {
	wrapped   http.RoundTripper
	UserAgent string
}

func (ug *userAgentRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", ug.UserAgent)
	return ug.wrapped.RoundTrip(r)
}
