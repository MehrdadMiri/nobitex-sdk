// Package errors defines the typed error model for transport failures and
// Nobitex API error bodies.
//
// Typical unsuccessful JSON (see https://apidocs.nobitex.ir/general_notes):
//
//	{"status":"failed","code":"ErrorCode","message":"..."}
//
// Rate-limit bodies may also include backOff (seconds) and limit. Legacy
// endpoints can return HTTP 200 with status=failed; newer ones use 4xx/5xx.
package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

// Kind classifies how a call failed.
type Kind string

const (
	// KindTransport is a network, TLS, timeout, or other HTTP-client failure.
	KindTransport Kind = "transport"
	// KindAPI is a Nobitex error body and/or non-success HTTP status.
	KindAPI Kind = "api"
	// KindDecode is a JSON unmarshal failure on an otherwise successful HTTP response.
	KindDecode Kind = "decode"
)

// Error is the SDK's typed error. Callers should use errors.As, not string matching.
type Error struct {
	Kind       Kind
	StatusCode int
	Status     types.Status
	Code       string
	Message    string
	BackOff    int
	Limit      int
	RawBody    []byte
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	switch e.Kind {
	case KindTransport:
		if e.Err != nil {
			return fmt.Sprintf("nobitex transport error: %v", e.Err)
		}
		return "nobitex transport error"
	case KindDecode:
		if e.Err != nil {
			return fmt.Sprintf("nobitex decode error: %v", e.Err)
		}
		return "nobitex decode error"
	default:
		if e.Code != "" && e.Message != "" {
			return fmt.Sprintf("nobitex api error: http=%d code=%s message=%s", e.StatusCode, e.Code, e.Message)
		}
		if e.Code != "" {
			return fmt.Sprintf("nobitex api error: http=%d code=%s", e.StatusCode, e.Code)
		}
		if e.Message != "" {
			return fmt.Sprintf("nobitex api error: http=%d message=%s", e.StatusCode, e.Message)
		}
		return fmt.Sprintf("nobitex api error: http=%d", e.StatusCode)
	}
}

// Unwrap supports errors.Is / errors.As chains.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Transport wraps a client/network failure.
func Transport(err error) *Error {
	return &Error{Kind: KindTransport, Err: err, Message: err.Error()}
}

// Decode wraps a JSON decode failure after a successful HTTP round-trip.
func Decode(statusCode int, body []byte, err error) *Error {
	return &Error{
		Kind:       KindDecode,
		StatusCode: statusCode,
		RawBody:    append([]byte(nil), body...),
		Err:        err,
		Message:    err.Error(),
	}
}

// FromResponse maps an HTTP status and body to an API error.
// It always returns a non-nil *Error; callers should use Check first.
func FromResponse(statusCode int, body []byte) *Error {
	e := &Error{
		Kind:       KindAPI,
		StatusCode: statusCode,
		RawBody:    append([]byte(nil), body...),
	}
	var env types.Envelope
	if len(body) > 0 && json.Unmarshal(body, &env) == nil {
		e.Status = env.Status
		e.Code = env.Code
		e.Message = env.Message
		e.BackOff = env.BackOff
		e.Limit = env.Limit
	}
	if e.Message == "" && len(body) > 0 && e.Code == "" {
		// Non-JSON or unexpected body: keep a short preview for debugging.
		preview := strings.TrimSpace(string(body))
		if len(preview) > 256 {
			preview = preview[:256]
		}
		e.Message = preview
	}
	if e.Message == "" {
		e.Message = http.StatusText(statusCode)
	}
	return e
}

// Check returns a typed API error when the HTTP status or JSON body indicates
// failure. It returns nil for successful responses (including HTTP 200 with
// status ok/success, or a non-JSON 2xx body).
func Check(statusCode int, body []byte) error {
	failedBody := false
	if len(body) > 0 {
		var env types.Envelope
		if json.Unmarshal(body, &env) == nil && env.Status.IsFailed() {
			failedBody = true
		}
	}
	if statusCode >= 400 || failedBody {
		return FromResponse(statusCode, body)
	}
	return nil
}

// IsKind reports whether err is an *Error of the given kind.
func IsKind(err error, kind Kind) bool {
	var e *Error
	return As(err, &e) && e.Kind == kind
}

// As is a convenience wrapper around stdlib errors.As for *Error.
func As(err error, target **Error) bool {
	return stderrors.As(err, target)
}

// IsTransport reports a network/HTTP-client failure.
func IsTransport(err error) bool { return IsKind(err, KindTransport) }

// IsAPI reports a mapped Nobitex API error body or HTTP error status.
func IsAPI(err error) bool { return IsKind(err, KindAPI) }

// IsDecode reports a JSON decode failure.
func IsDecode(err error) bool { return IsKind(err, KindDecode) }

// IsRateLimited reports a TooManyRequests API error (typically HTTP 429).
func IsRateLimited(err error) bool {
	var e *Error
	if !As(err, &e) {
		return false
	}
	return e.Code == "TooManyRequests" || e.StatusCode == http.StatusTooManyRequests
}
