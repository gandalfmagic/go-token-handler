package oidc

import (
	"context"
	"fmt"

	"github.com/gandalfmagic/go-token-handler/opentelemetry"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type Config struct {
	oauth2.Config
	issuer        string
	OidcEndpoints Endpoints
}

func NewConfiguration(clientID, clientSecret, issuer, redirectURL string) (*Config, error) {
	oidcEndpoints, err := getOidcEndpoints(issuer)
	if err != nil {
		return nil, err
	}

	// Configure the OAuth2 client
	oauthConfig := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  oidcEndpoints.AuthorizationEndpoint,
			TokenURL: oidcEndpoints.TokenEndpoint,
		},
		Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
		RedirectURL: redirectURL,
	}

	return &Config{oauthConfig, issuer, oidcEndpoints}, nil
}

type IDToken struct {
	Token    *oidc.IDToken
	RawToken string
}

func (c *Config) GetIdToken(ctx context.Context, token *oauth2.Token) (*IDToken, error) {
	_, span := opentelemetry.TracerFromContext(ctx).Start(ctx, "oidc: verify and decode the the id-token")
	defer span.End()

	// Get the ID token from the OAuth2 token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no valid id_token in the response")
	}

	provider, err := oidc.NewProvider(ctx, c.issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to verify the id-token (1001)")
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: c.ClientID})

	// Parse the ID token
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify the id-token (1002)")
	}

	return &IDToken{Token: idToken, RawToken: rawIDToken}, nil
}

func (c *Config) AccessTokenIsValid(ctx context.Context, accessToken string) error {
	_, span := opentelemetry.TracerFromContext(ctx).Start(ctx, "oidc: verify and decode the the id-token")
	defer span.End()

	provider, err := oidc.NewProvider(ctx, c.issuer)
	if err != nil {
		return fmt.Errorf("failed to verify the access-token (1001)")
	}

	// The access token should have "audience" set to "account"
	verifier := provider.Verifier(&oidc.Config{ClientID: "account"})

	// Parse the ID token
	if _, err = verifier.Verify(ctx, accessToken); err != nil {
		return fmt.Errorf("failed to verify the access-token (1002)")
	}

	return nil
}
