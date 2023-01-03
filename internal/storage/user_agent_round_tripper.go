package storage

import "net/http"

// WithUserAgent returns a ClientOption that sets the User-Agent. This option is incompatible with the WithHTTPClient option.
// As we are using http-client, we will need to add this header via RoundTripper middleware.
// https://pkg.go.dev/google.golang.org/api/option#WithUserAgent
type userAgentRoundTripper struct {
	wrapped   http.RoundTripper
	UserAgent string
}

func (ug *userAgentRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", ug.UserAgent)
	return ug.wrapped.RoundTrip(r)
}
