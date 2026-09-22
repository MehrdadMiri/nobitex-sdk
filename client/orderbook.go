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
	// SymbolAll is the documented path parameter that returns every market's book.
	// It is case-sensitive on the wire ("all", not "ALL").
	SymbolAll = "all"

	orderBookPathPrefix = "/v3/orderbook/"
)

// OrderBook fetches GET /v3/orderbook/:symbol for one market (public, no token).
// symbol is trimmed and uppercased (BTCIRT). Pass a specific pair; for every
// market use OrderBookAll. Rate limit: 300/min.
//
// Docs: https://apidocs.nobitex.ir
func (c *Client) OrderBook(ctx context.Context, symbol string) (*types.OrderBook, error) {
	s, err := normalizeOrderBookSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if s == SymbolAll {
		return nil, fmt.Errorf("client: use OrderBookAll for symbol %q", SymbolAll)
	}
	var out types.OrderBook
	if err := c.DoJSON(ctx, http.MethodGet, orderBookPath(s), &out, WithoutAuth()); err != nil {
		return nil, err
	}
	out.Symbol = s
	return &out, nil
}

// OrderBookAll fetches GET /v3/orderbook/all (public, no token) and returns
// consolidated books keyed by market symbol.
//
// Docs: https://apidocs.nobitex.ir
func (c *Client) OrderBookAll(ctx context.Context) (*types.OrderBookAll, error) {
	var out types.OrderBookAll
	if err := c.DoJSON(ctx, http.MethodGet, orderBookPath(SymbolAll), &out, WithoutAuth()); err != nil {
		return nil, err
	}
	return &out, nil
}

func normalizeOrderBookSymbol(symbol string) (string, error) {
	s := strings.TrimSpace(symbol)
	if s == "" {
		return "", fmt.Errorf("client: orderbook symbol is required")
	}
	if strings.EqualFold(s, SymbolAll) {
		return SymbolAll, nil
	}
	return strings.ToUpper(s), nil
}

func orderBookPath(symbol string) string {
	return orderBookPathPrefix + url.PathEscape(symbol)
}
