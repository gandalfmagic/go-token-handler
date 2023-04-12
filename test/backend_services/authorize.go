package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gandalfmagic/go-token-handler/zlogger"

	"github.com/coreos/go-oidc/v3/oidc"
)

type ClaimsCtxKey struct{}

type Claims struct {
	Sub               string   `json:"sub"`
	Azp               string   `json:"azp"`
	Scope             string   `json:"scope"`
	EmailVerified     bool     `json:"email_verified"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	GivenName         string   `json:"given_name"`
	FamilyName        string   `json:"family_name"`
	Email             string   `json:"email"`
	Emails            []string `json:"emails"`
}

var (
	issuer = os.Getenv("OIDC_ISSUER")
	//clientId = os.Getenv("OIDC_CLIENT_ID")
	clientId = "account"
)

var provider *oidc.Provider

func init() {
	var err error

	provider, err = oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		log.Fatalf("error initializing the oidc provider: %s", err)
	}
}

func Authorize(next http.Handler) http.Handler {
	verifier := provider.Verifier(&oidc.Config{ClientID: clientId})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zlog := zlogger.FromContext(r.Context())

		accessToken := r.Header.Get("Authorization")

		splitToken := strings.Split(accessToken, "Bearer")
		if len(splitToken) != 2 {
			zlog.JsonError(w, http.StatusUnauthorized, "", fmt.Errorf("cannot split the bearer token"))
			return
		}

		accessToken = strings.TrimSpace(splitToken[1])

		token, err := verifier.Verify(r.Context(), accessToken)
		if err != nil {
			zlog.JsonError(w, http.StatusUnauthorized, "", err)
			return
		}

		var claims Claims
		if err = token.Claims(&claims); err != nil {
			zlog.JsonError(w, http.StatusUnauthorized, "", err)
			return
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ClaimsCtxKey{}, &claims)))
	})
}
