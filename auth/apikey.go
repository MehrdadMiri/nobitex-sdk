package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIKeyAuth signs requests with an Ed25519 API key as documented at
// https://apidocs.nobitex.ir/api_key/api-key-guide
//
// Payload (UTF-8, concatenated, no separators):
//
//	timestamp + METHOD + full_path + raw_body
//
// Headers set on each request:
//
//	Nobitex-Key, Nobitex-Signature (URL-safe Base64), Nobitex-Timestamp (unix seconds UTC)
//
// Timestamp must be within ~30s of server time in production.
type APIKeyAuth struct {
	publicKey string
	private   ed25519.PrivateKey
	now       func() time.Time
}

// NewAPIKeyAuth parses a URL-safe Base64 private key (32-byte seed or 64-byte
// expanded key) and returns a signer. publicKey is sent unchanged as Nobitex-Key.
func NewAPIKeyAuth(publicKey, privateKeyB64 string) (*APIKeyAuth, error) {
	publicKey = strings.TrimSpace(publicKey)
	privateKeyB64 = strings.TrimSpace(privateKeyB64)
	if publicKey == "" {
		return nil, fmt.Errorf("auth: empty API public key")
	}
	seed, err := decodeKey(privateKeyB64)
	if err != nil {
		return nil, fmt.Errorf("auth: decode API private key: %w", err)
	}
	priv, err := privateKeyFromBytes(seed)
	if err != nil {
		return nil, err
	}
	return &APIKeyAuth{
		publicKey: publicKey,
		private:   priv,
		now:       func() time.Time { return time.Now().UTC() },
	}, nil
}

// Sign returns the URL-safe Base64 Ed25519 signature for the documented payload.
func (a *APIKeyAuth) Sign(timestamp, method, fullPath string, rawBody []byte) (string, error) {
	if a == nil || len(a.private) == 0 {
		return "", fmt.Errorf("auth: API key signer not initialized")
	}
	payload := signingPayload(timestamp, method, fullPath, rawBody)
	sig := ed25519.Sign(a.private, payload)
	return base64.URLEncoding.EncodeToString(sig), nil
}

// Apply implements Authenticator.
func (a *APIKeyAuth) Apply(req *http.Request, rawBody []byte) error {
	if a == nil || len(a.private) == 0 {
		return fmt.Errorf("auth: API key signer not initialized")
	}
	if req.URL == nil {
		return fmt.Errorf("auth: request URL is nil")
	}
	ts := strconv.FormatInt(a.now().Unix(), 10)
	method := strings.ToUpper(req.Method)
	fullPath := req.URL.RequestURI()
	sig, err := a.Sign(ts, method, fullPath, rawBody)
	if err != nil {
		return err
	}
	req.Header.Set(headerNobitexKey, a.publicKey)
	req.Header.Set(headerNobitexTimestamp, ts)
	req.Header.Set(headerNobitexSignature, sig)
	return nil
}

func signingPayload(timestamp, method, fullPath string, rawBody []byte) []byte {
	var b strings.Builder
	b.Grow(len(timestamp) + len(method) + len(fullPath) + len(rawBody))
	b.WriteString(timestamp)
	b.WriteString(strings.ToUpper(method))
	b.WriteString(fullPath)
	if len(rawBody) > 0 {
		b.Write(rawBody)
	}
	return []byte(b.String())
}

func decodeKey(s string) ([]byte, error) {
	encodings := []*base64.Encoding{
		base64.URLEncoding,
		base64.RawURLEncoding,
		base64.StdEncoding,
		base64.RawStdEncoding,
	}
	var last error
	for _, enc := range encodings {
		out, err := enc.DecodeString(s)
		if err == nil {
			return out, nil
		}
		last = err
	}
	return nil, last
}

func privateKeyFromBytes(raw []byte) (ed25519.PrivateKey, error) {
	switch len(raw) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(raw), nil
	default:
		return nil, fmt.Errorf("auth: invalid Ed25519 private key length %d (want %d or %d)", len(raw), ed25519.SeedSize, ed25519.PrivateKeySize)
	}
}
