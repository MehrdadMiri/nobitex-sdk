package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

const (
	usersProfilePath     = "/users/profile"
	usersLimitationsPath = "/users/limitations"
	walletsListPath      = "/users/wallets/list"
	walletsV2Path        = "/v2/wallets"
	walletBalancePath    = "/users/wallets/balance"
	depositsListPath     = "/users/wallets/deposits/list"
)

// UserProfile fetches GET /users/profile (auth, API-key READ).
// profile.websocketAuthParam is required to name private WS channels.
func (c *Client) UserProfile(ctx context.Context) (*types.UserProfileResponse, error) {
	if err := c.requireAuth("GET /users/profile", "READ"); err != nil {
		return nil, err
	}
	var out types.UserProfileResponse
	if err := c.DoJSON(ctx, http.MethodGet, usersProfilePath, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UserLimitations fetches POST /users/limitations (auth, API-key READ).
func (c *Client) UserLimitations(ctx context.Context) (*types.UserLimitationsResponse, error) {
	if err := c.requireAuth("POST /users/limitations", "READ"); err != nil {
		return nil, err
	}
	var out types.UserLimitationsResponse
	if err := c.DoJSON(ctx, http.MethodPost, usersLimitationsPath, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListWallets fetches POST /users/wallets/list (auth, API-key READ, 20/min).
// Empty Type lets the API default to spot.
func (c *Client) ListWallets(ctx context.Context, req types.WalletListRequest) (*types.WalletListResponse, error) {
	if err := c.requireAuth("POST /users/wallets/list", "READ"); err != nil {
		return nil, err
	}
	wire, err := req.Prepare()
	if err != nil {
		return nil, err
	}
	var out types.WalletListResponse
	opts := []RequestOption{}
	if wire.Type != "" {
		opts = append(opts, WithJSONBody(wire))
	}
	if err := c.DoJSON(ctx, http.MethodPost, walletsListPath, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListWalletsV2 fetches POST /v2/wallets (auth, API-key READ, 15/min).
func (c *Client) ListWalletsV2(ctx context.Context, req types.WalletsV2Request) (*types.WalletsV2Response, error) {
	if err := c.requireAuth("POST /v2/wallets", "READ"); err != nil {
		return nil, err
	}
	wire, err := req.Prepare()
	if err != nil {
		return nil, err
	}
	var out types.WalletsV2Response
	if err := c.DoJSON(ctx, http.MethodPost, walletsV2Path, &out, WithJSONBody(wire)); err != nil {
		return nil, err
	}
	return &out, nil
}

// WalletBalance fetches POST /users/wallets/balance (auth, API-key READ, 60/2min).
func (c *Client) WalletBalance(ctx context.Context, currency string) (*types.WalletBalanceResponse, error) {
	if err := c.requireAuth("POST /users/wallets/balance", "READ"); err != nil {
		return nil, err
	}
	wire, err := (types.WalletBalanceRequest{Currency: currency}).Prepare()
	if err != nil {
		return nil, err
	}
	var out types.WalletBalanceResponse
	if err := c.DoJSON(ctx, http.MethodPost, walletBalancePath, &out, WithJSONBody(wire)); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListDeposits fetches GET /users/wallets/deposits/list (auth, API-key READ).
func (c *Client) ListDeposits(ctx context.Context, q types.DepositListQuery) (*types.DepositListResponse, error) {
	if err := c.requireAuth("GET /users/wallets/deposits/list", "READ"); err != nil {
		return nil, err
	}
	norm, err := q.Normalize()
	if err != nil {
		return nil, err
	}
	var out types.DepositListResponse
	opts := []RequestOption{}
	if query := encodeDepositListQuery(norm); len(query) > 0 {
		opts = append(opts, WithQuery(query))
	}
	if err := c.DoJSON(ctx, http.MethodGet, depositsListPath, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

func encodeDepositListQuery(q types.DepositListQuery) url.Values {
	v := url.Values{}
	if q.Wallet != "" {
		v.Set("wallet", q.Wallet)
	}
	if q.Page >= 1 {
		v.Set("page", strconv.Itoa(q.Page))
	}
	if q.PageSize >= 1 {
		v.Set("pageSize", strconv.Itoa(q.PageSize))
	}
	return v
}
