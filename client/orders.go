package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

const (
	marketOrdersListPath         = "/market/orders/list"
	marketOrdersUpdateStatusPath = "/market/orders/update-status"
)

// ListOrders fetches GET /market/orders/list (auth required).
//
// Documented query filters include status, type, execution, tradeType
// (spot|margin), srcCurrency, dstCurrency, details, fromId, order (sort),
// page, and pageSize. Set TradeType to types.TradeTypeFilterMargin — or call
// ListMarginOrders — to list margin (تعهدی) orders. API keys need READ.
// Rate limit: 30/min. details=2 includes id/status so callers can cancel.
//
// Docs: https://apidocs.nobitex.ir
func (c *Client) ListOrders(ctx context.Context, q types.OrderListQuery) (*types.OrderListResponse, error) {
	return c.listOrders(ctx, http.MethodGet, q)
}

// ListOrdersPost is the POST form of /market/orders/list. Filters are sent as
// a JSON body (same field names as the GET query). GET is the documented
// method; POST is provided for clients that post the same payload.
//
// Docs: https://apidocs.nobitex.ir
func (c *Client) ListOrdersPost(ctx context.Context, q types.OrderListQuery) (*types.OrderListResponse, error) {
	return c.listOrders(ctx, http.MethodPost, q)
}

// ListMarginOrders is ListOrders with tradeType=margin. Other query fields
// are kept. API keys need READ.
func (c *Client) ListMarginOrders(ctx context.Context, q types.OrderListQuery) (*types.OrderListResponse, error) {
	return c.ListOrders(ctx, q.WithMargin())
}

// ListMarginOrdersPost is ListOrdersPost with tradeType=margin.
func (c *Client) ListMarginOrdersPost(ctx context.Context, q types.OrderListQuery) (*types.OrderListResponse, error) {
	return c.ListOrdersPost(ctx, q.WithMargin())
}

func (c *Client) listOrders(ctx context.Context, method string, q types.OrderListQuery) (*types.OrderListResponse, error) {
	if err := c.requireAuth(method+" "+marketOrdersListPath, "READ"); err != nil {
		return nil, err
	}
	norm, err := q.Prepare()
	if err != nil {
		return nil, err
	}
	var opts []RequestOption
	if method == http.MethodPost {
		opts = append(opts, WithJSONBody(norm))
	} else if query := encodeOrderListQuery(norm); len(query) > 0 {
		opts = append(opts, WithQuery(query))
	}
	var out types.OrderListResponse
	if err := c.DoJSON(ctx, method, marketOrdersListPath, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelOrder posts status=canceled to POST /market/orders/update-status.
//
// Supply order id and/or clientOrderId (at least one). If both are sent, the
// server prefers order. Search by clientOrderId only covers open orders.
// API keys need the TRADE permission
// (https://apidocs.nobitex.ir/api_key/api-key-guide). Rate limit: 90/min.
//
// Docs: https://apidocs.nobitex.ir
func (c *Client) CancelOrder(ctx context.Context, req types.CancelOrderRequest) (*types.CancelOrderResponse, error) {
	if err := c.requireAuth("POST "+marketOrdersUpdateStatusPath, "TRADE"); err != nil {
		return nil, err
	}
	wire, err := req.Prepare()
	if err != nil {
		return nil, err
	}
	var out types.CancelOrderResponse
	if err := c.DoJSON(ctx, http.MethodPost, marketOrdersUpdateStatusPath, &out, WithJSONBody(wire)); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelOrderByID cancels via POST /market/orders/update-status with the
// server order id. API keys need TRADE.
func (c *Client) CancelOrderByID(ctx context.Context, orderID int64) (*types.CancelOrderResponse, error) {
	return c.CancelOrder(ctx, types.NewCancelOrderByID(orderID))
}

// CancelOrderByClientOrderID cancels via POST /market/orders/update-status
// with clientOrderId (open orders only). API keys need TRADE.
func (c *Client) CancelOrderByClientOrderID(ctx context.Context, clientOrderID string) (*types.CancelOrderResponse, error) {
	return c.CancelOrder(ctx, types.NewCancelOrderByClientOrderID(clientOrderID))
}

func encodeOrderListQuery(q types.OrderListQuery) url.Values {
	v := url.Values{}
	if q.Status != "" {
		v.Set("status", string(q.Status))
	}
	if q.Type != "" {
		v.Set("type", string(q.Type))
	}
	if q.Execution != "" {
		v.Set("execution", string(q.Execution))
	}
	if q.TradeType != "" {
		v.Set("tradeType", string(q.TradeType))
	}
	if q.SrcCurrency != "" {
		v.Set("srcCurrency", q.SrcCurrency)
	}
	if q.DstCurrency != "" {
		v.Set("dstCurrency", q.DstCurrency)
	}
	if q.Details != 0 {
		v.Set("details", strconv.Itoa(q.Details))
	}
	if q.FromID >= 1 {
		v.Set("fromId", strconv.FormatInt(q.FromID, 10))
	}
	if q.Sort != "" {
		v.Set("order", q.Sort)
	}
	if q.Page >= 1 {
		v.Set("page", strconv.Itoa(q.Page))
	}
	if q.PageSize >= 1 {
		v.Set("pageSize", strconv.Itoa(q.PageSize))
	}
	return v
}
