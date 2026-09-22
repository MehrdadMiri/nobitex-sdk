package auth

import (
	"fmt"
	"net/http"
	"strings"
)

// TokenAuth sends `Authorization: Token <token>` as documented by Nobitex.
type TokenAuth struct {
	token string
}

// NewTokenAuth returns a Token authenticator. The token is stored in memory only.
func NewTokenAuth(token string) *TokenAuth {
	return &TokenAuth{token: strings.TrimSpace(token)}
}

// Apply implements Authenticator.
func (t *TokenAuth) Apply(req *http.Request, _ []byte) error {
	if t == nil || t.token == "" {
		return fmt.Errorf("auth: empty token")
	}
	req.Header.Set(headerAuthorization, "Token "+t.token)
	return nil
}
