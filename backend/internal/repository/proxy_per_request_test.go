package repository

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPerRequestProxyConnectKeepsHTTPSAvailable(t *testing.T) {
	for _, proxyScheme := range []string{"http", "https"} {
		t.Run(proxyScheme, func(t *testing.T) {
			origin := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				require.Empty(t, request.Header.Get("X-Client-Request-ID"))
				require.Empty(t, request.Header.Get("X-Sub2API-Attempt-ID"))
				_, _ = io.WriteString(writer, "ok")
			}))
			origin.EnableHTTP2 = true
			origin.StartTLS()
			defer origin.Close()

			connectHeaders := make(chan http.Header, 2)
			proxy := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				connectHeaders <- request.Header.Clone()
				upstream, err := net.Dial("tcp", request.Host)
				if err != nil {
					writer.WriteHeader(http.StatusBadGateway)
					return
				}
				defer upstream.Close()
				client, buffered, err := http.NewResponseController(writer).Hijack()
				if err != nil {
					return
				}
				defer client.Close()
				_, _ = io.WriteString(client, "HTTP/1.1 200 Connection Established\r\n\r\n")
				go func() { _, _ = io.Copy(upstream, buffered) }()
				_, _ = io.Copy(client, upstream)
			}))
			if proxyScheme == "https" {
				proxy.StartTLS()
			} else {
				proxy.Start()
			}
			defer proxy.Close()

			proxyURL, err := url.Parse(proxy.URL)
			require.NoError(t, err)
			proxyURL.User = url.UserPassword("user", "pass")
			proxyRoute := proxyURL.String() + "#" + service.ProxyPerRequestURLFragment
			gateway := NewHTTPUpstream(nil).(*httpUpstreamService)
			entry, err := gateway.getOrCreateClient(proxyRoute, 42, 1)
			require.NoError(t, err)
			transport := entry.client.Transport.(*http.Transport)
			require.True(t, transport.DisableKeepAlives)
			require.False(t, transport.ForceAttemptHTTP2)
			roots := x509.NewCertPool()
			roots.AddCert(origin.Certificate())
			if proxyScheme == "https" {
				roots.AddCert(proxy.Certificate())
			}
			transport.TLSClientConfig = &tls.Config{RootCAs: roots}
			defer entry.client.CloseIdleConnections()

			for _, requestID := range []string{"request-a", "request-b"} {
				request, createErr := http.NewRequestWithContext(context.WithValue(context.Background(), ctxkey.ClientRequestID, requestID), http.MethodGet, origin.URL, nil)
				require.NoError(t, createErr)
				response, doErr := gateway.Do(request, proxyRoute, 42, 1)
				require.NoError(t, doErr)
				require.Equal(t, 1, response.ProtoMajor)
				body, readErr := io.ReadAll(response.Body)
				require.NoError(t, readErr)
				require.Equal(t, "ok", string(body))
				require.NoError(t, response.Body.Close())
			}
			require.Len(t, connectHeaders, 2)
			first, second := <-connectHeaders, <-connectHeaders
			require.Equal(t, "request-a", first.Get("X-Client-Request-ID"))
			require.Equal(t, "request-b", second.Get("X-Client-Request-ID"))
			require.NotEmpty(t, first.Get("X-Sub2API-Attempt-ID"))
			require.NotEqual(t, first.Get("X-Sub2API-Attempt-ID"), second.Get("X-Sub2API-Attempt-ID"))
			require.Empty(t, first.Get("Connection"))
			require.Equal(t, "Basic dXNlcjpwYXNz", first.Get("Proxy-Authorization"))
		})
	}
}

func TestPerRequestProxyPlainHTTPHeaders(t *testing.T) {
	requests := make(chan http.Header, 2)
	connections := make(chan string, 2)
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Header.Clone()
		connections <- request.RemoteAddr
		_, _ = io.WriteString(writer, "proxied")
	}))
	defer proxy.Close()
	proxyRoute := proxy.URL + "#" + service.ProxyPerRequestURLFragment
	upstream := NewHTTPUpstream(nil).(*httpUpstreamService)
	entry, err := upstream.getOrCreateClient(proxyRoute, 42, 1)
	require.NoError(t, err)
	transport := entry.client.Transport.(*http.Transport)
	require.True(t, transport.DisableKeepAlives)
	require.Nil(t, transport.TLSNextProto["h2"])
	defer entry.client.CloseIdleConnections()
	for _, requestID := range []string{"request-a", "request-b"} {
		request, createErr := http.NewRequestWithContext(context.WithValue(context.Background(), ctxkey.ClientRequestID, requestID), http.MethodGet, "http://example.com/path", nil)
		require.NoError(t, createErr)
		response, doErr := upstream.Do(request, proxyRoute, 42, 1)
		require.NoError(t, doErr)
		require.NoError(t, response.Body.Close())
	}
	require.Len(t, requests, 2)
	first, second := <-requests, <-requests
	require.Equal(t, "request-a", first.Get("X-Client-Request-ID"))
	require.Equal(t, "request-b", second.Get("X-Client-Request-ID"))
	require.NotEqual(t, first.Get("X-Sub2API-Attempt-ID"), second.Get("X-Sub2API-Attempt-ID"))
	require.NotEqual(t, <-connections, <-connections)
}

func TestPerRequestProxyModeDoesNotChangeExistingPool(t *testing.T) {
	upstream := NewHTTPUpstream(nil).(*httpUpstreamService)
	normal, err := upstream.getOrCreateClient("http://proxy.example:8080", 7, 1)
	require.NoError(t, err)
	isolated, err := upstream.getOrCreateClient("http://proxy.example:8080#"+service.ProxyPerRequestURLFragment, 7, 1)
	require.NoError(t, err)
	require.NotSame(t, normal, isolated)
	require.False(t, normal.client.Transport.(*http.Transport).DisableKeepAlives)
	require.True(t, isolated.client.Transport.(*http.Transport).DisableKeepAlives)
	same, err := upstream.getOrCreateClient("http://proxy.example:8080", 7, 1)
	require.NoError(t, err)
	require.Same(t, normal, same)
	require.Len(t, upstream.clients, 2)
	socksKey, _, err := normalizeProxyURL("socks5://proxy.example:1080#" + service.ProxyPerRequestURLFragment)
	require.NoError(t, err)
	require.NotContains(t, socksKey, service.ProxyPerRequestURLFragment)
	require.Same(t, normal.client, withProxyRequestHeaders(normal.client, socksKey))
}

func TestPerRequestProxyDisablesOpenAIHTTP2Fallback(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.OpenAIHTTP2.Enabled = true
	upstream := NewHTTPUpstream(cfg).(*httpUpstreamService)
	entry, err := upstream.getClientEntry("http://proxy.example:8080#"+service.ProxyPerRequestURLFragment, 7, 1, service.HTTPUpstreamProfileOpenAI, false, false)
	require.NoError(t, err)
	require.Equal(t, upstreamProtocolModeOpenAIH1, entry.protocolMode)
	require.True(t, entry.client.Transport.(*http.Transport).DisableKeepAlives)
}

func TestPerRequestProxyRejectsUntrustedHeaderValues(t *testing.T) {
	require.Nil(t, proxyRequestHeaders(context.Background()))
	require.Nil(t, proxyRequestHeaders(context.WithValue(context.Background(), ctxkey.ClientRequestID, "bad\r\nheader")))
}

func TestPerRequestProxyURLStillWorksForProbeClients(t *testing.T) {
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer proxy.Close()
	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL: proxy.URL + "#" + service.ProxyPerRequestURLFragment,
		Timeout:  time.Second,
	})
	require.NoError(t, err)
	request, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	require.NoError(t, err)
	response, err := client.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, response.StatusCode)
	require.NoError(t, response.Body.Close())
}

func TestPerRequestTLSFingerprintConnectHeaders(t *testing.T) {
	headers := make(chan http.Header, 1)
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		headers <- request.Header.Clone()
		writer.WriteHeader(http.StatusForbidden)
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	require.NoError(t, err)
	proxyURL.User = url.UserPassword("user", "pass")
	settings := defaultPoolSettings(nil)
	settings.perRequest = true
	transport, err := buildUpstreamTransportWithTLSFingerprint(settings, proxyURL, &tlsfingerprint.Profile{Name: "test"})
	require.NoError(t, err)
	require.True(t, transport.DisableKeepAlives)
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "fingerprint-request")
	_, err = transport.DialTLSContext(ctx, "tcp", "example.com:443")
	require.Error(t, err)
	connectHeader := <-headers
	require.Equal(t, "fingerprint-request", connectHeader.Get("X-Client-Request-ID"))
	require.NotEmpty(t, connectHeader.Get("X-Sub2API-Attempt-ID"))
	require.Equal(t, "Basic dXNlcjpwYXNz", connectHeader.Get("Proxy-Authorization"))
}
