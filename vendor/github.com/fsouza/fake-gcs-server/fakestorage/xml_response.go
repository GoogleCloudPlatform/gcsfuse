package fakestorage

import (
	"encoding/xml"
	"net/http"
)

type xmlResponse struct {
	status       int
	header       http.Header
	data         interface{}
	errorMessage string
}

type xmlHandler = func(r *http.Request) xmlResponse

func xmlToHTTPHandler(h xmlHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := h(r)
		w.Header().Set("Content-Type", "application/xml")
		for name, values := range resp.header {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}

		status := resp.getStatus()
		var data interface{}
		if status > 399 {
			data = newErrorResponse(status, resp.getErrorMessage(status), nil)
		} else {
			data = resp.data
		}

		w.WriteHeader(status)
		xml.NewEncoder(w).Encode(data)
	}
}

func (r *xmlResponse) getStatus() int {
	if r.status > 0 {
		return r.status
	}
	if r.errorMessage != "" {
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

func (r *xmlResponse) getErrorMessage(status int) string {
	if r.errorMessage != "" {
		return r.errorMessage
	}
	return http.StatusText(status)
}
