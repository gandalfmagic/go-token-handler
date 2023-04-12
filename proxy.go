package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gandalfmagic/go-token-handler/sessions"
	"github.com/gandalfmagic/go-token-handler/zlogger"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type ProxyConfig struct {
	Timeout         time.Duration
	KeepAlive       time.Duration
	MaxIdleConns    int
	IdleConnTimeout time.Duration
}

func NewProxy(ctx context.Context, targetHost string, config ProxyConfig) (*httputil.ReverseProxy, error) {
	targetURL, err := url.Parse(targetHost)
	if err != nil {
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL = targetURL
			r.Host = targetURL.Host
			rCtx := r.Context()

			// Retrieve the auth token from the context. Because the
			// r.Context().Value() method always returns an interface{} type, we
			// need to type assert it into a *sql.DB before using it.
			accessToken, ok := rCtx.Value(sessions.ContextKeyAccessTokenName).(string)
			if ok {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
			}

			r.Header.Add("X-Forwarded-For", r.RemoteAddr)
			otel.GetTextMapPropagator().Inject(rCtx, propagation.HeaderCarrier(r.Header))
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			zlogger.FromContext(ctx).JsonError(w, http.StatusBadGateway, "reverse proxy error", err)
		},
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   config.Timeout,
				KeepAlive: config.KeepAlive,
			}).DialContext,
			ForceAttemptHTTP2: true,
			MaxIdleConns:      config.MaxIdleConns,
			IdleConnTimeout:   config.IdleConnTimeout,
			//TLSHandshakeTimeout:   10 * time.Second,
			//ExpectContinueTimeout: 1 * time.Second,
		},
		FlushInterval: -1,
	}

	return proxy, nil
}
