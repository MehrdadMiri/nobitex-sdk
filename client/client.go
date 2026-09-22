// Package client is the HTTP core for the Nobitex SDK.
//
// It owns base URL, User-Agent, auth application, and error mapping. Typed
// endpoint methods call DoJSON / Do. Order book v3 and SystemOptions
// (GET /v2/options) are implemented here; margin, positions, and cancel
// land in follow-up tickets.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/MehrdadMiri/nobitex-sdk/auth"
	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
)

const (
	// DefaultBaseURL is the production Nobitex REST host.
	DefaultBaseURL = "https://apiv2.nobitex.ir"
	// DefaultAppName is the User-Agent name segment when none is configured.
	DefaultAppName = "nobitex-sdk"
	// DefaultAppVersion is the User-Agent version segment when none is configured.
	DefaultAppVersion = "0.1.0"
	// DefaultTimeout is used when the caller does not supply an HTTP client.
	DefaultTimeout = 30 * time.Second

	envBaseURL    = "NOBITEX_BASE_URL"
	envAppName    = "NOBITEX_APP_NAME"
	envAppVersion = "NOBITEX_APP_VERSION"
)

// Client is a reusable Nobitex HTTP client.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	auth       auth.Authenticator
	appName    string
	appVersion string
}

// New constructs a Client. The default base URL is DefaultBaseURL and the
// default User-Agent is TraderBot/nobitex-sdk-0.1.0.
func New(opts ...Option) (*Client, error) {
	c := &Client{
		appName:    DefaultAppName,
		appVersion: DefaultAppVersion,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if c.baseURL == nil {
		if err := c.setBaseURL(DefaultBaseURL); err != nil {
			return nil, err
		}
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	return c, nil
}

// NewFromEnv constructs a Client using environment variables (see README).
// Explicit opts are applied last and win. Missing credentials are allowed so
// public endpoints can be used without auth.
func NewFromEnv(opts ...Option) (*Client, error) {
	envOpts := make([]Option, 0, 4)
	if v := strings.TrimSpace(os.Getenv(envBaseURL)); v != "" {
		envOpts = append(envOpts, WithBaseURL(v))
	}
	if v := strings.TrimSpace(os.Getenv(envAppName)); v != "" {
		envOpts = append(envOpts, WithAppName(v))
	}
	if v := strings.TrimSpace(os.Getenv(envAppVersion)); v != "" {
		envOpts = append(envOpts, WithAppVersion(v))
	}
	a, err := auth.FromEnv()
	if err != nil {
		return nil, err
	}
	if a != nil {
		envOpts = append(envOpts, WithAuth(a))
	}
	return New(append(envOpts, opts...)...)
}

// BaseURL returns the configured API root without a trailing slash.
func (c *Client) BaseURL() string {
	if c == nil || c.baseURL == nil {
		return ""
	}
	return strings.TrimRight(c.baseURL.String(), "/")
}

// UserAgent returns the User-Agent sent on every outbound request.
// Format: TraderBot/<name>-<version> (see https://apidocs.nobitex.ir/general_notes).
func (c *Client) UserAgent() string {
	name := c.appName
	version := c.appVersion
	if strings.TrimSpace(name) == "" {
		name = DefaultAppName
	}
	if strings.TrimSpace(version) == "" {
		version = DefaultAppVersion
	}
	return "TraderBot/" + name + "-" + version
}

// Auth returns the configured authenticator, or nil for unauthenticated clients.
func (c *Client) Auth() auth.Authenticator {
	return c.auth
}

// NewRequest builds an HTTP request against the client's base URL.
// path should start with '/' (for example "/v3/orderbook/BTCIRT").
func (c *Client) NewRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	if c == nil || c.baseURL == nil {
		return nil, fmt.Errorf("client: not initialized")
	}
	u, err := c.joinURL(path)
	if err != nil {
		return nil, err
	}

	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), rdr)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
		req.ContentLength = int64(len(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent())
	return req, nil
}

// Do executes req after applying User-Agent and authentication.
// Callers must close the response body when the error is nil.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.do(req, true)
}

func (c *Client) do(req *http.Request, applyAuth bool) (*http.Response, error) {
	if c == nil || c.httpClient == nil {
		return nil, sdkerr.Transport(fmt.Errorf("client: not initialized"))
	}
	if req == nil {
		return nil, sdkerr.Transport(fmt.Errorf("client: nil request"))
	}

	req.Header.Set("User-Agent", c.UserAgent())

	var rawBody []byte
	if req.Body != nil {
		var err error
		rawBody, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, sdkerr.Transport(fmt.Errorf("client: read request body: %w", err))
		}
		req.Body = io.NopCloser(bytes.NewReader(rawBody))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(rawBody)), nil
		}
		req.ContentLength = int64(len(rawBody))
	}

	if applyAuth && c.auth != nil {
		if err := c.auth.Apply(req, rawBody); err != nil {
			return nil, sdkerr.Transport(fmt.Errorf("client: apply auth: %w", err))
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, sdkerr.Transport(err)
	}
	return resp, nil
}

// DoJSON marshals in (if non-nil), performs the request, maps Nobitex errors,
// and unmarshals a successful JSON body into dest (if non-nil).
func (c *Client) DoJSON(ctx context.Context, method, path string, dest any, opts ...RequestOption) error {
	cfg := requestConfig{auth: true}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	var body []byte
	if cfg.rawBody != nil {
		body = cfg.rawBody
	} else if cfg.jsonBody != nil {
		var err error
		body, err = json.Marshal(cfg.jsonBody)
		if err != nil {
			return sdkerr.Decode(0, nil, fmt.Errorf("client: marshal body: %w", err))
		}
	}

	if len(cfg.query) > 0 {
		rel, err := url.Parse(path)
		if err != nil {
			return sdkerr.Transport(fmt.Errorf("client: parse path %q: %w", path, err))
		}
		q := rel.Query()
		for k, vs := range cfg.query {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		rel.RawQuery = q.Encode()
		path = rel.String()
	}

	req, err := c.NewRequest(ctx, method, path, body)
	if err != nil {
		return sdkerr.Transport(err)
	}

	resp, err := c.do(req, cfg.auth)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return sdkerr.Transport(fmt.Errorf("client: read response: %w", err))
	}
	if err := sdkerr.Check(resp.StatusCode, respBody); err != nil {
		return err
	}
	if dest == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, dest); err != nil {
		return sdkerr.Decode(resp.StatusCode, respBody, err)
	}
	return nil
}

// joinURL appends a relative API path (and optional query) onto the base URL.
// A path starting with '/' does not replace a gateway prefix on the base URL
// (unlike url.URL.ResolveReference).
func (c *Client) joinURL(path string) (*url.URL, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("client: parse path %q: %w", path, err)
	}
	if rel.Scheme != "" || rel.Host != "" {
		return nil, fmt.Errorf("client: path must be relative, got %q", path)
	}

	relPath := rel.Path
	if relPath == "" {
		relPath = "/"
	} else if !strings.HasPrefix(relPath, "/") {
		relPath = "/" + relPath
	}

	u := *c.baseURL
	u.Path = strings.TrimSuffix(u.Path, "/") + relPath
	u.RawPath = ""
	u.RawQuery = rel.RawQuery
	u.Fragment = rel.Fragment
	return &u, nil
}

func (c *Client) setBaseURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("client: empty base URL")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("client: parse base URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("client: base URL must include scheme and host")
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	c.baseURL = u
	return nil
}
