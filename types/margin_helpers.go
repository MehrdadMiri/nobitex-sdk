package types

import (
	"fmt"
	"strings"
)

// MarginMarket is one GET /margin/markets/list entry. Delegation/fee extras
// may be null on the public (unauthenticated) payload.
type MarginMarket struct {
	SrcCurrency                   string `json:"srcCurrency"`
	DstCurrency                   string `json:"dstCurrency"`
	PositionFeeRate               *Money `json:"positionFeeRate"`
	MaxLeverage                   Money  `json:"maxLeverage"`
	SellEnabled                   bool   `json:"sellEnabled"`
	BuyEnabled                    bool   `json:"buyEnabled"`
	SellPositionFeeRate           *Money `json:"sellPositionFeeRate"`
	BuyPositionFeeRate            *Money `json:"buyPositionFeeRate"`
	SellMaxDelegation             *Money `json:"sellMaxDelegation"`
	BuyMaxDelegation              *Money `json:"buyMaxDelegation"`
	BuyMaxDelegationInSrcCurrency *Money `json:"buyMaxDelegationInSrcCurrency"`
}

// MarginMarketsResponse is GET /margin/markets/list. Works without auth;
// a token can personalize maxLeverage / delegation caps.
type MarginMarketsResponse struct {
	Envelope
	Markets map[string]MarginMarket `json:"markets"`
}

// Market looks up a margin market by symbol (BTCIRT, exact then uppercased).
func (r *MarginMarketsResponse) Market(symbol string) (MarginMarket, bool) {
	if r == nil || r.Markets == nil {
		return MarginMarket{}, false
	}
	if m, ok := r.Markets[symbol]; ok {
		return m, true
	}
	alt := strings.ToUpper(strings.TrimSpace(symbol))
	if alt != symbol {
		if m, ok := r.Markets[alt]; ok {
			return m, true
		}
	}
	return MarginMarket{}, false
}

// LeverageLimit is one remaining-capacity row from GET /margin/v2/delegation-limit.
type LeverageLimit struct {
	Leverage Money `json:"leverage"`
	Limit    Money `json:"limit"`
}

// DelegationLimits is remaining buy/sell capacity per leverage step (1 … max, step 0.5).
type DelegationLimits struct {
	Buy  []LeverageLimit `json:"buy"`
	Sell []LeverageLimit `json:"sell"`
}

// DelegationLimitResponse is GET /margin/v2/delegation-limit?market=.
// Sell limit is in srcCurrency; buy limit is in dstCurrency (order notional).
type DelegationLimitResponse struct {
	Envelope
	Limits DelegationLimits `json:"limits"`
}

// LimitFor returns the remaining capacity for side + leverage (string match
// after trim). Missing rows return false.
func (r *DelegationLimitResponse) LimitFor(side OrderSide, leverage Money) (Money, bool) {
	if r == nil {
		return "", false
	}
	want := strings.TrimSpace(string(leverage))
	var rows []LeverageLimit
	switch OrderSide(strings.ToLower(strings.TrimSpace(string(side)))) {
	case OrderSideBuy:
		rows = r.Limits.Buy
	case OrderSideSell:
		rows = r.Limits.Sell
	default:
		return "", false
	}
	for _, row := range rows {
		if strings.TrimSpace(string(row.Leverage)) == want {
			return row.Limit, true
		}
	}
	return "", false
}

// WalletTransferRequest is POST /wallets/transfer (spot ↔ margin only).
type WalletTransferRequest struct {
	Currency string     `json:"currency"`
	Amount   Money      `json:"amount"`
	Src      WalletType `json:"src"`
	Dst      WalletType `json:"dst"`
}

// NewSpotToMarginTransfer moves collateral from spot into the margin wallet.
func NewSpotToMarginTransfer(currency string, amount Money) WalletTransferRequest {
	return WalletTransferRequest{
		Currency: currency,
		Amount:   amount,
		Src:      WalletSpot,
		Dst:      WalletMargin,
	}
}

// NewMarginToSpotTransfer moves collateral from the margin wallet back to spot.
func NewMarginToSpotTransfer(currency string, amount Money) WalletTransferRequest {
	return WalletTransferRequest{
		Currency: currency,
		Amount:   amount,
		Src:      WalletMargin,
		Dst:      WalletSpot,
	}
}

func transferWalletOK(t WalletType) bool {
	return t == WalletSpot || t == WalletMargin
}

// Prepare lowercases currency/src/dst and checks the documented pairing rule.
func (r WalletTransferRequest) Prepare() (WalletTransferRequest, error) {
	out := r
	out.Currency = strings.ToLower(strings.TrimSpace(out.Currency))
	out.Amount = Money(strings.TrimSpace(string(out.Amount)))
	out.Src = WalletType(strings.ToLower(strings.TrimSpace(string(out.Src))))
	out.Dst = WalletType(strings.ToLower(strings.TrimSpace(string(out.Dst))))
	if out.Currency == "" {
		return WalletTransferRequest{}, fmt.Errorf("types: margin transfer currency is required")
	}
	if out.Amount == "" {
		return WalletTransferRequest{}, fmt.Errorf("types: margin transfer amount is required")
	}
	if !transferWalletOK(out.Src) || !transferWalletOK(out.Dst) {
		return WalletTransferRequest{}, fmt.Errorf("types: margin transfer src/dst must be spot or margin")
	}
	if out.Src == out.Dst {
		return WalletTransferRequest{}, fmt.Errorf("types: margin transfer src and dst must differ")
	}
	return out, nil
}

// WalletTransferResponse is POST /wallets/transfer.
type WalletTransferResponse struct {
	Envelope
	SrcWallet Wallet `json:"srcWallet"`
	DstWallet Wallet `json:"dstWallet"`
}
