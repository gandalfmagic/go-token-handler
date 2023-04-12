package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gandalfmagic/go-token-handler/opentelemetry"
	"github.com/gandalfmagic/go-token-handler/zlogger"

	"github.com/gandalfmagic/realip"
	"go.uber.org/zap"
)

var (
	listenAddress = ":9081"
)

func init() {
	if os.Getenv("LISTEN_ADDRESS") != "" {
		listenAddress = os.Getenv("LISTEN_ADDRESS")
	}
}

func main() {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	zlog, err := zlogger.NewLogger("info", false)
	if err != nil {
		log.Fatalf("error initializing the logger: %s", err)
	}
	defer zlog.Sync()
	ctx = zlogger.NewContext(ctx, zlog)

	// Create the opentelemetry tracer
	tp, err := opentelemetry.InitTracer("backend-test-service", "localhost", "6831")
	if err != nil {
		zlog.Fatal("error initializing the open-telemetry tracer", zap.Error(err))
	}
	defer func() {
		if err = tp.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			zlog.Error("error shutting down open-telemetry provider", zap.Error(err))
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/", opentelemetry.Middleware(http.HandlerFunc(rootHandler), "gitlab.oitech.it/devops/token-handler", "/"))

	// Start the HTTP server
	go func() {
		if err := http.ListenAndServe(listenAddress, zlog.Middleware(Authorize(mux))); err != nil && !errors.Is(err, http.ErrServerClosed) {
			stop()
			log.Fatal("error starting the api server:", zap.Error(err))
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	zlog.Info("shutting down gracefully, press Ctrl+C again to force")
	stop()
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	zlog := zlogger.FromContext(r.Context())

	claims, _ := r.Context().Value(ClaimsCtxKey{}).(*Claims)
	if claims == nil {
		zlog.JsonError(w, http.StatusUnauthorized, "", fmt.Errorf("cannot retrieve the token claims"))
		return
	}

	// TODO: here we should check them claims in detail for authorization

	realIP, _ := realip.Get(r)

	data := struct {
		Timestamp time.Time `json:"timestamp"`
		ProxyIP   string    `json:"proxy_ip"`
		RealIP    string    `json:"real_ip"`
	}{
		Timestamp: time.Now(),
		ProxyIP:   r.RemoteAddr,
		RealIP:    realIP,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		zlog.JsonError(w, http.StatusInternalServerError, "", err)
		return
	}

	_, _ = fmt.Fprint(w, string(jsonData))
}
