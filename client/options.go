package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MehrdadMiri/nobitex-sdk/auth"
)

// Option configures a Client. Options that fail (for example an invalid base
// URL) abort New.
type Option func(*Client) error

// WithBaseURL overrides the API root. Default: https://apiv2.nobitex.ir
func WithBaseURL(raw string) Option {
	return func(c *Client) error {
		return c.setBaseURL(raw)
	}
}

// WithHTTPClient replaces the default http.Client (timeout DefaultTimeout).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc == nil {
			return fmt.Errorf("client: nil http.Client")
		}
		c.httpClient = hc
		return nil
	}
}

// WithTimeout sets a timeout on a newly constructed default HTTP client.
// Ignored when WithHTTPClient is also used (apply timeout on that client instead).
func WithTimeout(d time.Duration) Option {
	return func(c *Client) error {
		if c.httpClient == nil {
			c.httpClient = &http.Client{Timeout: d}
			return nil
		}
		c.httpClient.Timeout = d
		return nil
	}
}

// WithAuth sets the request authenticator (token or API-key signer).
func WithAuth(a auth.Authenticator) Option {
	return func(c *Client) error {
		c.auth = a
		return nil
	}
}

// WithToken is a convenience wrapper around auth.NewTokenAuth.
func WithToken(token string) Option {
	return func(c *Client) error {
		token = strings.TrimSpace(token)
		if token == "" {
			return fmt.Errorf("client: empty token")
		}
		c.auth = auth.NewTokenAuth(token)
		return nil
	}
}

// WithAPIKey configures Ed25519 API-key signing from the public key and
// URL-safe Base64 private key. Credentials must come from env/config, not source.
func WithAPIKey(publicKey, privateKeyB64 string) Option {
	return func(c *Client) error {
		a, err := auth.NewAPIKeyAuth(publicKey, privateKeyB64)
		if err != nil {
			return err
		}
		c.auth = a
		return nil
	}
}

// WithApp sets both User-Agent segments: TraderBot/<name>-<version>.
func WithApp(name, version string) Option {
	return func(c *Client) error {
		c.appName = strings.TrimSpace(name)
		c.appVersion = strings.TrimSpace(version)
		return nil
	}
}

// WithAppName sets the User-Agent name segment.
func WithAppName(name string) Option {
	return func(c *Client) error {
		c.appName = strings.TrimSpace(name)
		return nil
	}
}

// WithAppVersion sets the User-Agent version segment.
func WithAppVersion(version string) Option {
	return func(c *Client) error {
		c.appVersion = strings.TrimSpace(version)
		return nil
	}
}

// RequestOption configures a single DoJSON call.
type RequestOption func(*requestConfig)

type requestConfig struct {
	query    url.Values
	jsonBody any
	rawBody  []byte
	auth     bool
}

// WithQuery adds query string parameters.
func WithQuery(q url.Values) RequestOption {
	return func(c *requestConfig) {
		c.query = q
	}
}

// WithJSONBody marshals v as the JSON request body (compact encoding).
func WithJSONBody(v any) RequestOption {
	return func(c *requestConfig) {
		c.jsonBody = v
	}
}

// WithRawBody sends body verbatim (the same bytes are used for API-key signing).
func WithRawBody(body []byte) RequestOption {
	return func(c *requestConfig) {
		c.rawBody = append([]byte(nil), body...)
	}
}

// WithoutAuth skips the client's authenticator (public endpoints).
func WithoutAuth() RequestOption {
	return func(c *requestConfig) {
		c.auth = false
	}
}
