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

type posCapture struct {
	method, path, rawQuery, ua, auth, key, sig, ts, ctype string
	body                                                  []byte
}

func newPosServer(t *testing.T, status int, fixture string, cap *posCapture) *httptest.Server {
	t.Helper()
	payload := testdata(t, fixture)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cap != nil {
			cap.method = r.Method
			cap.path = r.URL.Path
			cap.rawQuery = r.URL.RawQuery
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

func TestListPositionsFiltersTokenAuth(t *testing.T) {
	t.Parallel()
	var got posCapture
	srv := newPosServer(t, http.StatusOK, "positions_list.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("secret-token"),
	)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.ListPositions(context.Background(), types.PositionListQuery{
		SrcCurrency: " BTC ",
		DstCurrency: "rls",
		Status:      types.PositionListActive,
		Page:        1,
		PageSize:    50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.path != "/positions/list" {
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

	q := parseQuery(got.rawQuery)
	if q["srcCurrency"] != "btc" || q["dstCurrency"] != "rls" || q["status"] != "active" {
		t.Fatalf("query = %q parsed=%v", got.rawQuery, q)
	}
	if q["page"] != "1" || q["pageSize"] != "50" {
		t.Fatalf("paging query = %v", q)
	}

	if !resp.Status.IsOK() || resp.HasNext || len(resp.Positions) != 2 {
		t.Fatalf("resp = %+v", resp)
	}
	open, ok := resp.Position(128)
	if !ok || !open.IsOpen() || open.ID != 128 || open.Status != types.PositionStatusOpen {
		t.Fatalf("open = %+v ok=%v", open, ok)
	}
	if types.MoneyValue(open.Liability) != "0.0300450676" {
		t.Fatalf("liability = %q", types.MoneyValue(open.Liability))
	}
	past, ok := resp.Position(32)
	if !ok || past.Status != types.PositionStatusClosed || types.MoneyValue(past.PNL) != "118.46" {
		t.Fatalf("past = %+v ok=%v", past, ok)
	}
}

func TestListPositionsOmitsEmptyFilters(t *testing.T) {
	t.Parallel()
	var got posCapture
	srv := newPosServer(t, http.StatusOK, "positions_list.json", &got)
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListPositions(context.Background(), types.PositionListQuery{}); err != nil {
		t.Fatal(err)
	}
	if got.path != "/positions/list" {
		t.Fatalf("path = %q", got.path)
	}
	if got.rawQuery != "" {
		t.Fatalf("expected no query, got %q", got.rawQuery)
	}
}

func TestListPositionsPastFilter(t *testing.T) {
	t.Parallel()
	var got posCapture
	srv := newPosServer(t, http.StatusOK, "positions_list.json", &got)
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListPositions(context.Background(), types.PositionListQuery{Status: types.PositionListPast}); err != nil {
		t.Fatal(err)
	}
	if parseQuery(got.rawQuery)["status"] != "past" {
		t.Fatalf("query = %q", got.rawQuery)
	}
}

func TestListPositionsRequiresAuth(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("missing-auth list must not hit the network: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithApp("MyBot", "1.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ListPositions(context.Background(), types.PositionListQuery{})
	if err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("err = %v", err)
	}
}

func TestListPositionsAPIError(t *testing.T) {
	t.Parallel()
	srv := newPosServer(t, http.StatusOK, "positions_list_failed.json", nil)
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ListPositions(context.Background(), types.PositionListQuery{Status: types.PositionListActive})
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.Code != "ParseError" {
		t.Fatalf("code = %q", e.Code)
	}
}

func TestListPositionsRejectsInvalidStatusWithoutNetwork(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid status must not hit the network")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ListPositions(context.Background(), types.PositionListQuery{Status: "Open"})
	if err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("err = %v", err)
	}
}

func TestClosePositionLimitTokenAuth(t *testing.T) {
	t.Parallel()
	var got posCapture
	srv := newPosServer(t, http.StatusOK, "close_position_limit.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("secret-token"),
	)
	if err != nil {
		t.Fatal(err)
	}

	req := types.NewCloseLimitOrder("0.0100150225", "6200000000")
	req.ClientOrderID = "close-position-128"

	resp, err := c.ClosePosition(context.Background(), 128, req)
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPost || got.path != "/positions/128/close" {
		t.Fatalf("request %s %s", got.method, got.path)
	}
	if got.ua != "TraderBot/MyBot-1.0.0" {
		t.Fatalf("User-Agent = %q", got.ua)
	}
	if got.auth != "Token secret-token" {
		t.Fatalf("Authorization = %q", got.auth)
	}
	if !strings.Contains(got.ctype, "application/json") {
		t.Fatalf("Content-Type = %q", got.ctype)
	}

	var body map[string]string
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["execution"] != "limit" || body["amount"] != "0.0100150225" || body["price"] != "6200000000" {
		t.Fatalf("body = %s", got.body)
	}
	if body["clientOrderId"] != "close-position-128" {
		t.Fatalf("body = %s", got.body)
	}
	if _, ok := body["stopPrice"]; ok {
		t.Fatalf("limit body included stopPrice: %s", got.body)
	}

	if resp.Order == nil || resp.Order.ID != 28 || resp.Order.Side != types.CloseOrderSideClose {
		t.Fatalf("order = %+v", resp.Order)
	}
	if resp.Order.Type != "buy" { // opposite-side close of a sell position
		t.Fatalf("type = %q", resp.Order.Type)
	}
	if ids := resp.OrderIDs(); len(ids) != 1 || ids[0] != 28 {
		t.Fatalf("OrderIDs = %v", ids)
	}
}

func TestClosePositionExecutions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		fixture  string
		req      types.ClosePositionRequest
		wantExec string
		wantID   int64
		check    func(t *testing.T, sent map[string]string, resp *types.ClosePositionResponse)
	}{
		{
			name:     "market",
			fixture:  "close_position_market.json",
			req:      types.NewCloseMarketOrder("0.0100150225"),
			wantExec: "market",
			wantID:   41,
			check: func(t *testing.T, sent map[string]string, resp *types.ClosePositionResponse) {
				t.Helper()
				if _, ok := sent["price"]; ok {
					t.Fatalf("market sent price: %v", sent)
				}
				if resp.Order.Execution != types.CloseExecutionResponseMarket || resp.Order.Price != "market" {
					t.Fatalf("order = %+v", resp.Order)
				}
			},
		},
		{
			name:     "stop_limit",
			fixture:  "close_position_stop_limit.json",
			req:      types.NewCloseStopLimitOrder("0.0100150225", "6200000000", "6100000000"),
			wantExec: "stop_limit",
			wantID:   52,
			check: func(t *testing.T, sent map[string]string, resp *types.ClosePositionResponse) {
				t.Helper()
				if sent["price"] != "6200000000" || sent["stopPrice"] != "6100000000" {
					t.Fatalf("sent = %v", sent)
				}
				if resp.Order.Execution != types.CloseExecutionResponseStopLimit || resp.Order.Param1 != "6100000000" {
					t.Fatalf("order = %+v", resp.Order)
				}
			},
		},
		{
			name:     "stop_market",
			fixture:  "close_position_stop_market.json",
			req:      types.NewCloseStopMarketOrder("0.0100150225", "6100000000"),
			wantExec: "stop_market",
			wantID:   53,
			check: func(t *testing.T, sent map[string]string, resp *types.ClosePositionResponse) {
				t.Helper()
				if sent["stopPrice"] != "6100000000" {
					t.Fatalf("sent = %v", sent)
				}
				if resp.Order.Execution != types.CloseExecutionResponseStopMarket {
					t.Fatalf("execution = %q", resp.Order.Execution)
				}
			},
		},
		{
			name:    "oco",
			fixture: "close_position_oco.json",
			req: types.ClosePositionRequest{
				Execution:      types.CloseExecutionOCO, // rewritten to execution=limit + mode=oco
				Amount:         "0.0100150225",
				Price:          "12600000000",
				StopPrice:      "13600000000",
				StopLimitPrice: "13610000000",
			},
			wantID: 29,
			check: func(t *testing.T, sent map[string]string, resp *types.ClosePositionResponse) {
				t.Helper()
				if sent["execution"] != "limit" || sent["mode"] != "oco" {
					t.Fatalf("sent = %v", sent)
				}
				if sent["stopLimitPrice"] != "13610000000" {
					t.Fatalf("sent = %v", sent)
				}
				orders := resp.CloseOrders()
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
			var got posCapture
			srv := newPosServer(t, http.StatusOK, tc.fixture, &got)
			defer srv.Close()
			c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
			if err != nil {
				t.Fatal(err)
			}
			resp, err := c.ClosePosition(context.Background(), 128, tc.req)
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
			if got.path != "/positions/128/close" || got.auth != "Token tok" {
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

func TestClosePositionAPIKeyTRADEHeaders(t *testing.T) {
	t.Parallel()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := base64.URLEncoding.EncodeToString(priv.Seed())

	var got posCapture
	srv := newPosServer(t, http.StatusOK, "close_position_limit.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("tradex", "0.1.0"),
		client.WithAPIKey("pub", secret),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ClosePosition(context.Background(), 128, types.NewCloseLimitOrder("0.01", "6200000000")); err != nil {
		t.Fatal(err)
	}
	if got.key != "pub" || got.sig == "" || got.ts == "" {
		t.Fatalf("API-key headers key=%q sig=%q ts=%q", got.key, got.sig, got.ts)
	}
	if got.auth != "" {
		t.Fatalf("unexpected Authorization = %q", got.auth)
	}
	if got.ua != "TraderBot/tradex-0.1.0" {
		t.Fatalf("User-Agent = %q", got.ua)
	}
	if got.path != "/positions/128/close" {
		t.Fatalf("path = %q", got.path)
	}
}

func TestListPositionsAPIKeyTRADEHeaders(t *testing.T) {
	t.Parallel()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := base64.URLEncoding.EncodeToString(priv.Seed())

	var got posCapture
	srv := newPosServer(t, http.StatusOK, "positions_list.json", &got)
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithAPIKey("pub", secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListPositions(context.Background(), types.PositionListQuery{SrcCurrency: "btc"}); err != nil {
		t.Fatal(err)
	}
	if got.key != "pub" || got.sig == "" || got.ts == "" {
		t.Fatalf("API-key headers key=%q sig=%q ts=%q", got.key, got.sig, got.ts)
	}
	if got.auth != "" {
		t.Fatalf("unexpected Authorization = %q", got.auth)
	}
}

func TestClosePositionRequiresAuth(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("missing-auth close must not hit the network")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ClosePosition(context.Background(), 128, types.NewCloseMarketOrder("0.01"))
	if err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("err = %v", err)
	}
}

func TestClosePositionRejectsInvalidIDWithoutNetwork(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid id must not hit the network")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ClosePosition(context.Background(), 0, types.NewCloseMarketOrder("0.01"))
	if err == nil || !strings.Contains(err.Error(), "position id") {
		t.Fatalf("err = %v", err)
	}
}

func TestClosePositionHTTP200Failed(t *testing.T) {
	t.Parallel()
	srv := newPosServer(t, http.StatusOK, "close_position_failed.json", nil)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ClosePosition(context.Background(), 128, types.NewCloseLimitOrder("99", "1"))
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.Code != "ExceedLiability" {
		t.Fatalf("code = %q", e.Code)
	}
}

func TestClosePositionNotFound(t *testing.T) {
	t.Parallel()
	srv := newPosServer(t, http.StatusNotFound, "close_position_no_open.json", nil)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ClosePosition(context.Background(), 999, types.NewCloseMarketOrder("0.01"))
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.Code != "NoOpenPosition" || e.StatusCode != http.StatusNotFound {
		t.Fatalf("err = %+v", e)
	}
}

func TestClosePositionHTTP401(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"status":"failed","code":"UnAuthenticated","message":"Authentication credentials were not provided."}`)
	}))
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("bad"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ClosePosition(context.Background(), 128, types.NewCloseMarketOrder("0.01"))
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", e.StatusCode)
	}
}

func TestPositionFixturesPresent(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"positions_list.json",
		"positions_list_failed.json",
		"close_position_limit.json",
		"close_position_market.json",
		"close_position_stop_limit.json",
		"close_position_stop_market.json",
		"close_position_oco.json",
		"close_position_failed.json",
		"close_position_no_open.json",
	} {
		b := testdata(t, name)
		if len(b) == 0 || !strings.Contains(string(b), "status") {
			t.Fatalf("fixture %s looks empty", name)
		}
	}
}

func parseQuery(raw string) map[string]string {
	out := map[string]string{}
	if raw == "" {
		return out
	}
	for _, part := range strings.Split(raw, "&") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		out[k] = v
	}
	return out
}
