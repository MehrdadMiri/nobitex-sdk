// Package types holds shared request/response shapes used across the SDK.
//
// Domain-specific structs for order book, margin, positions, and options belong
// in follow-up tickets; this package only ships the common envelope and
// conventions needed by the HTTP client core.
package types

// Status is the JSON "status" field used by most Nobitex HTTP APIs.
type Status string

const (
	// StatusOK is the usual success value.
	StatusOK Status = "ok"
	// StatusSuccess is an alternate success value used by some endpoints.
	StatusSuccess Status = "success"
	// StatusFailed is the documented unsuccessful body status, including on HTTP 200.
	StatusFailed Status = "failed"
)

// Envelope is the common JSON wrapper documented at https://apidocs.nobitex.ir/general_notes.
type Envelope struct {
	Status  Status `json:"status"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	BackOff int    `json:"backOff,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// Money is a decimal amount encoded as a string. Nobitex documents monetary
// fields as strings to avoid binary floating-point rounding.
type Money string

// IsOK reports whether status represents a successful API body.
func (s Status) IsOK() bool {
	return s == StatusOK || s == StatusSuccess
}

// IsFailed reports whether status represents a documented API failure body.
func (s Status) IsFailed() bool {
	return s == StatusFailed
}
