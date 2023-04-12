package oidc

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	openIDConfigurationPath = "/.well-known/openid-configuration"
)

type Endpoints struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	EndSessionEndpoint    string `json:"end_session_endpoint"`
	IntrospectionEndpoint string `json:"introspection_endpoint"`
	JWKSUri               string `json:"jwks_uri"`
	RevocationEndpoint    string `json:"revocation_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
}

func getOidcEndpoints(issuer string) (Endpoints, error) {
	// Make the GET request
	resp, err := http.Get(fmt.Sprintf("%s%s", issuer, openIDConfigurationPath))
	if err != nil {
		return Endpoints{}, err
	}
	defer resp.Body.Close()

	// Parse the JSON response
	var config Endpoints
	err = json.NewDecoder(resp.Body).Decode(&config)
	if err != nil {
		return Endpoints{}, err
	}

	return config, nil
}
