package sessions

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gandalfmagic/go-token-handler/oidc"
	"github.com/gandalfmagic/go-token-handler/zlogger"

	"golang.org/x/oauth2"
)

func generateOAuthState() string {
	b := make([]byte, 128)
	_, _ = rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	return state
}

func (m *Manager) LoginHandlerOidc(config *oidc.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zlog := zlogger.FromContext(r.Context())

		session, err := m.NewSession(r, config)
		if err != nil {
			zlog.JsonError(w, http.StatusInternalServerError, "cannot create a new session", err)
			return
		}

		state := generateOAuthState()
		if err = session.saveState(w, r, state, m.loginTimeout); err != nil {
			zlog.JsonError(w, http.StatusInternalServerError, "cannot save the oauth state in the session", err)
			return
		}

		// Redirect the user to the login URL
		loginURL := config.AuthCodeURL(state, oauth2.AccessTypeOnline)
		http.Redirect(w, r, loginURL, http.StatusFound)
	})
}

func (m *Manager) CallbackHandlerOidc(config *oidc.Config, postLoginRedirectURI string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zlog := zlogger.FromContext(r.Context())

		// Get the state from the session cookie
		session, err := m.GetSession(r, config)
		if err != nil {
			zlog.JsonError(w, http.StatusInternalServerError, "cannot retrieve the session for oidc callback", err)
			return
		}

		// Verify that the "state" value in the response matches the one in the session
		responseState := r.URL.Query().Get(sessionStateName)
		if session.getState(r) != responseState {
			zlog.JsonError(w, http.StatusUnauthorized, "cannot validate state value for oidc callback", err)
			return
		}

		// Complete the authentication using the "code" field
		token, err := config.Exchange(r.Context(), r.FormValue("code"))
		if err != nil {
			zlog.JsonError(w, http.StatusUnauthorized, "cannot validate oauth code for oidc callback", err)
			return
		}

		// Preventing Session Fixation, renew the session token
		newSession, err := m.NewSession(r, config)
		if err != nil {
			zlog.JsonError(w, http.StatusInternalServerError, "cannot reinitialize the existing session for oidc callback", err)
			return
		}

		// Save the session in the database and in the cookie
		if err = newSession.Save(w, r, token); err != nil {
			zlog.JsonError(w, http.StatusInternalServerError, "cannot save the new session for oidc callback", err)
			return
		}

		// Redirect the user to the home page
		http.Redirect(w, r, postLoginRedirectURI, http.StatusFound)
	})
}

func (m *Manager) LogoutHandlerOidc(config *oidc.Config, postLogoutRedirectURI string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zlog := zlogger.FromContext(r.Context())

		// Get the state and the cookie containing it
		session, err := m.GetSession(r, config)
		if err != nil {
			zlog.JsonError(w, http.StatusInternalServerError, "cannot retrieve session for logout handler", err)
			return
		}

		// Delete the session cookie from the browser
		if err = session.Delete(w, r); err != nil {
			zlog.JsonError(w, http.StatusInternalServerError, "cannot delete the session for logout handler", err)
			return
		}

		// Redirect the user to the home page
		query := fmt.Sprintf("id_token_hint=%s&post_logout_redirect_uri=%s", url.QueryEscape(session.data.IDToken), url.QueryEscape(postLogoutRedirectURI))
		logoutUrl := fmt.Sprintf("%s?%s", config.OidcEndpoints.EndSessionEndpoint, query)
		http.Redirect(w, r, logoutUrl, http.StatusFound)
	})
}

func (m *Manager) UserInfoHandlerOidc(config *oidc.Config) http.Handler {
	return m.AuthenticationMiddleware(config, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zlog := zlogger.FromContext(ctx)

		// Retrieve the auth token from the context. Because the
		// r.Context().Value() method always returns an interface{} type, we
		// need to type assert it into a *sql.DB before using it.
		accessToken, ok := ctx.Value(ContextKeyAccessTokenName).(string)
		if !ok {
			zlog.JsonError(w, http.StatusInternalServerError, "cannot retrieve the access token from the context", nil)
			return
		}

		// Use the access token to get the user's profile information
		client := config.Client(r.Context(), &oauth2.Token{
			AccessToken: accessToken,
		})

		resp, err := client.Get(config.OidcEndpoints.UserInfoEndpoint)
		if err != nil {
			zlog.JsonError(w, http.StatusInternalServerError, "cannot retrieve the oidc userinfo endpoint", err)
			return
		}
		defer func() {
			if err = resp.Body.Close(); err != nil {
				zlog.SetError("error closing userinfo handler body: ", err)
			}
		}()

		// Write the response to the output
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			zlog.JsonError(w, http.StatusInternalServerError, "cannot copy userinfo response to the client", err)
			return
		}
	}))
}
