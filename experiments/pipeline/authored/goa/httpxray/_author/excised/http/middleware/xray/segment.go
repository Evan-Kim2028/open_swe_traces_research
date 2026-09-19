package xray

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"

	"example.internal/apikit/v3/middleware/xray"
)

type (
	// HTTPSegment represents an AWS X-Ray segment document for HTTP services.
	// It wraps the AWS X-Ray segment with the http response writer.
	HTTPSegment struct {
		*xray.Segment
		http.ResponseWriter
	}
)

// RecordRequest traces a request.
//
// It sets Http.Request & Namespace (ex: "remote")
func (s *HTTPSegment) RecordRequest(req *http.Request, namespace string) { panic("excised: RecordRequest") }

// RecordResponse traces a response.
//
// It sets Throttle, Fault, Error and HTTP.Response
func (s *HTTPSegment) RecordResponse(resp *http.Response) { panic("excised: RecordResponse") }

// WriteHeader records the HTTP response code and calls the corresponding
// ResponseWriter method.
func (s *HTTPSegment) WriteHeader(code int) { panic("excised: WriteHeader") }

// Write records the HTTP response content length and error (if any)
// and calls the corresponding ResponseWriter method.
func (s *HTTPSegment) Write(p []byte) (int, error) { panic("excised: Write") }

// Hijack supports the http.Hijacker interface.
func (s *HTTPSegment) Hijack() (net.Conn, *bufio.ReadWriter, error) { panic("excised: Hijack") }

// recordStatusCode sets Throttle, Fault, Error
//
// It is expected that the mutex has already been locked when calling this method.
func (s *HTTPSegment) recordStatusCode(statusCode int) { panic("excised: recordStatusCode") }

// requestData creates a Request from a http.Request.
func requestData(req *http.Request) *xray.Request {
	var (
		scheme = "http"
		host   = req.Host
	)
	if len(req.URL.Scheme) > 0 {
		scheme = req.URL.Scheme
	}
	if len(req.URL.Host) > 0 {
		host = req.URL.Host
	}

	return &xray.Request{
		Method:        req.Method,
		URL:           fmt.Sprintf("%s://%s%s", scheme, host, req.URL.Path),
		ClientIP:      getIP(req),
		UserAgent:     req.UserAgent(),
		ContentLength: req.ContentLength,
	}
}

// responseData creates a Response from a http.Response.
func responseData(resp *http.Response) *xray.Response {
	return &xray.Response{
		Status:        resp.StatusCode,
		ContentLength: resp.ContentLength,
	}
}

// getIP implements a heuristic that returns an origin IP address for a request.
func getIP(req *http.Request) string {
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		for _, ip := range strings.Split(req.Header.Get(h), ",") {
			if len(ip) == 0 {
				continue
			}
			realIP := net.ParseIP(strings.ReplaceAll(ip, " ", ""))
			return realIP.String()
		}
	}

	// not found in header
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}
