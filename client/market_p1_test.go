package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestMarketStatsPublicNoAuth(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "market_stats.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("should-not-be-sent"),
	)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.MarketStats(context.Background(), types.MarketStatsQuery{
		SrcCurrency: " BTC,ETH ",
		DstCurrency: "RLS",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.path != "/market/stats" {
		t.Fatalf("request %s %s", got.method, got.path)
	}
	if got.ua != "TraderBot/MyBot-1.0.0" {
		t.Fatalf("User-Agent = %q", got.ua)
	}
	if got.auth != "" || got.key != "" {
		t.Fatalf("public stats sent auth Authorization=%q Nobitex-Key=%q", got.auth, got.key)
	}
	q := parseQuery(got.rawQuery)
	if q["srcCurrency"] != "btc%2Ceth" && q["srcCurrency"] != "btc,eth" {
		// url.Values Encode uses + or %2C depending on encoding; WithQuery uses q.Encode()
		if !strings.Contains(got.rawQuery, "srcCurrency=btc") || !strings.Contains(got.rawQuery, "dstCurrency=rls") {
			t.Fatalf("query = %q parsed=%v", got.rawQuery, q)
		}
	}
	st, ok := resp.Stat("BTCIRT")
	if !ok || types.MoneyValue(st.Latest) != "199079999990" {
		t.Fatalf("btc-rls = %+v ok=%v", st, ok)
	}
	usdt, ok := resp.Stat("BTCUSDT")
	if !ok || types.MoneyValue(usdt.Latest) != "9750.00" {
		t.Fatalf("btc-usdt = %+v ok=%v", usdt, ok)
	}
}

func TestMarketTradesAndDepth(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "market_trades.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	tr, err := c.MarketTrades(context.Background(), " btcirt ")
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v2/trades/BTCIRT" || got.auth != "" {
		t.Fatalf("path=%q auth=%q", got.path, got.auth)
	}
	if tr.Symbol != "BTCIRT" || len(tr.Trades) != 2 || tr.Trades[0].Type != types.OrderSideSell {
		t.Fatalf("%+v", tr)
	}

	depthSrv := newP1Server(t, http.StatusOK, "market_depth.json", &got)
	defer depthSrv.Close()
	c2, err := client.New(client.WithBaseURL(depthSrv.URL))
	if err != nil {
		t.Fatal(err)
	}
	d, err := c2.MarketDepth(context.Background(), "BTCIRT")
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v2/depth/BTCIRT" {
		t.Fatalf("path=%q", got.path)
	}
	if d.LastUpdate.Int64() != 1657620166710 || d.LastTradePrice != "831" {
		t.Fatalf("%+v", d)
	}
	if len(d.Asks) != 2 || d.Asks[0].Price != "199200000000" {
		t.Fatalf("asks=%+v", d.Asks)
	}
}

func TestMarketTradesRejectsAll(t *testing.T) {
	t.Parallel()
	c, err := client.New(client.WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.MarketTrades(context.Background(), "all"); err == nil || !strings.Contains(err.Error(), "all") {
		t.Fatalf("err=%v", err)
	}
	if _, err := c.MarketDepth(context.Background(), "ALL"); err == nil {
		t.Fatal("expected error")
	}
}

func TestMarketOHLC(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "udf_history.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.MarketOHLC(context.Background(), types.UDFHistoryQuery{
		Symbol:     "btcirt",
		Resolution: "D",
		From:       1562058167,
		To:         1562230967,
		Countback:  4,
		Page:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/market/udf/history" || got.auth != "" {
		t.Fatalf("path=%q auth=%q", got.path, got.auth)
	}
	q := parseQuery(got.rawQuery)
	if q["symbol"] != "BTCIRT" || q["resolution"] != "D" || q["to"] != "1562230967" {
		t.Fatalf("query=%q parsed=%v", got.rawQuery, q)
	}
	if q["countback"] != "4" || q["from"] != "1562058167" {
		t.Fatalf("query extras=%v", q)
	}
	cs := h.Candles()
	if !h.IsOK() || len(cs) != 2 || cs[0].Open != "146272500" || cs[1].Volume != "9.8592626506" {
		t.Fatalf("history=%+v candles=%+v", h, cs)
	}
}

func TestMarketOHLCNoDataAndError(t *testing.T) {
	t.Parallel()
	empty := newP1Server(t, http.StatusOK, "udf_nodata.json", nil)
	defer empty.Close()
	c, err := client.New(client.WithBaseURL(empty.URL))
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.MarketOHLC(context.Background(), types.UDFHistoryQuery{Symbol: "BTCIRT", Resolution: "D", To: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !h.HasNoData() || len(h.Candles()) != 0 {
		t.Fatalf("%+v", h)
	}

	bad := newP1Server(t, http.StatusOK, "udf_error.json", nil)
	defer bad.Close()
	c2, err := client.New(client.WithBaseURL(bad.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c2.MarketOHLC(context.Background(), types.UDFHistoryQuery{Symbol: "BTCIRT", Resolution: "BAD", To: 1})
	if err == nil || !sdkerr.IsAPI(err) {
		t.Fatalf("err=%v", err)
	}
	var e *sdkerr.Error
	if !sdkerr.As(err, &e) || e.Code != "UDFError" || !strings.Contains(e.Message, "Invalid resolution") {
		t.Fatalf("api err=%+v", e)
	}
}

func TestMarketTradesInvalidSymbol(t *testing.T) {
	t.Parallel()
	srv := newP1Server(t, http.StatusOK, "orderbook_invalid.json", nil)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.MarketTrades(context.Background(), "NOPE")
	if err == nil || !sdkerr.IsAPI(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestMarketDataLivePublic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live public call")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c, err := client.New(
		client.WithApp("nobitex-sdk-test", "0.1.0"),
		client.WithTimeout(12*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	stats, err := c.MarketStats(ctx, types.MarketStatsQuery{SrcCurrency: "btc", DstCurrency: "rls"})
	if err != nil {
		if sdkerr.IsTransport(err) || sdkerr.IsRateLimited(err) {
			t.Skipf("live stats unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if !stats.Status.IsOK() {
		t.Fatalf("stats status=%q", stats.Status)
	}
	if _, ok := stats.Stat("BTCIRT"); !ok {
		t.Fatalf("missing btc-rls in %+v", stats.Stats)
	}

	tr, err := c.MarketTrades(ctx, "BTCIRT")
	if err != nil {
		if sdkerr.IsTransport(err) || sdkerr.IsRateLimited(err) {
			t.Skipf("live trades unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if !tr.Status.IsOK() || len(tr.Trades) == 0 {
		t.Fatalf("trades=%+v", tr)
	}

	d, err := c.MarketDepth(ctx, "BTCIRT")
	if err != nil {
		if sdkerr.IsTransport(err) || sdkerr.IsRateLimited(err) {
			t.Skipf("live depth unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if !d.Status.IsOK() || len(d.Asks) == 0 || len(d.Bids) == 0 {
		t.Fatalf("depth=%+v", d)
	}

	now := time.Now().Unix()
	h, err := c.MarketOHLC(ctx, types.UDFHistoryQuery{
		Symbol:     "BTCIRT",
		Resolution: types.ResolutionD,
		To:         now,
		Countback:  2,
	})
	if err != nil {
		if sdkerr.IsTransport(err) || sdkerr.IsRateLimited(err) {
			t.Skipf("live ohlc unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if !h.IsOK() && !h.HasNoData() {
		t.Fatalf("ohlc s=%q", h.S)
	}

	mk, err := c.MarginMarkets(ctx, false)
	if err != nil {
		if sdkerr.IsTransport(err) || sdkerr.IsRateLimited(err) {
			t.Skipf("live margin markets unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if !mk.Status.IsOK() {
		t.Fatalf("markets status=%q", mk.Status)
	}
	if _, ok := mk.Market("BTCIRT"); !ok {
		t.Fatal("live margin markets missing BTCIRT")
	}
}

func TestMarketP1FixturesPresent(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"market_stats.json", "market_trades.json", "market_depth.json",
		"udf_history.json", "udf_nodata.json", "udf_error.json",
	} {
		b := testdata(t, name)
		if len(b) == 0 {
			t.Fatalf("empty %s", name)
		}
		var raw map[string]any
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
