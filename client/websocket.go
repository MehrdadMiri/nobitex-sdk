package client

import (
	"context"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

// WebSocketOverview returns the documented Centrifugo connection URLs and
// channel list. This is an overview stub — it does not open a socket or run
// a trading bot. Docs: https://apidocs.nobitex.ir/websocket/%D9%88%D8%A8-%D8%B3%D9%88%DA%A9%D8%AA
func WebSocketOverview() types.WebSocketOverview {
	return types.DefaultWebSocketOverview()
}

// WebSocketToken fetches GET /auth/ws/token/ (auth, API-key READ).
// The JWT authenticates a Centrifugo connection for private channels and
// lasts 1200 seconds. websocketAuthParam comes from UserProfile, not here.
func (c *Client) WebSocketToken(ctx context.Context) (*types.WebSocketTokenResponse, error) {
	if err := c.requireAuth("GET /auth/ws/token/", "READ"); err != nil {
		return nil, err
	}
	var out types.WebSocketTokenResponse
	if err := c.DoJSON(ctx, "GET", types.WebSocketTokenPath, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
