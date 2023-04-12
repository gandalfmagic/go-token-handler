package sessions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gandalfmagic/go-token-handler/database"
	"github.com/gandalfmagic/go-token-handler/oidc"
	"github.com/gandalfmagic/go-token-handler/opentelemetry"
	"github.com/gandalfmagic/go-token-handler/zlogger"

	"github.com/gorilla/sessions"
	"go.uber.org/zap"
)

type Manager struct {
	cookieName     string
	loginTimeout   int
	sessionTimeout time.Duration
	store          *sessions.CookieStore
	sessionImpl    database.SessionImpl
	done           chan struct{}
}

var (
	ErrNilValue = errors.New("you cannot pass a nil value")
)

type parseType string

const (
	parseCurrent parseType = "current"
	parseOld     parseType = "old"
)

var (
	once sync.Once
)

func NewManager(ctx context.Context, c Configuration) (*Manager, error) {
	if c.SessionImpl == nil {
		return nil, fmt.Errorf("%w: %s", ErrNilValue, "Configuration.DB")
	}

	kpSlice, err := c.keyPairsAsSlice()
	if err != nil {
		return nil, err
	}

	store := sessions.NewCookieStore(kpSlice...)
	store.Options = &sessions.Options{
		HttpOnly: true,
		MaxAge:   0,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Domain:   c.CookieDomain,
	}

	done := make(chan struct{})
	go once.Do(func() {
		expiredSessionsCleaner(ctx, c.SessionImpl, done)
	})

	return &Manager{
		cookieName:     c.CookieName,
		loginTimeout:   int(c.LoginTimeout / time.Second),
		sessionTimeout: c.SessionTimeout,
		store:          store,
		sessionImpl:    c.SessionImpl,
		done:           done,
	}, nil
}

func expiredSessionsCleaner(ctx context.Context, db database.SessionImpl, done chan<- struct{}) {
	ticker := time.NewTicker(10 * time.Minute)
	defer func() {
		ticker.Stop()
		close(done)
	}()

	for {
		select {
		case <-ctx.Done():
			zlogger.FromContext(ctx).Debug("session manager: stopping the expired sessions cleaner job")
			return
		case <-ticker.C:
			if err := db.Purge(ctx); err != nil {
				zlogger.FromContext(ctx).Error("cannot clear the expired sessions", zap.Error(err))
			}
		}
	}
}

func (m *Manager) WaitSessionCleaner(ctx context.Context) {
	<-m.done
	zlogger.FromContext(ctx).Debug("session manager: the expired sessions cleaner is now stopped")
}

func (m *Manager) NewSession(r *http.Request, config *oidc.Config) (*Session, error) {
	_, span := opentelemetry.TracerFromContext(r.Context()).Start(r.Context(), "session-manager: create new session into a cookie")
	defer span.End()

	session, err := m.store.New(r, m.cookieName)
	if err != nil {
		return nil, fmt.Errorf("%w '%s': %s", ErrCannotCreateCookie, m.cookieName, err.Error())
	}

	return &Session{session: session, sessionImpl: m.sessionImpl, timeout: m.sessionTimeout, oidcConfig: config}, nil
}

func (m *Manager) GetSession(r *http.Request, config *oidc.Config) (*Session, error) {
	ctx, span := opentelemetry.TracerFromContext(r.Context()).Start(r.Context(), "session-manager: retrieve existing session from a cookie")
	defer span.End()

	session, err := m.store.Get(r, m.cookieName)
	if err != nil {
		return nil, fmt.Errorf("%w '%s': %s", ErrCannotGetCookie, m.cookieName, err)
	}
	s := &Session{session: session, sessionImpl: m.sessionImpl, timeout: m.sessionTimeout, oidcConfig: config}

	// If the session already exists, and contains a state, we are in the login callback,
	// the actual session still doesn't exist, so we don't load the data from the DB
	if _, ok := s.session.Values[sessionStateName].(string); ok {
		return s, nil
	}

	if err = s.dataFromDB(ctx); err != nil {
		return nil, err
	}

	return s, nil
}
