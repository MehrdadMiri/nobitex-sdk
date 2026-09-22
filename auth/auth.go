// Package auth implements Nobitex authentication for the HTTP client.
//
// Two mechanisms are supported, matching https://apidocs.nobitex.ir:
//
//  1. Token header — `Authorization: Token <token>` (legacy / session token).
//  2. API-key signing — `Nobitex-Key`, `Nobitex-Signature`, `Nobitex-Timestamp`
//     using Ed25519 over `timestamp + METHOD + full_path + raw_body`.
//
// Credentials are loaded from config or process environment only. Never hardcode
// tokens or keys, and never commit them to git. API keys that place or cancel
// orders need the TRADE permission (see https://apidocs.nobitex.ir/api_key/api-key-guide).
package auth

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	// EnvToken is the session/API token sent as Authorization: Token …
	EnvToken = "NOBITEX_TOKEN"
	// EnvAPIKey is the public API key (Nobitex-Key header).
	EnvAPIKey = "NOBITEX_API_KEY"
	// EnvAPISecret is the URL-safe Base64 Ed25519 private key used for signing.
	EnvAPISecret = "NOBITEX_API_SECRET"
)

const (
	headerAuthorization    = "Authorization"
	headerNobitexKey       = "Nobitex-Key"
	headerNobitexSignature = "Nobitex-Signature"
	headerNobitexTimestamp = "Nobitex-Timestamp"
)

// Authenticator attaches credentials to an outbound request.
// rawBody must be the exact bytes that will be written on the wire (needed for API-key signing).
type Authenticator interface {
	Apply(req *http.Request, rawBody []byte) error
}

// FromEnv builds an Authenticator from process environment.
//
// Priority: API key + secret (if both set) then token. If neither is configured,
// it returns (nil, nil) so public endpoints can still be called. If only one of
// the API-key pair is set, it returns an error (misconfiguration).
func FromEnv() (Authenticator, error) {
	key := strings.TrimSpace(os.Getenv(EnvAPIKey))
	secret := strings.TrimSpace(os.Getenv(EnvAPISecret))
	switch {
	case key != "" && secret != "":
		return NewAPIKeyAuth(key, secret)
	case key != "" || secret != "":
		return nil, fmt.Errorf("auth: set both %s and %s, or neither", EnvAPIKey, EnvAPISecret)
	}

	if token := strings.TrimSpace(os.Getenv(EnvToken)); token != "" {
		return NewTokenAuth(token), nil
	}
	return nil, nil
}
