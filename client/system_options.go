package client

import (
	"context"
	"net/http"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

const systemOptionsPath = "/v2/options"

// SystemOptions fetches GET /v2/options (public, no token) and returns system
// settings including market amountPrecisions and pricePrecisions.
//
// Those maps are the smallest allowed order amount / price increments per
// market (keys like BTCIRT, BTCUSDT). Use types.SystemOptions helpers
// (ValidateOrderDecimals, AmountPrecision, PricePrecision) from later
// order-placement code.
//
// Docs: https://apidocs.nobitex.ir/options/get-system-options
func (c *Client) SystemOptions(ctx context.Context) (*types.SystemOptions, error) {
	var out types.SystemOptions
	if err := c.DoJSON(ctx, http.MethodGet, systemOptionsPath, &out, WithoutAuth()); err != nil {
		return nil, err
	}
	return &out, nil
}
