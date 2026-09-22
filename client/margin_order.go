package client

import (
	"context"
	"net/http"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

const marginOrdersAddPath = "/margin/orders/add"

// AddMarginOrder places a margin (تعهدی) order via POST /margin/orders/add.
//
// Auth is required: Authorization Token and/or API-key signing. Keys must have
// the TRADE permission (https://apidocs.nobitex.ir/api_key/api-key-guide).
// Execution types: limit, market, stop_limit, stop_market, and oco
// (oco is sent as execution=limit + mode=oco). Rate limit: 300/10 min, shared
// with spot order placement.
//
// The response includes order id and clientOrderId (and pairId for OCO) for
// later cancel/list. This method does not hit the live API from tests; callers
// must supply credentials from env/config.
//
// Docs: https://apidocs.nobitex.ir
func (c *Client) AddMarginOrder(ctx context.Context, req types.MarginOrderRequest) (*types.MarginOrderAddResponse, error) {
	if err := c.requireAuth("POST /margin/orders/add", "TRADE"); err != nil {
		return nil, err
	}
	wire, err := req.Prepare()
	if err != nil {
		return nil, err
	}
	var out types.MarginOrderAddResponse
	if err := c.DoJSON(ctx, http.MethodPost, marginOrdersAddPath, &out, WithJSONBody(wire)); err != nil {
		return nil, err
	}
	return &out, nil
}
