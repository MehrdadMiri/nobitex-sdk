package client_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
)

func TestNewDefaultBaseURL(t *testing.T) {
	t.Parallel()
	c, err := client.New()
	if err != nil {
		t.Fatal(err)
	}
	if got := c.BaseURL(); got != client.DefaultBaseURL {
		t.Fatalf("BaseURL = %q, want %q", got, client.DefaultBaseURL)
	}
	wantUA := "TraderBot/" + client.DefaultAppName + "-" + client.DefaultAppVersion
	if got := c.UserAgent(); got != wantUA {
		t.Fatalf("UserAgent = %q, want %q", got, wantUA)
	}
}

func TestNewBaseURLOverride(t *testing.T) {
	t.Parallel()
	c, err := client.New(client.WithBaseURL("https://example.test/v2/"))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.BaseURL(); got != "https://example.test/v2" {
		t.Fatalf("BaseURL = %q", got)
	}
}

func TestNewInvalidBaseURL(t *testing.T) {
	t.Parallel()
	if _, err := client.New(client.WithBaseURL("not a url")); err == nil {
		t.Fatal("expected error")
	}
	if _, err := client.New(client.WithBaseURL("")); err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestUserAgentHeader(t *testing.T) {
	t.Parallel()
	var gotUA, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("secret-token"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if c.UserAgent() != "TraderBot/MyBot-1.0.0" {
		t.Fatalf("UserAgent() = %q", c.UserAgent())
	}

	var dest map[string]any
	if err := c.DoJSON(context.Background(), http.MethodGet, "/ping", &dest); err != nil {
		t.Fatal(err)
	}
	if gotUA != "TraderBot/MyBot-1.0.0" {
		t.Fatalf("request User-Agent = %q", gotUA)
	}
	if gotAuth != "Token secret-token" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if dest["status"] != "ok" {
		t.Fatalf("dest = %#v", dest)
	}
}

func TestUserAgentAlwaysOverwrites(t *testing.T) {
	t.Parallel()
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithApp("tradex", "2.3.4"))
	if err != nil {
		t.Fatal(err)
	}
	req, err := c.NewRequest(context.Background(), http.MethodGet, "/v2/options", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "definitely-not-traderbot")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if gotUA != "TraderBot/tradex-2.3.4" {
		t.Fatalf("User-Agent = %q", gotUA)
	}
}

func TestDoJSONAPIErrorHTTP200Failed(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"status":"failed","code":"InvalidSymbol","message":"bad"}`)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	err = c.DoJSON(context.Background(), http.MethodGet, "/v3/orderbook/NOPE", nil)
	if !sdkerr.IsAPI(err) {
		t.Fatalf("IsAPI = false, err=%v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.Code != "InvalidSymbol" {
		t.Fatalf("Code = %q", e.Code)
	}
}

func TestDoJSONHTTPStatusError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"status":"failed","code":"UnAuthenticated"}`)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	err = c.DoJSON(context.Background(), http.MethodGet, "/users/profile", nil)
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestTransportError(t *testing.T) {
	t.Parallel()
	rt := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})
	c, err := client.New(
		client.WithBaseURL("https://apiv2.nobitex.ir"),
		client.WithHTTPClient(&http.Client{Transport: rt}),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = c.DoJSON(context.Background(), http.MethodGet, "/v2/options", nil)
	if !sdkerr.IsTransport(err) {
		t.Fatalf("IsTransport = false, err=%v", err)
	}
}

func TestWithoutAuth(t *testing.T) {
	t.Parallel()
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.DoJSON(context.Background(), http.MethodGet, "/v3/orderbook/all", nil, client.WithoutAuth()); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "" {
		t.Fatalf("Authorization = %q, want empty", gotAuth)
	}
}

func TestAPIKeyHeadersOnDoJSON(t *testing.T) {
	t.Parallel()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := base64.URLEncoding.EncodeToString(priv.Seed())

	var key, ts, sig, ua string
	var rawBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key = r.Header.Get("Nobitex-Key")
		ts = r.Header.Get("Nobitex-Timestamp")
		sig = r.Header.Get("Nobitex-Signature")
		ua = r.Header.Get("User-Agent")
		rawBody, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithAPIKey("pub", secret),
	)
	if err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"hours": 2.4}
	if err := c.DoJSON(context.Background(), http.MethodPost, "/market/orders/cancel-old", nil, client.WithJSONBody(payload)); err != nil {
		t.Fatal(err)
	}
	if key != "pub" || ts == "" || sig == "" {
		t.Fatalf("missing API-key headers key=%q ts=%q sig=%q", key, ts, sig)
	}
	if ua != c.UserAgent() || ua == "" {
		t.Fatalf("User-Agent = %q", ua)
	}
	var decoded map[string]any
	if err := json.Unmarshal(rawBody, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["hours"] != 2.4 {
		t.Fatalf("body = %s", rawBody)
	}
}

func TestNewFromEnv(t *testing.T) {
	t.Setenv("NOBITEX_BASE_URL", "https://sandbox.example.test")
	t.Setenv("NOBITEX_APP_NAME", "tradex")
	t.Setenv("NOBITEX_APP_VERSION", "9.9.9")
	t.Setenv("NOBITEX_TOKEN", "env-token")
	t.Setenv("NOBITEX_API_KEY", "")
	t.Setenv("NOBITEX_API_SECRET", "")

	c, err := client.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL() != "https://sandbox.example.test" {
		t.Fatalf("BaseURL = %q", c.BaseURL())
	}
	if c.UserAgent() != "TraderBot/tradex-9.9.9" {
		t.Fatalf("UserAgent = %q", c.UserAgent())
	}
	if c.Auth() == nil {
		t.Fatal("expected token auth from env")
	}
}

func TestBaseURLPathPrefix(t *testing.T) {
	t.Parallel()
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL + "/gateway"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.DoJSON(context.Background(), http.MethodGet, "/v3/orderbook/all", nil); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/gateway/v3/orderbook/all" {
		t.Fatalf("path = %q", gotPath)
	}
}

func TestNewRequestRejectsAbsolutePath(t *testing.T) {
	t.Parallel()
	c, err := client.New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.NewRequest(context.Background(), http.MethodGet, "https://evil.test/x", nil); err == nil {
		t.Fatal("expected error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
