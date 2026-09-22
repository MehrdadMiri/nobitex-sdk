package types

import (
	"fmt"
	"strings"
)

// WalletType is the documented wallet venue: spot, margin, credit, debit.
type WalletType string

const (
	WalletSpot   WalletType = "spot"
	WalletMargin WalletType = "margin"
	WalletCredit WalletType = "credit"
	WalletDebit  WalletType = "debit"
)

func (t WalletType) valid() bool {
	switch t {
	case "", WalletSpot, WalletMargin, WalletCredit, WalletDebit:
		return true
	default:
		return false
	}
}

// UserProfileResponse is GET /users/profile.
// websocketAuthParam on Profile is what private WS channel names need.
type UserProfileResponse struct {
	Envelope
	Profile           UserProfile `json:"profile"`
	IsBusinessAccount bool        `json:"isBusinessAccount"`
	TradeStats        TradeStats  `json:"tradeStats"`
}

// UserProfile is the documented profile object (common fields). Extra
// undocumented keys are ignored — this ticket prefers breadth over KYC depth.
type UserProfile struct {
	Email              string `json:"email"`
	FirstName          string `json:"firstName"`
	LastName           string `json:"lastName"`
	NationalCode       string `json:"nationalCode"`
	Nickname           string `json:"nickname"`
	NickName           string `json:"nickName"`
	Phone              string `json:"phone"`
	Mobile             string `json:"mobile"`
	Username           string `json:"username"`
	Verified           bool   `json:"verified"`
	WebsocketAuthParam string `json:"websocketAuthParam"`
	PendingWithdrawals int    `json:"pendingWithdrawals"`
}

// DisplayNickname prefers nickname / nickName.
func (p UserProfile) DisplayNickname() string {
	if s := strings.TrimSpace(p.Nickname); s != "" {
		return s
	}
	return strings.TrimSpace(p.NickName)
}

// TradeStats is the profile trade-stats object. Fields vary by account; keep
// the common numeric/string ones and ignore the rest.
type TradeStats struct {
	TotalTrades int   `json:"totalTrades"`
	MonthTrades int   `json:"monthTrades"`
	MonthVolume Money `json:"monthVolume"`
	TotalVolume Money `json:"totalVolume"`
}

// UserLimitationsResponse is POST /users/limitations.
type UserLimitationsResponse struct {
	Envelope
	Limitations UserLimitations `json:"limitations"`
}

// UserLimitations is user level plus nested capability/limit maps.
type UserLimitations struct {
	UserLevel     FlexibleString `json:"userLevel"`
	Features      map[string]any `json:"features"`
	Limits        map[string]any `json:"limits"`
	DepositLimits map[string]any `json:"depositLimits"`
}

// Wallet is one row of POST /users/wallets/list.
type Wallet struct {
	ID              int64      `json:"id"`
	Currency        string     `json:"currency"`
	Balance         Money      `json:"balance"`
	BlockedBalance  Money      `json:"blockedBalance"`
	ActiveBalance   Money      `json:"activeBalance"`
	RialBalance     JSONInt64  `json:"rialBalance"`
	RialBalanceSell JSONInt64  `json:"rialBalanceSell"`
	Type            WalletType `json:"type"`
}

// WalletListRequest is POST /users/wallets/list. Empty Type lets the API
// default to spot.
type WalletListRequest struct {
	Type WalletType `json:"type,omitempty"`
}

// Prepare lowercases type. Empty type is left empty (API default: spot).
func (r WalletListRequest) Prepare() (WalletListRequest, error) {
	out := r
	out.Type = WalletType(strings.ToLower(strings.TrimSpace(string(out.Type))))
	if !out.Type.valid() {
		return WalletListRequest{}, fmt.Errorf("types: wallet type must be spot, margin, credit, or debit")
	}
	return out, nil
}

// WalletListResponse is POST /users/wallets/list.
type WalletListResponse struct {
	Envelope
	Wallets []Wallet `json:"wallets"`
}

// WalletByCurrency returns the first wallet for currency (case-insensitive).
func (r *WalletListResponse) WalletByCurrency(currency string) (Wallet, bool) {
	if r == nil {
		return Wallet{}, false
	}
	want := strings.ToLower(strings.TrimSpace(currency))
	for _, w := range r.Wallets {
		if strings.ToLower(w.Currency) == want {
			return w, true
		}
	}
	return Wallet{}, false
}

// WalletsV2Request is POST /v2/wallets (selected currencies).
type WalletsV2Request struct {
	Currencies string     `json:"currencies,omitempty"` // comma-separated
	Type       WalletType `json:"type,omitempty"`
}

// Prepare lowercases currencies/type.
func (r WalletsV2Request) Prepare() (WalletsV2Request, error) {
	out := r
	out.Currencies = normalizeCurrencyList(out.Currencies)
	out.Type = WalletType(strings.ToLower(strings.TrimSpace(string(out.Type))))
	if !out.Type.valid() {
		return WalletsV2Request{}, fmt.Errorf("types: wallet type must be spot, margin, credit, or debit")
	}
	return out, nil
}

// WalletV2 is one selected wallet. The v2 payload is a map of currency → object.
type WalletV2 struct {
	ID      int64 `json:"id"`
	Balance Money `json:"balance"`
	Blocked Money `json:"blocked"`
}

// WalletsV2Response is POST /v2/wallets.
type WalletsV2Response struct {
	Envelope
	Wallets map[string]WalletV2 `json:"wallets"`
}

// Wallet returns the selected wallet for currency (case-insensitive).
func (r *WalletsV2Response) Wallet(currency string) (WalletV2, bool) {
	if r == nil || r.Wallets == nil {
		return WalletV2{}, false
	}
	if w, ok := r.Wallets[currency]; ok {
		return w, true
	}
	alt := strings.ToLower(strings.TrimSpace(currency))
	if w, ok := r.Wallets[alt]; ok {
		return w, true
	}
	return WalletV2{}, false
}

// WalletBalanceRequest is POST /users/wallets/balance.
type WalletBalanceRequest struct {
	Currency string `json:"currency"`
}

// Prepare lowercases currency.
func (r WalletBalanceRequest) Prepare() (WalletBalanceRequest, error) {
	out := r
	out.Currency = strings.ToLower(strings.TrimSpace(out.Currency))
	if out.Currency == "" {
		return WalletBalanceRequest{}, fmt.Errorf("types: wallet balance currency is required")
	}
	return out, nil
}

// WalletBalanceResponse is POST /users/wallets/balance.
type WalletBalanceResponse struct {
	Envelope
	Balance Money `json:"balance"`
}

// DepositListQuery is GET /users/wallets/deposits/list.
type DepositListQuery struct {
	Wallet   string // wallet id or "all" (API default all)
	Page     int
	PageSize int
}

// Normalize trims wallet. Empty wallet is omitted (API default: all).
func (q DepositListQuery) Normalize() (DepositListQuery, error) {
	out := q
	out.Wallet = strings.TrimSpace(out.Wallet)
	if out.Page < 0 || out.PageSize < 0 {
		return DepositListQuery{}, fmt.Errorf("types: deposit list page/pageSize must be >= 0")
	}
	return out, nil
}

// Deposit is one inbound transfer. Field names follow the documented list plus
// common live keys; unknown keys are ignored.
type Deposit struct {
	TxHash         string `json:"txHash"`
	Address        string `json:"address"`
	Confirmed      bool   `json:"confirmed"`
	Transaction    any    `json:"transaction"`
	Currency       string `json:"currency"`
	Amount         Money  `json:"amount"`
	WalletID       int64  `json:"wallet_id"`
	BlockchainURL  string `json:"blockchainUrl"`
	BlockchainURL2 string `json:"blockchain_url"`
}

// DepositListResponse is GET /users/wallets/deposits/list.
type DepositListResponse struct {
	Envelope
	Deposits  []Deposit  `json:"deposits"`
	Withdraws []Withdraw `json:"withdraws"`
	HasNext   bool       `json:"hasNext"`
}
