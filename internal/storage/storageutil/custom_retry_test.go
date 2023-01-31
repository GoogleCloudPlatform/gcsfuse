package storageutil

import (
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

func (t customRetryTest) TestGoogleAPIErrorRetry() {
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
	// 400 - bad request
	var err400 = googleapi.Error{
		Code: 400,
	}

	ExpectEq(ShouldRetry(&err401), true)
	ExpectEq(ShouldRetry(&err502), true)
	ExpectEq(ShouldRetry(&err429), true)
	ExpectEq(ShouldRetry(&err400), false)
}

func (t customRetryTest) TestUnexpectedEOFErrorRetry() {
	ExpectEq(ShouldRetry(io.ErrUnexpectedEOF), true)
}

func (t customRetryTest) TestNetworkErrorRetry() {
	ExpectEq(ShouldRetry(&net.OpError{}), true)
}

func (t customRetryTest) TestURLErrorRetry() {
	var urlErr401 = url.Error{
		Err: &googleapi.Error{
			Code: 401,
			Body: "Invalid Credential",
		},
	}
	var urlErr400 = url.Error{
		Err: &googleapi.Error{
			Code: 400,
			Body: "Bad Request",
		},
	}
	var urlEOF = url.Error{
		Err: io.EOF,
	}

	ExpectEq(ShouldRetry(&urlErr400), false)
	ExpectEq(ShouldRetry(&urlErr401), true)
	ExpectEq(ShouldRetry(&urlEOF), true)
}
