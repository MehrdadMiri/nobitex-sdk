package types

import (
	"fmt"
	"strings"
	"time"
)

// WithdrawStatus is a documented withdrawal lifecycle value.
type WithdrawStatus string

const (
	WithdrawNew        WithdrawStatus = "New"
	WithdrawVerified   WithdrawStatus = "Verified"
	WithdrawAccepted   WithdrawStatus = "Accepted"
	WithdrawProcessing WithdrawStatus = "Processing"
	WithdrawWaiting    WithdrawStatus = "Waiting"
	WithdrawSent       WithdrawStatus = "Sent"
	WithdrawDone       WithdrawStatus = "Done"
	WithdrawCanceled   WithdrawStatus = "Canceled"
	WithdrawRejected   WithdrawStatus = "Rejected"
)

// Withdraw is one crypto or rial withdrawal (list + get).
type Withdraw struct {
	ID            int64          `json:"id"`
	CreatedAt     time.Time      `json:"createdAt"`
	Status        WithdrawStatus `json:"status"`
	Amount        Money          `json:"amount"`
	Currency      string         `json:"currency"`
	Network       string         `json:"network"`
	Address       string         `json:"address"`
	Tag           *string        `json:"tag"`
	WalletID      int64          `json:"wallet_id"`
	BlockchainURL *string        `json:"blockchain_url"`
	IsCancelable  bool           `json:"is_cancelable"`
	Invoice       *string        `json:"invoice"`
}

// WithdrawListQuery is GET /users/wallets/withdraws/list.
type WithdrawListQuery struct {
	Wallet   string // wallet id or "all" (API default all)
	From     string // YYYY-MM-DD (created_at >= from)
	To       string // YYYY-MM-DD (created_at <= to)
	Page     int
	PageSize int
}

// Normalize trims filters. Empty wallet is omitted (API default: all).
func (q WithdrawListQuery) Normalize() (WithdrawListQuery, error) {
	out := q
	out.Wallet = strings.TrimSpace(out.Wallet)
	out.From = strings.TrimSpace(out.From)
	out.To = strings.TrimSpace(out.To)
	if out.Page < 0 || out.PageSize < 0 {
		return WithdrawListQuery{}, fmt.Errorf("types: withdraw list page/pageSize must be >= 0")
	}
	return out, nil
}

// WithdrawListResponse is GET /users/wallets/withdraws/list (read-only).
type WithdrawListResponse struct {
	Envelope
	Withdraws []Withdraw `json:"withdraws"`
	HasNext   bool       `json:"hasNext"`
}

// WithdrawByID returns the row with the given id, if present.
func (r *WithdrawListResponse) WithdrawByID(id int64) (Withdraw, bool) {
	if r == nil {
		return Withdraw{}, false
	}
	for _, w := range r.Withdraws {
		if w.ID == id {
			return w, true
		}
	}
	return Withdraw{}, false
}

// WithdrawResponse is GET /withdraws/:withdrawId (read-only).
type WithdrawResponse struct {
	Envelope
	Withdraw *Withdraw `json:"withdraw"`
}
