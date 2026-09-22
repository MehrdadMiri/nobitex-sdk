package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

const (
	spotOrdersAddPath   = "/market/orders/add"
	spotOrderStatusPath = "/market/orders/status"
	userTradesListPath  = "/market/trades/list"
)

// AddSpotOrder places a spot order via POST /market/orders/add (auth, API-key
// TRADE, 300/10 min, shared with margin placement).
//
// This is a spot helper: it does not list or cancel orders (Kanban #6).
// Docs: https://apidocs.nobitex.ir/spot_trade/%D8%AB%D8%A8%D8%AA-%D8%B3%D9%81%D8%A7%D8%B1%D8%B4-%D8%AC%D8%AF%DB%8C%D8%AF-%D8%A7%D8%B3%D9%BE%D8%A7%D8%AA
func (c *Client) AddSpotOrder(ctx context.Context, req types.SpotOrderRequest) (*types.SpotOrderAddResponse, error) {
	if err := c.requireAuth("POST /market/orders/add", "TRADE"); err != nil {
		return nil, err
	}
	wire, err := req.Prepare()
	if err != nil {
		return nil, err
	}
	var out types.SpotOrderAddResponse
	if err := c.DoJSON(ctx, http.MethodPost, spotOrdersAddPath, &out, WithJSONBody(wire)); err != nil {
		return nil, err
	}
	return &out, nil
}

// SpotOrderStatus fetches POST /market/orders/status (auth, API-key READ, 300/min).
// Lookup by server id or clientOrderId (open orders only for clientOrderId).
func (c *Client) SpotOrderStatus(ctx context.Context, req types.SpotOrderStatusRequest) (*types.SpotOrderStatusResponse, error) {
	if err := c.requireAuth("POST /market/orders/status", "READ"); err != nil {
		return nil, err
	}
	wire, err := req.Prepare()
	if err != nil {
		return nil, err
	}
	var out types.SpotOrderStatusResponse
	if err := c.DoJSON(ctx, http.MethodPost, spotOrderStatusPath, &out, WithJSONBody(wire)); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListUserTrades fetches GET /market/trades/list (auth, API-key READ, 30/min).
// Spot fills from the last 180 days. srcCurrency/dstCurrency must be paired.
func (c *Client) ListUserTrades(ctx context.Context, q types.UserTradesQuery) (*types.UserTradesResponse, error) {
	if err := c.requireAuth("GET /market/trades/list", "READ"); err != nil {
		return nil, err
	}
	norm, err := q.Normalize()
	if err != nil {
		return nil, err
	}
	var out types.UserTradesResponse
	opts := []RequestOption{}
	if query := encodeUserTradesQuery(norm); len(query) > 0 {
		opts = append(opts, WithQuery(query))
	}
	if err := c.DoJSON(ctx, http.MethodGet, userTradesListPath, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

func encodeUserTradesQuery(q types.UserTradesQuery) url.Values {
	v := url.Values{}
	if q.SrcCurrency != "" {
		v.Set("srcCurrency", q.SrcCurrency)
	}
	if q.DstCurrency != "" {
		v.Set("dstCurrency", q.DstCurrency)
	}
	if q.FromID >= 1 {
		v.Set("fromId", strconv.FormatInt(q.FromID, 10))
	}
	if q.TradeType != "" {
		v.Set("tradeType", string(q.TradeType))
	}
	if q.TradeOrder != "" {
		v.Set("tradeOrder", q.TradeOrder)
	}
	if q.Page >= 1 {
		v.Set("page", strconv.Itoa(q.Page))
	}
	if q.PageSize >= 1 {
		v.Set("pageSize", strconv.Itoa(q.PageSize))
	}
	return v
}
