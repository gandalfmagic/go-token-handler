package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os/signal"
	"syscall"
	"time"

	"github.com/gandalfmagic/go-token-handler/config"
	"github.com/gandalfmagic/go-token-handler/database"
	"github.com/gandalfmagic/go-token-handler/oidc"
	"github.com/gandalfmagic/go-token-handler/opentelemetry"
	"github.com/gandalfmagic/go-token-handler/sessions"
	"github.com/gandalfmagic/go-token-handler/zlogger"

	"github.com/gandalfmagic/encryption"
	"go.uber.org/zap"
)

func main() {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load the configuration
	c, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("error loading the configuation: %s", err)
	}

	// Create the logger
	zlog, err := zlogger.NewLogger(c.LogLevel, c.IsProduction)
	if err != nil {
		log.Fatalf("error initializing the logger: %s", err)
	}
	defer zlog.Sync()
	ctx = zlogger.NewContext(ctx, zlog)

	// Create the opentelemetry tracer
	tp, err := opentelemetry.InitTracer("token-handler", "localhost", "6831")
	if err != nil {
		zlog.Fatal("error initializing the open-telemetry tracer", zap.Error(err))
	}
	defer func() {
		if err = tp.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			zlog.Error("error shutting down open-telemetry provider", zap.Error(err))
		}
	}()

	// Create Encryption cipher
	cipher, err := encryption.NewXChaCha20Cipher(c.SessionDBKey, c.SessionOldDBKey)
	if err != nil && !errors.Is(err, encryption.ErrNoEncryptionKeys) {
		zlog.Fatal("error initializing encryption", zap.Error(err))
	}

	// Connect to the database
	var sessionImpl database.SessionImpl
	switch c.DBType {
	case "sqlite":
		sessionImpl, err = database.NewSQLiteSessionImpl(ctx, cipher, c.DBName)
	case "postgresql":
		sessionImpl, err = database.NewPostgresqlSessionImpl(ctx, cipher, c.DBHost, c.DBName, c.DBUsername, c.DBPassword)
	}
	if err != nil {
		zlog.Fatal("error creating a database connection", zap.Error(err))
	}
	defer func() {
		if err = sessionImpl.CloseConnection(ctx); err != nil {
			zlog.Error("error closing the database connection", zap.Error(err))
		}
	}()

	oidcConfig, err := oidc.NewConfiguration(c.OidcClientID, c.OidcClientSecret, c.OidcIssuer, c.OidcRedirectURL)
	if err != nil {
		zlog.Fatal("error creating a new oidc configuration", zap.Error(err))
	}

	// Create the sessions store
	mc := sessions.Configuration{
		NewKeyPair: sessions.KeyPair{
			Authentication: c.SessionAuthSecret,
			Encryption:     c.SessionEncSecret,
		},
		OldKeyPair: sessions.KeyPair{
			Authentication: c.SessionOldAuthSecret,
			Encryption:     c.SessionOldEncSecret,
		},
		CookieName:     c.CookieName,
		CookieDomain:   c.CookieDomain,
		LoginTimeout:   5 * time.Minute,
		SessionTimeout: 30 * time.Minute,
		SessionImpl:    sessionImpl,
	}
	sessionManager, err := sessions.NewManager(ctx, mc)
	if err != nil {
		zlog.Fatal("error creating a new session manager", zap.Error(err))
	}
	defer sessionManager.WaitSessionCleaner(ctx)

	mux := http.NewServeMux()

	// Set up the HTTP routes
	// TODO: add CORS, all the endpoints use the SPA as origin, the login callback uses Keycloak
	mux.Handle("/login", opentelemetry.Middleware(sessionManager.LoginHandlerOidc(oidcConfig), "gitlab.oitech.it/devops/token-handler", "GET /login"))
	mux.Handle("/callback", opentelemetry.Middleware(sessionManager.CallbackHandlerOidc(oidcConfig, c.OidcPostLoginRedirectURL), "gitlab.oitech.it/devops/token-handler", "GET /callback"))
	mux.Handle("/logout", opentelemetry.Middleware(sessionManager.LogoutHandlerOidc(oidcConfig, c.OidcPostLogoutRedirectURL), "gitlab.oitech.it/devops/token-handler", "GET /logout"))
	mux.Handle("/userinfo", opentelemetry.Middleware(sessionManager.UserInfoHandlerOidc(oidcConfig), "gitlab.oitech.it/devops/token-handler", "GET /userinfo"))

	if c.ProxyConfig != "" {
		proxyConfigs, err := c.ReadProxyConfig()
		if err != nil {
			zlog.Fatal("cannot read the proxy configuration", zap.Error(err))
		}

		for _, proxyConfig := range proxyConfigs.Proxies {
			zlog.Info(fmt.Sprintf("creating proxy for service %s on %s", proxyConfig.Target, proxyConfig.Endpoint))
			proxy, err := NewProxy(ctx, proxyConfig.Target, ProxyConfig{
				IdleConnTimeout: proxyConfig.Parameters.IdleConnTimeout,
				MaxIdleConns:    proxyConfig.Parameters.MaxIdleConns,
				DialKeepAlive:   proxyConfig.Parameters.DialKeepAlive,
				DialTimeout:     proxyConfig.Parameters.DialTimeout,
			})
			if err != nil {
				zlog.Fatal(fmt.Sprintf("error creating proxy service for %s on %s", proxyConfig.Target, proxyConfig.Endpoint), zap.Error(err))
			}

			mux.Handle(proxyConfig.Endpoint, opentelemetry.Middleware(sessionManager.AuthenticationMiddleware(oidcConfig, http.HandlerFunc(ProxyRequestHandler(proxy))), "gitlab.oitech.it/devops/token-handler", "GET /proxy"))
		}
	}

	// You can add your own handlers to the mux to customize the token-handler functionality
	//mux.Handle("/test", opentelemetry.Middleware(sessionManager.AuthenticationMiddleware(oidcConfig, http.HandlerFunc(customHandler1)), "gitlab.oitech.it/devops/token-handler", "GET /test"))
	//mux.Handle("/test", opentelemetry.Middleware(sessionManager.AuthenticationMiddleware(oidcConfig, http.HandlerFunc(customHandler2)), "gitlab.oitech.it/devops/token-handler", "GET /test"))

	mux.Handle("/", opentelemetry.Middleware(http.HandlerFunc(rootHandler), "gitlab.oitech.it/devops/token-handler", "GET /"))

	// Start the HTTP server
	zlog.Info(fmt.Sprintf("main service is listening on %s", c.ListenAddr))
	go func() {
		if err = http.ListenAndServe(c.ListenAddr, zlog.Middleware(mux)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zlog.Fatal("error starting the api server", zap.Error(err))
			stop()
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	zlog.Info("shutting down gracefully, press Ctrl+C again to force")
	stop()
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	zlogger.FromContext(r.Context()).JsonError(w, http.StatusNotFound, "", nil)
}

// ProxyRequestHandler handles the http request using proxy
func ProxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}
