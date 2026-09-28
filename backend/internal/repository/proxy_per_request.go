package repository

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

type proxyRequestHeaderTransport struct {
	base http.RoundTripper
}

func proxyRequestHeaders(ctx context.Context) http.Header {
	requestID, _ := ctx.Value(ctxkey.ClientRequestID).(string)
	if len(requestID) == 0 || len(requestID) > 64 {
		return nil
	}
	for _, character := range requestID {
		if character < 33 || character > 126 {
			return nil
		}
	}
	return http.Header{
		"X-Client-Request-Id":  {requestID},
		"X-Sub2api-Attempt-Id": {uuid.NewString()},
	}
}

func proxyRequestConnectHeaders(ctx context.Context, _ *url.URL, _ string) (http.Header, error) {
	return proxyRequestHeaders(ctx), nil
}

func withProxyRequestHeaders(client *http.Client, proxyKey string) *http.Client {
	if !strings.HasSuffix(proxyKey, "#"+service.ProxyPerRequestURLFragment) {
		return client
	}
	copyClient := *client
	copyClient.Transport = &proxyRequestHeaderTransport{base: client.Transport}
	return &copyClient
}

func (t *proxyRequestHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !strings.EqualFold(req.URL.Scheme, "http") {
		return t.base.RoundTrip(req)
	}
	headers := proxyRequestHeaders(req.Context())
	if headers == nil {
		return t.base.RoundTrip(req)
	}
	copyRequest := req.Clone(req.Context())
	if copyRequest.Header == nil {
		copyRequest.Header = make(http.Header)
	}
	for name, values := range headers {
		copyRequest.Header[name] = values
	}
	return t.base.RoundTrip(copyRequest)
}
