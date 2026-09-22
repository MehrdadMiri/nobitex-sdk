package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

const (
	marketStatsPath    = "/market/stats"
	marketTradesPrefix = "/v2/trades/"
	marketDepthPrefix  = "/v2/depth/"
	marketUDFPath      = "/market/udf/history"
)

// MarketStats fetches GET /market/stats (public, no token, 20 req/min).
// Omit both currencies for every market. Docs: https://apidocs.nobitex.ir
func (c *Client) MarketStats(ctx context.Context, q types.MarketStatsQuery) (*types.MarketStatsResponse, error) {
	norm := q.Normalize()
	var out types.MarketStatsResponse
	opts := []RequestOption{WithoutAuth()}
	if query := encodeMarketStatsQuery(norm); len(query) > 0 {
		opts = append(opts, WithQuery(query))
	}
	if err := c.DoJSON(ctx, http.MethodGet, marketStatsPath, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// MarketTrades fetches GET /v2/trades/:symbol (public, no token, 60 req/min).
// symbol=all is not supported. At most 20 recent trades.
func (c *Client) MarketTrades(ctx context.Context, symbol string) (*types.MarketTradesResponse, error) {
	s, err := normalizeMarketSymbol(symbol, "trades")
	if err != nil {
		return nil, err
	}
	var out types.MarketTradesResponse
	if err := c.DoJSON(ctx, http.MethodGet, marketTradesPrefix+url.PathEscape(s), &out, WithoutAuth()); err != nil {
		return nil, err
	}
	out.Symbol = s
	return &out, nil
}

// MarketDepth fetches GET /v2/depth/:symbol (public, no token, 300 req/min).
// symbol=all is not supported.
func (c *Client) MarketDepth(ctx context.Context, symbol string) (*types.MarketDepth, error) {
	s, err := normalizeMarketSymbol(symbol, "depth")
	if err != nil {
		return nil, err
	}
	var out types.MarketDepth
	if err := c.DoJSON(ctx, http.MethodGet, marketDepthPrefix+url.PathEscape(s), &out, WithoutAuth()); err != nil {
		return nil, err
	}
	out.Symbol = s
	return &out, nil
}

// MarketOHLC fetches GET /market/udf/history (public, no token, 60 req/min,
// max 500 candles). UDF uses `s` rather than `status`; s=error is mapped to a
// typed API error. s=no_data is a successful empty payload.
func (c *Client) MarketOHLC(ctx context.Context, q types.UDFHistoryQuery) (*types.UDFHistory, error) {
	norm, err := q.Normalize()
	if err != nil {
		return nil, err
	}
	var out types.UDFHistory
	if err := c.DoJSON(ctx, http.MethodGet, marketUDFPath, &out, WithoutAuth(), WithQuery(encodeUDFQuery(norm))); err != nil {
		return nil, err
	}
	if out.IsError() {
		msg := strings.TrimSpace(out.ErrMsg)
		if msg == "" {
			msg = "udf error"
		}
		return &out, &sdkerr.Error{
			Kind:    sdkerr.KindAPI,
			Code:    "UDFError",
			Message: msg,
			Status:  types.StatusFailed,
		}
	}
	return &out, nil
}

func normalizeMarketSymbol(symbol, op string) (string, error) {
	s := strings.TrimSpace(symbol)
	if s == "" {
		return "", fmt.Errorf("client: %s symbol is required", op)
	}
	if strings.EqualFold(s, SymbolAll) {
		return "", fmt.Errorf("client: %s does not support symbol %q", op, SymbolAll)
	}
	return strings.ToUpper(s), nil
}

func encodeMarketStatsQuery(q types.MarketStatsQuery) url.Values {
	v := url.Values{}
	if q.SrcCurrency != "" {
		v.Set("srcCurrency", q.SrcCurrency)
	}
	if q.DstCurrency != "" {
		v.Set("dstCurrency", q.DstCurrency)
	}
	return v
}

func encodeUDFQuery(q types.UDFHistoryQuery) url.Values {
	v := url.Values{}
	v.Set("symbol", q.Symbol)
	v.Set("resolution", q.Resolution)
	if q.To != 0 {
		v.Set("to", strconv.FormatInt(q.To, 10))
	}
	if q.From != 0 {
		v.Set("from", strconv.FormatInt(q.From, 10))
	}
	if q.Countback > 0 {
		v.Set("countback", strconv.Itoa(q.Countback))
	}
	if q.Page >= 1 {
		v.Set("page", strconv.Itoa(q.Page))
	}
	return v
}
