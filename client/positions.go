package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

const positionsListPath = "/positions/list"

// ListPositions fetches GET /positions/list (auth required).
//
// Documented query filters: srcCurrency, dstCurrency, status (active|past),
// page, pageSize (default 50). Empty query uses the API default of active
// positions. Rate limit: 30 requests / 10 minutes. API keys need TRADE.
//
// Docs: https://apidocs.nobitex.ir
func (c *Client) ListPositions(ctx context.Context, q types.PositionListQuery) (*types.PositionListResponse, error) {
	if err := c.requireAuth("GET /positions/list"); err != nil {
		return nil, err
	}
	norm, err := q.Normalize()
	if err != nil {
		return nil, err
	}
	var out types.PositionListResponse
	opts := []RequestOption{}
	if query := encodePositionListQuery(norm); len(query) > 0 {
		opts = append(opts, WithQuery(query))
	}
	if err := c.DoJSON(ctx, http.MethodGet, positionsListPath, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ClosePosition places an opposite-side close via POST /positions/:positionId/close
// (auth required). Sell positions close with a margin buy; buy positions close
// with a margin sell. Executions: limit, market, stop_limit, stop_market, oco
// (oco is sent as execution=limit + mode=oco). Rate limit: 300 / 10 minutes.
// API keys need TRADE.
//
// Docs: https://apidocs.nobitex.ir
func (c *Client) ClosePosition(ctx context.Context, positionID int64, req types.ClosePositionRequest) (*types.ClosePositionResponse, error) {
	if err := c.requireAuth("POST /positions/:positionId/close"); err != nil {
		return nil, err
	}
	if positionID <= 0 {
		return nil, fmt.Errorf("client: position id must be a positive integer")
	}
	wire, err := req.Prepare()
	if err != nil {
		return nil, err
	}
	var out types.ClosePositionResponse
	if err := c.DoJSON(ctx, http.MethodPost, closePositionPath(positionID), &out, WithJSONBody(wire)); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) requireAuth(op string) error {
	if c == nil {
		return fmt.Errorf("client: not initialized")
	}
	if c.Auth() == nil {
		return fmt.Errorf("client: %s requires Token or API-key (TRADE) authentication", op)
	}
	return nil
}

func closePositionPath(positionID int64) string {
	return "/positions/" + strconv.FormatInt(positionID, 10) + "/close"
}

func encodePositionListQuery(q types.PositionListQuery) url.Values {
	v := url.Values{}
	if q.SrcCurrency != "" {
		v.Set("srcCurrency", q.SrcCurrency)
	}
	if q.DstCurrency != "" {
		v.Set("dstCurrency", q.DstCurrency)
	}
	if q.Status != "" {
		v.Set("status", string(q.Status))
	}
	if q.Page >= 1 {
		v.Set("page", strconv.Itoa(q.Page))
	}
	if q.PageSize >= 1 {
		v.Set("pageSize", strconv.Itoa(q.PageSize))
	}
	return v
}
