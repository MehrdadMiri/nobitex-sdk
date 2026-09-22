package client_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestAddSpotOrderLimitTokenAuth(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "spot_order_limit.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("secret-token"),
	)
	if err != nil {
		t.Fatal(err)
	}
	req := types.NewSpotLimitOrder(types.OrderSideBuy, "BTC", "RLS", "0.6", "520000000")
	req.ClientOrderID = "order1"
	resp, err := c.AddSpotOrder(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPost || got.path != "/market/orders/add" {
		t.Fatalf("%s %s", got.method, got.path)
	}
	if got.ua != "TraderBot/MyBot-1.0.0" || got.auth != "Token secret-token" {
		t.Fatalf("ua=%q auth=%q", got.ua, got.auth)
	}
	var body map[string]any
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["execution"] != "limit" || body["srcCurrency"] != "btc" || body["type"] != "buy" {
		t.Fatalf("body=%v", body)
	}
	if body["mode"] != nil {
		t.Fatalf("unexpected mode=%v", body["mode"])
	}
	ids := resp.OrderIDs()
	if len(ids) != 1 || ids[0] != 25 || resp.PlacedOrders()[0].TradeType != types.TradeTypeSpot {
		t.Fatalf("%+v", resp)
	}
	if resp.PlacedOrders()[0].ClientOrderIDValue() != "order1" {
		t.Fatalf("clientOrderId=%q", resp.PlacedOrders()[0].ClientOrderIDValue())
	}
}

func TestAddSpotOrderOCO(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "spot_order_oco.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.AddSpotOrder(context.Background(), types.NewSpotOCOOrder(
		types.OrderSideBuy, "btc", "usdt", "0.01", "42390", "42700", "42715",
	))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["execution"] != "limit" || body["mode"] != "oco" {
		t.Fatalf("body=%v", body)
	}
	if len(resp.OrderIDs()) != 2 || resp.OrderIDs()[0] != 27 {
		t.Fatalf("ids=%v", resp.OrderIDs())
	}
}

func TestAddSpotOrderFailedAndRequiresAuth(t *testing.T) {
	t.Parallel()
	srv := newP1Server(t, http.StatusOK, "spot_order_failed.json", nil)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.AddSpotOrder(context.Background(), types.NewSpotLimitOrder(types.OrderSideBuy, "btc", "rls", "0.6", "1"))
	if err == nil || !sdkerr.IsAPI(err) {
		t.Fatalf("err=%v", err)
	}

	plain, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = plain.AddSpotOrder(context.Background(), types.NewSpotLimitOrder(types.OrderSideBuy, "btc", "rls", "0.6", "1"))
	if err == nil || !strings.Contains(err.Error(), "TRADE") {
		t.Fatalf("err=%v", err)
	}
}

func TestSpotOrderStatusAndUserTrades(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "spot_order_status.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := c.SpotOrderStatus(context.Background(), types.SpotOrderStatusRequest{ClientOrderID: "order1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/market/orders/status" {
		t.Fatalf("path=%q", got.path)
	}
	var body map[string]any
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["clientOrderId"] != "order1" {
		t.Fatalf("body=%v", body)
	}
	if st.Order == nil || st.Order.ID != 5684 || st.Order.Status != types.OrderStatusActive {
		t.Fatalf("%+v", st.Order)
	}

	trSrv := newP1Server(t, http.StatusOK, "user_trades.json", &got)
	defer trSrv.Close()
	c2, err := client.New(client.WithBaseURL(trSrv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	tr, err := c2.ListUserTrades(context.Background(), types.UserTradesQuery{
		SrcCurrency: "USDT",
		DstCurrency: "rls",
		TradeType:   types.OrderSideSell,
		Page:        1,
		PageSize:    30,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/market/trades/list" || got.method != http.MethodGet {
		t.Fatalf("%s %s", got.method, got.path)
	}
	q := parseQuery(got.rawQuery)
	if q["srcCurrency"] != "usdt" || q["dstCurrency"] != "rls" || q["tradeType"] != "sell" {
		t.Fatalf("query=%v", q)
	}
	if len(tr.Trades) != 1 || tr.Trades[0].ID != 123412 || tr.HasNext {
		t.Fatalf("%+v", tr)
	}
}

func TestAddSpotOrderAPIKeyTRADEHeaders(t *testing.T) {
	t.Parallel()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := base64.URLEncoding.EncodeToString(priv.Seed())
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "spot_order_limit.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithAPIKey("pub", secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddSpotOrder(context.Background(), types.NewSpotLimitOrder(types.OrderSideBuy, "btc", "rls", "0.6", "520000000")); err != nil {
		t.Fatal(err)
	}
	if got.key != "pub" || got.sig == "" || got.ts == "" || got.auth != "" {
		t.Fatalf("key=%q sig=%q ts=%q auth=%q", got.key, got.sig, got.ts, got.auth)
	}
}
