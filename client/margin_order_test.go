package client_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

type marginCapture struct {
	method, path, ua, auth, key, sig, ts, ctype string
	body                                        []byte
}

func newMarginServer(t *testing.T, status int, fixture string, cap *marginCapture) *httptest.Server {
	t.Helper()
	payload := testdata(t, fixture)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cap != nil {
			cap.method = r.Method
			cap.path = r.URL.Path
			cap.ua = r.Header.Get("User-Agent")
			cap.auth = r.Header.Get("Authorization")
			cap.key = r.Header.Get("Nobitex-Key")
			cap.sig = r.Header.Get("Nobitex-Signature")
			cap.ts = r.Header.Get("Nobitex-Timestamp")
			cap.ctype = r.Header.Get("Content-Type")
			cap.body, _ = io.ReadAll(r.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(payload)
	}))
}

func TestAddMarginOrderLimitTokenAuth(t *testing.T) {
	t.Parallel()
	var got marginCapture
	srv := newMarginServer(t, http.StatusOK, "margin_order_limit.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("secret-token"),
	)
	if err != nil {
		t.Fatal(err)
	}

	req := types.NewMarginLimitOrder(types.OrderSideSell, "BTC", "USDT", "0.01", "13400000000")
	req.Leverage = "2"
	req.ClientOrderID = "my-order-123"

	resp, err := c.AddMarginOrder(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPost || got.path != "/margin/orders/add" {
		t.Fatalf("request %s %s", got.method, got.path)
	}
	if got.ua != "TraderBot/MyBot-1.0.0" {
		t.Fatalf("User-Agent = %q", got.ua)
	}
	if got.auth != "Token secret-token" {
		t.Fatalf("Authorization = %q", got.auth)
	}
	if got.key != "" || got.sig != "" {
		t.Fatalf("unexpected API-key headers key=%q sig=%q", got.key, got.sig)
	}
	if !strings.Contains(got.ctype, "application/json") {
		t.Fatalf("Content-Type = %q", got.ctype)
	}

	var body map[string]string
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["execution"] != "limit" || body["type"] != "sell" || body["srcCurrency"] != "btc" {
		t.Fatalf("body = %s", got.body)
	}
	if body["price"] != "13400000000" || body["clientOrderId"] != "my-order-123" || body["leverage"] != "2" {
		t.Fatalf("body = %s", got.body)
	}
	if _, ok := body["stopPrice"]; ok {
		t.Fatalf("limit body included stopPrice: %s", got.body)
	}

	if resp.Order == nil || resp.Order.ID != 25 {
		t.Fatalf("order = %+v", resp.Order)
	}
	if resp.Order.ClientOrderIDValue() != "my-order-123" {
		t.Fatalf("clientOrderId = %q", resp.Order.ClientOrderIDValue())
	}
	if ids := resp.OrderIDs(); len(ids) != 1 || ids[0] != 25 {
		t.Fatalf("OrderIDs = %v", ids)
	}
}

func TestAddMarginOrderExecutions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		fixture  string
		req      types.MarginOrderRequest
		wantExec string
		wantID   int64
		check    func(t *testing.T, sent map[string]string, resp *types.MarginOrderAddResponse)
	}{
		{
			name:     "market",
			fixture:  "margin_order_market.json",
			req:      types.NewMarginMarketOrder(types.OrderSideBuy, "btc", "usdt", "0.01"),
			wantExec: "market",
			wantID:   41,
			check: func(t *testing.T, sent map[string]string, resp *types.MarginOrderAddResponse) {
				t.Helper()
				if _, ok := sent["price"]; ok {
					t.Fatalf("market sent price: %v", sent)
				}
				if resp.Order.Price != "market" || resp.Order.ClientOrderID != nil {
					t.Fatalf("order = %+v", resp.Order)
				}
			},
		},
		{
			name:     "stop_limit",
			fixture:  "margin_order_stop_limit.json",
			req:      types.NewMarginStopLimitOrder(types.OrderSideSell, "btc", "usdt", "0.01", "12500000000", "12600000000"),
			wantExec: "stop_limit",
			wantID:   52,
			check: func(t *testing.T, sent map[string]string, resp *types.MarginOrderAddResponse) {
				t.Helper()
				if sent["price"] != "12500000000" || sent["stopPrice"] != "12600000000" {
					t.Fatalf("sent = %v", sent)
				}
				if resp.Order.Execution != types.ExecutionResponseStopLimit || resp.Order.Param1 != "12600000000" {
					t.Fatalf("order = %+v", resp.Order)
				}
			},
		},
		{
			name:     "stop_market",
			fixture:  "margin_order_stop_market.json",
			req:      types.NewMarginStopMarketOrder(types.OrderSideSell, "btc", "usdt", "0.01", "12600000000"),
			wantExec: "stop_market",
			wantID:   53,
			check: func(t *testing.T, sent map[string]string, resp *types.MarginOrderAddResponse) {
				t.Helper()
				if sent["stopPrice"] != "12600000000" {
					t.Fatalf("sent = %v", sent)
				}
				if resp.Order.Execution != types.ExecutionResponseStopMarket {
					t.Fatalf("execution = %q", resp.Order.Execution)
				}
			},
		},
		{
			name:    "oco",
			fixture: "margin_order_oco.json",
			req: types.MarginOrderRequest{
				Execution:      types.ExecutionOCO, // rewritten to execution=limit + mode=oco
				Type:           types.OrderSideBuy,
				SrcCurrency:    "btc",
				DstCurrency:    "rls",
				Amount:         "0.01",
				Price:          "12600000000",
				StopPrice:      "13600000000",
				StopLimitPrice: "13610000000",
			},
			wantID: 29,
			check: func(t *testing.T, sent map[string]string, resp *types.MarginOrderAddResponse) {
				t.Helper()
				if sent["execution"] != "limit" || sent["mode"] != "oco" {
					t.Fatalf("sent = %v", sent)
				}
				if sent["stopLimitPrice"] != "13610000000" {
					t.Fatalf("sent = %v", sent)
				}
				orders := resp.PlacedOrders()
				if len(orders) != 2 || orders[0].ID != 29 || orders[1].ID != 30 {
					t.Fatalf("orders = %+v", orders)
				}
				if orders[0].PairID == nil || *orders[0].PairID != 30 {
					t.Fatalf("pairId = %v", orders[0].PairID)
				}
				if ids := resp.OrderIDs(); len(ids) != 2 {
					t.Fatalf("OrderIDs = %v", ids)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got marginCapture
			srv := newMarginServer(t, http.StatusOK, tc.fixture, &got)
			defer srv.Close()
			c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
			if err != nil {
				t.Fatal(err)
			}
			resp, err := c.AddMarginOrder(context.Background(), tc.req)
			if err != nil {
				t.Fatal(err)
			}
			var sent map[string]string
			if err := json.Unmarshal(got.body, &sent); err != nil {
				t.Fatal(err)
			}
			if tc.wantExec != "" && sent["execution"] != tc.wantExec {
				t.Fatalf("execution = %q want %q body=%s", sent["execution"], tc.wantExec, got.body)
			}
			if got.path != "/margin/orders/add" || got.auth != "Token tok" {
				t.Fatalf("path=%s auth=%s", got.path, got.auth)
			}
			if tc.wantID != 0 {
				found := false
				for _, id := range resp.OrderIDs() {
					if id == tc.wantID {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing id %d in %v", tc.wantID, resp.OrderIDs())
				}
			}
			if tc.check != nil {
				tc.check(t, sent, resp)
			}
		})
	}
}

func TestAddMarginOrderAPIKeyTRADEHeaders(t *testing.T) {
	t.Parallel()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := base64.URLEncoding.EncodeToString(priv.Seed())

	var got marginCapture
	srv := newMarginServer(t, http.StatusOK, "margin_order_limit.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("tradex", "0.1.0"),
		client.WithAPIKey("pub-key", secret),
	)
	if err != nil {
		t.Fatal(err)
	}
	req := types.NewMarginLimitOrder(types.OrderSideSell, "btc", "usdt", "0.01", "13400000000")
	if _, err := c.AddMarginOrder(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got.key != "pub-key" || got.ts == "" || got.sig == "" {
		t.Fatalf("API-key headers key=%q ts=%q sig=%q", got.key, got.ts, got.sig)
	}
	if got.auth != "" {
		t.Fatalf("Authorization = %q, want empty for API-key auth", got.auth)
	}
	if got.ua != "TraderBot/tradex-0.1.0" {
		t.Fatalf("User-Agent = %q", got.ua)
	}
	if len(got.body) == 0 {
		t.Fatal("empty signed body")
	}
}

func TestAddMarginOrderRequiresAuth(t *testing.T) {
	t.Parallel()
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Errorf("unauthenticated client must not call the API")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithApp("MyBot", "1.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.AddMarginOrder(context.Background(), types.NewMarginMarketOrder(types.OrderSideBuy, "btc", "usdt", "0.01"))
	if err == nil || !strings.Contains(err.Error(), "requires Token or API-key") {
		t.Fatalf("err = %v", err)
	}
	if called {
		t.Fatal("HTTP request was sent without credentials")
	}
}

func TestAddMarginOrderFailedBody(t *testing.T) {
	t.Parallel()
	srv := newMarginServer(t, http.StatusOK, "margin_order_failed.json", nil)
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.AddMarginOrder(context.Background(), types.NewMarginStopMarketOrder(types.OrderSideSell, "btc", "usdt", "0.01", "1"))
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.Code != "MissingStopPrice" {
		t.Fatalf("code = %q", e.Code)
	}
}

func TestAddMarginOrderHTTPUnauthorized(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"status":"failed","code":"UnAuthenticated"}`)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("bad"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.AddMarginOrder(context.Background(), types.NewMarginLimitOrder(types.OrderSideSell, "btc", "usdt", "0.01", "1"))
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.Code != "UnAuthenticated" || e.StatusCode != http.StatusUnauthorized {
		t.Fatalf("code=%q http=%d", e.Code, e.StatusCode)
	}
}

func TestAddMarginOrderRejectsInvalidRequest(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid request should not be sent")
	}))
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.AddMarginOrder(context.Background(), types.NewMarginLimitOrder(types.OrderSideSell, "btc", "usdt", "0.01", ""))
	if err == nil || !strings.Contains(err.Error(), "price") {
		t.Fatalf("err = %v", err)
	}
}

func TestAddMarginOrderFixturesPresent(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"margin_order_limit.json",
		"margin_order_market.json",
		"margin_order_stop_limit.json",
		"margin_order_stop_market.json",
		"margin_order_oco.json",
		"margin_order_failed.json",
	} {
		b := testdata(t, name)
		if len(b) == 0 || !strings.Contains(string(b), "status") {
			t.Fatalf("fixture %s looks empty", name)
		}
	}
}
