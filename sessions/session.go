package sessions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gandalfmagic/go-token-handler/database"
	"github.com/gandalfmagic/go-token-handler/oidc"
	"github.com/gandalfmagic/go-token-handler/opentelemetry"

	"github.com/gorilla/sessions"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
)

var (
	ErrCannotCreateCookie        = errors.New("cannot create a new cookie")
	ErrCannotGetCookie           = errors.New("cannot get the cookie")
	ErrSessionNotFound           = errors.New("cannot retrieve session from storage")
	ErrCannotRetrieveSessionData = errors.New("cannot retrieve session data")
	ErrSessionInvalid            = errors.New("cannot retrieve session id from memory")
)

type Session struct {
	sessionImpl database.SessionImpl
	session     *sessions.Session
	oidcConfig  *oidc.Config
	data        database.SessionData
	timeout     time.Duration
}

func (s *Session) dataFromDB(ctx context.Context) (err error) {
	_, span := opentelemetry.TracerFromContext(ctx).Start(ctx, "session: get session data from db")
	defer span.End()

	id, ok := s.session.Values[sessionIdName].(string)
	if !ok {
		return ErrSessionInvalid
	}

	s.data, err = s.sessionImpl.Get(ctx, id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return ErrSessionNotFound
		default:
			return fmt.Errorf("%w: %s", ErrCannotRetrieveSessionData, err)
		}
	}

	return
}

func (s *Session) saveState(w http.ResponseWriter, r *http.Request, state string, age int) error {
	_, span := opentelemetry.TracerFromContext(r.Context()).Start(r.Context(), "session: save state into a cookie")
	defer span.End()

	s.session.Options.MaxAge = age
	s.session.Values[sessionStateName] = state

	return s.session.Save(r, w)
}

func (s *Session) getState(r *http.Request) string {
	_, span := opentelemetry.TracerFromContext(r.Context()).Start(r.Context(), "session: get state from a cookie")
	defer span.End()

	state, ok := s.session.Values[sessionStateName].(string)
	if !ok {
		return ""
	}

	return state
}

func (s *Session) newData(ctx context.Context, token *oauth2.Token) (database.SessionData, error) {
	var span trace.Span
	ctx, span = opentelemetry.TracerFromContext(ctx).Start(ctx, "session: create a new session dataset")
	defer span.End()

	// Verify the id-token and decode it
	idToken, err := s.oidcConfig.GetIdToken(ctx, token)
	if err != nil {
		return database.SessionData{}, err
	}

	return database.SessionData{
		Subject:      idToken.Token.Subject,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      idToken.RawToken,
		ExpiresAt:    token.Expiry, // token expiration
	}, nil
}

func (s *Session) saveSession(w http.ResponseWriter, r *http.Request, id string) error {
	_, span := opentelemetry.TracerFromContext(r.Context()).Start(r.Context(), "session: save session into a cookie")
	defer span.End()

	delete(s.session.Values, sessionStateName)
	s.session.Values[sessionIdName] = id
	s.session.Options.MaxAge = int(s.timeout.Seconds())

	return s.session.Save(r, w)
}

func (s *Session) Save(w http.ResponseWriter, r *http.Request, token *oauth2.Token) error {
	ctx, span := opentelemetry.TracerFromContext(r.Context()).Start(r.Context(), "session: save")
	defer span.End()

	var err error
	if s.data, err = s.newData(ctx, token); err != nil {
		return err
	}

	id, err := s.sessionImpl.Add(ctx, s.data)
	if err != nil {
		return err
	}

	return s.saveSession(w, r.WithContext(ctx), id)
}

func (s *Session) Update(w http.ResponseWriter, r *http.Request, token *oauth2.Token) error {
	ctx, span := opentelemetry.TracerFromContext(r.Context()).Start(r.Context(), "session: update")
	defer span.End()

	var err error
	if s.data, err = s.newData(ctx, token); err != nil {
		return err
	}

	id, ok := s.session.Values[sessionIdName].(string)
	if !ok {
		return ErrSessionInvalid
	}

	// Save the session in the database
	if err := s.sessionImpl.Update(ctx, id, s.data); err != nil {
		return err
	}

	return s.saveSession(w, r.WithContext(ctx), id)
}

func (s *Session) Delete(w http.ResponseWriter, r *http.Request) error {
	ctx, span := opentelemetry.TracerFromContext(r.Context()).Start(r.Context(), "session: delete")
	defer span.End()

	id, ok := s.session.Values[sessionIdName].(string)
	if !ok {
		return ErrSessionInvalid
	}

	// Delete the session from the database
	if err := s.sessionImpl.Delete(ctx, id); err != nil {
		return err
	}

	s.session.Options.MaxAge = -1

	return s.session.Save(r, w)
}
