package storageutil

import (
	"errors"
	"io"
	"net"
	"net/url"
	"testing"

	. "github.com/jacobsa/ogletest"
	"google.golang.org/api/googleapi"
)

func TestCustomRetry(t *testing.T) { RunTests(t) }

type customRetryTest struct {
}

func init() { RegisterTestSuite(&customRetryTest{}) }

func (t customRetryTest) ShouldRetryReturnsTrueWithGoogleApiError() {
	// 401
	var err401 = googleapi.Error{
		Code: 401,
		Body: "Invalid Credential",
	}
	// 50x error
	var err502 = googleapi.Error{
		Code: 502,
	}
	// 429 - rate limiting error
	var err429 = googleapi.Error{
		Code: 429,
		Body: "API rate limit exceeded",
	}

	ExpectEq(ShouldRetry(&err401), true)
	ExpectEq(ShouldRetry(&err502), true)
	ExpectEq(ShouldRetry(&err429), true)
}

func (t customRetryTest) ShouldRetryReturnsFalseWithGoogleApiError400() {
	// 400 - bad request
	var err400 = googleapi.Error{
		Code: 400,
	}

	ExpectEq(ShouldRetry(&err400), false)
}

func (t customRetryTest) ShouldRetryReturnsTrueWithUnexpectedEOFError() {
	ExpectEq(ShouldRetry(io.ErrUnexpectedEOF), true)
}

func (t customRetryTest) ShouldRetryReturnsTrueWithNetworkError() {
	ExpectEq(ShouldRetry(&net.OpError{
		Err: errors.New("use of closed network connection")}), true)
}

func (t customRetryTest) ShouldRetryReturnsTrueURLError() {
	var urlErrConnRefused = url.Error{
		Err: errors.New("connection refused"),
	}
	var urlErrConnReset = url.Error{
		Err: errors.New("connection reset"),
	}

	ExpectEq(ShouldRetry(&urlErrConnRefused), true)
	ExpectEq(ShouldRetry(&urlErrConnReset), true)
}
