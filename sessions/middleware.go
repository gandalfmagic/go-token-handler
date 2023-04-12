package sessions

import (
	"context"
	"net/http"

	"github.com/gandalfmagic/go-token-handler/oidc"
	"github.com/gandalfmagic/go-token-handler/zlogger"

	"golang.org/x/oauth2"
)

func (m *Manager) AuthenticationMiddleware(config *oidc.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zlog := zlogger.FromContext(r.Context())

		// Get the state and the cookie containing it
		session, err := m.GetSession(r, config)
		if err != nil {
			switch err {
			case ErrCannotGetCookie:
				zlog.JsonError(w, http.StatusUnauthorized, "invalid session data", err)
			case ErrSessionNotFound:
				zlog.JsonError(w, http.StatusUnauthorized, "no session found", err)
				return
			case ErrSessionInvalid:
				zlog.JsonError(w, http.StatusUnauthorized, "invalid or expired session", err)
				return
			default:
				zlog.JsonError(w, http.StatusInternalServerError, "cannot retrieve the session", err)
				return
			}
		}

		// Check if the access token is expired
		if session.data.IsExpired() {
			// The access token is expired (but not the user session), so we can
			// renew the access token using the refresh token we saved in the
			// database

			token, err := config.TokenSource(r.Context(), &oauth2.Token{
				RefreshToken: session.data.RefreshToken,
			}).Token()
			if err != nil {
				zlog.JsonError(w, http.StatusUnauthorized, "cannot renew the access token", err)
				return
			}

			// Save the session in the cookie, specify the cookie duration
			if err = session.Update(w, r, token); err != nil {
				zlog.JsonError(w, http.StatusInternalServerError, "cannot update the session", err)
				return
			}
		}

		// The token is saved in the context
		ctx := context.WithValue(r.Context(), ContextKeyAccessTokenName, session.data.AccessToken)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
