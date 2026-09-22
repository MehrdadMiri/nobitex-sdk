package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

const (
	withdrawsListPath  = "/users/wallets/withdraws/list"
	withdrawPathPrefix = "/withdraws/"
)

// ListWithdraws fetches GET /users/wallets/withdraws/list (auth, API-key READ,
// 60/2 min). Read-only — this SDK does not submit, confirm, or cancel
// withdrawals. Docs: https://apidocs.nobitex.ir
func (c *Client) ListWithdraws(ctx context.Context, q types.WithdrawListQuery) (*types.WithdrawListResponse, error) {
	if err := c.requireAuth("GET /users/wallets/withdraws/list", "READ"); err != nil {
		return nil, err
	}
	norm, err := q.Normalize()
	if err != nil {
		return nil, err
	}
	var out types.WithdrawListResponse
	opts := []RequestOption{}
	if query := encodeWithdrawListQuery(norm); len(query) > 0 {
		opts = append(opts, WithQuery(query))
	}
	if err := c.DoJSON(ctx, http.MethodGet, withdrawsListPath, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetWithdraw fetches GET /withdraws/:withdrawId (auth, API-key READ, 60/2 min).
// Read-only. Rial detail on this path is documented as soon-to-be deprecated.
func (c *Client) GetWithdraw(ctx context.Context, withdrawID int64) (*types.WithdrawResponse, error) {
	if err := c.requireAuth("GET /withdraws/:withdrawId", "READ"); err != nil {
		return nil, err
	}
	if withdrawID <= 0 {
		return nil, fmt.Errorf("client: withdraw id must be a positive integer")
	}
	var out types.WithdrawResponse
	if err := c.DoJSON(ctx, http.MethodGet, withdrawPathPrefix+strconv.FormatInt(withdrawID, 10), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func encodeWithdrawListQuery(q types.WithdrawListQuery) url.Values {
	v := url.Values{}
	if q.Wallet != "" {
		v.Set("wallet", q.Wallet)
	}
	if q.From != "" {
		v.Set("from", q.From)
	}
	if q.To != "" {
		v.Set("to", q.To)
	}
	if q.Page >= 1 {
		v.Set("page", strconv.Itoa(q.Page))
	}
	if q.PageSize >= 1 {
		v.Set("pageSize", strconv.Itoa(q.PageSize))
	}
	return v
}
