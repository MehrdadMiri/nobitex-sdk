package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

const (
	marginMarketsPath         = "/margin/markets/list"
	marginDelegationLimitPath = "/margin/v2/delegation-limit"
	walletsTransferPath       = "/wallets/transfer"
)

// MarginMarkets fetches GET /margin/markets/list (30/min).
// Public without a token; a configured authenticator is sent so the payload
// can include user-specific maxLeverage / delegation caps. Pass details=true
// to request the documented extra fields as a JSON body.
func (c *Client) MarginMarkets(ctx context.Context, details bool) (*types.MarginMarketsResponse, error) {
	var out types.MarginMarketsResponse
	opts := []RequestOption{}
	if details {
		opts = append(opts, WithJSONBody(map[string]bool{"details": true}))
	}
	if c == nil || c.Auth() == nil {
		opts = append(opts, WithoutAuth())
	}
	if err := c.DoJSON(ctx, http.MethodGet, marginMarketsPath, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// MarginDelegationLimit fetches GET /margin/v2/delegation-limit (auth, API-key
// READ, 12/min). Remaining buy/sell capacity is returned per leverage step.
func (c *Client) MarginDelegationLimit(ctx context.Context, market string) (*types.DelegationLimitResponse, error) {
	if err := c.requireAuth("GET /margin/v2/delegation-limit", "READ"); err != nil {
		return nil, err
	}
	m := strings.ToUpper(strings.TrimSpace(market))
	if m == "" {
		return nil, fmt.Errorf("client: margin delegation market is required")
	}
	var out types.DelegationLimitResponse
	q := url.Values{}
	q.Set("market", m)
	if err := c.DoJSON(ctx, http.MethodGet, marginDelegationLimitPath, &out, WithQuery(q)); err != nil {
		return nil, err
	}
	return &out, nil
}

// TransferWallet moves funds between spot and margin via POST /wallets/transfer
// (auth, 10/min). Src and dst must differ and be spot|margin.
func (c *Client) TransferWallet(ctx context.Context, req types.WalletTransferRequest) (*types.WalletTransferResponse, error) {
	if err := c.requireAuth("POST /wallets/transfer", "TRADE"); err != nil {
		return nil, err
	}
	wire, err := req.Prepare()
	if err != nil {
		return nil, err
	}
	var out types.WalletTransferResponse
	if err := c.DoJSON(ctx, http.MethodPost, walletsTransferPath, &out, WithJSONBody(wire)); err != nil {
		return nil, err
	}
	return &out, nil
}

// TransferSpotToMargin is TransferWallet from spot to margin.
func (c *Client) TransferSpotToMargin(ctx context.Context, currency string, amount types.Money) (*types.WalletTransferResponse, error) {
	return c.TransferWallet(ctx, types.NewSpotToMarginTransfer(currency, amount))
}

// TransferMarginToSpot is TransferWallet from margin to spot.
func (c *Client) TransferMarginToSpot(ctx context.Context, currency string, amount types.Money) (*types.WalletTransferResponse, error) {
	return c.TransferWallet(ctx, types.NewMarginToSpotTransfer(currency, amount))
}
