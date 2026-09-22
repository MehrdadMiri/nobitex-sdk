package auth_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/auth"
)

func TestTokenApply(t *testing.T) {
	t.Parallel()
	a := auth.NewTokenAuth("abc123")
	req, err := http.NewRequest(http.MethodGet, "https://apiv2.nobitex.ir/users/profile", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Apply(req, nil); err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("Authorization"); got != "Token abc123" {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestTokenEmpty(t *testing.T) {
	t.Parallel()
	a := auth.NewTokenAuth("  ")
	req, _ := http.NewRequest(http.MethodGet, "https://apiv2.nobitex.ir/", nil)
	if err := a.Apply(req, nil); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestAPIKeySignRoundTrip(t *testing.T) {
	t.Parallel()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := base64.URLEncoding.EncodeToString(priv.Seed())
	a, err := auth.NewAPIKeyAuth("test-public-key", secret)
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"hours":2.4}`)
	const ts, method, path = "1700000000", "POST", "/market/orders/cancel-old"
	sigB64, err := a.Sign(ts, method, path, body)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := base64.URLEncoding.DecodeString(sigB64)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(ts + method + path + string(body))
	if !ed25519.Verify(priv.Public().(ed25519.PublicKey), payload, sig) {
		t.Fatal("signature did not verify against documented payload")
	}
}

func TestAPIKeyApplyHeaders(t *testing.T) {
	t.Parallel()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := base64.RawURLEncoding.EncodeToString(priv.Seed())
	a, err := auth.NewAPIKeyAuth("pub-key", secret)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://apiv2.nobitex.ir/market/orders/cancel-old", nil)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"hours":2.4}`)
	if err := a.Apply(req, body); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Nobitex-Key") != "pub-key" {
		t.Errorf("Nobitex-Key = %q", req.Header.Get("Nobitex-Key"))
	}
	if req.Header.Get("Nobitex-Timestamp") == "" {
		t.Error("missing Nobitex-Timestamp")
	}
	if req.Header.Get("Nobitex-Signature") == "" {
		t.Error("missing Nobitex-Signature")
	}
	if req.Header.Get("Authorization") != "" {
		t.Error("API-key auth must not set Authorization")
	}
}

func TestFromEnvToken(t *testing.T) {
	t.Setenv(auth.EnvToken, "tok")
	t.Setenv(auth.EnvAPIKey, "")
	t.Setenv(auth.EnvAPISecret, "")
	a, err := auth.FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	tok, ok := a.(*auth.TokenAuth)
	if !ok || tok == nil {
		t.Fatalf("got %T", a)
	}
}

func TestFromEnvAPIKeyPrecedence(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(auth.EnvToken, "tok")
	t.Setenv(auth.EnvAPIKey, "pub")
	t.Setenv(auth.EnvAPISecret, base64.URLEncoding.EncodeToString(priv.Seed()))
	a, err := auth.FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.(*auth.APIKeyAuth); !ok {
		t.Fatalf("expected APIKeyAuth, got %T", a)
	}
}

func TestFromEnvIncompleteAPIKey(t *testing.T) {
	t.Setenv(auth.EnvAPIKey, "pub")
	t.Setenv(auth.EnvAPISecret, "")
	t.Setenv(auth.EnvToken, "")
	if _, err := auth.FromEnv(); err == nil {
		t.Fatal("expected misconfiguration error")
	}
}

func TestFromEnvNone(t *testing.T) {
	t.Setenv(auth.EnvToken, "")
	t.Setenv(auth.EnvAPIKey, "")
	t.Setenv(auth.EnvAPISecret, "")
	a, err := auth.FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if a != nil {
		t.Fatalf("expected nil authenticator, got %T", a)
	}
}

func TestNewAPIKeyAuthInvalidLength(t *testing.T) {
	t.Parallel()
	_, err := auth.NewAPIKeyAuth("pub", base64.URLEncoding.EncodeToString([]byte("short")))
	if err == nil {
		t.Fatal("expected invalid length error")
	}
}
