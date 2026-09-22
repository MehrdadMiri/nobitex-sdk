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

type orderCapture struct {
	method, path, rawQuery, ua, auth, key, sig, ts, ctype string
	body                                                  []byte
}

func newOrderServer(t *testing.T, status int, fixture string, cap *orderCapture) *httptest.Server {
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

func TestListMarginOrdersGETFilterTokenAuth(t *testing.T) {
	t.Parallel()
	var got orderCapture
	srv := newOrderServer(t, http.StatusOK, "orders_list_margin.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("secret-token"),
	)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.ListMarginOrders(context.Background(), types.OrderListQuery{
		Status:      types.OrderListStatusOpen,
		SrcCurrency: " BTC ",
		Details:     types.OrderListDetailsFull,
		Page:        1,
		PageSize:    100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.path != "/market/orders/list" {
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
	if q["tradeType"] != "margin" {
		t.Fatalf("missing margin filter: query = %q parsed=%v", got.rawQuery, q)
	}
	if q["status"] != "open" || q["srcCurrency"] != "btc" || q["details"] != "2" {
		t.Fatalf("query = %q parsed=%v", got.rawQuery, q)
	}
	if q["page"] != "1" || q["pageSize"] != "100" {
		t.Fatalf("paging = %v", q)
	}

	if !resp.Status.IsOK() || !resp.HasNext || len(resp.Orders) != 2 {
		t.Fatalf("resp = %+v", resp)
	}
	if ids := resp.OrderIDs(); len(ids) != 2 || ids[0] != 173546224 {
		t.Fatalf("OrderIDs = %v", ids)
	}
	if !resp.Orders[0].IsMargin() || resp.Orders[0].ClientOrderIDValue() != "margin-1" {
		t.Fatalf("order0 = %+v", resp.Orders[0])
	}
	if resp.Orders[1].ClientOrderID != nil || resp.Orders[1].Param1 != "9500" {
		t.Fatalf("order1 = %+v", resp.Orders[1])
	}
}

func TestListOrdersPOSTMarginBody(t *testing.T) {
	t.Parallel()
	var got orderCapture
	srv := newOrderServer(t, http.StatusOK, "orders_list_mixed.json", &got)
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.ListMarginOrdersPost(context.Background(), types.OrderListQuery{
		Type:      types.OrderSideSell,
		Execution: types.ExecutionLimit,
		Details:   2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPost || got.path != "/market/orders/list" {
		t.Fatalf("request %s %s", got.method, got.path)
	}
	if got.rawQuery != "" {
		t.Fatalf("POST should not use query string, got %q", got.rawQuery)
	}
	if !strings.Contains(got.ctype, "application/json") {
		t.Fatalf("Content-Type = %q", got.ctype)
	}
	var body map[string]any
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["tradeType"] != "margin" || body["type"] != "sell" || body["execution"] != "limit" {
		t.Fatalf("body = %s", got.body)
	}
	if int(body["details"].(float64)) != 2 {
		t.Fatalf("details = %v", body["details"])
	}
	margin := resp.MarginOrders()
	if len(margin) != 1 || margin[0].ID != 173546224 || !margin[0].IsMargin() {
		t.Fatalf("MarginOrders = %+v", margin)
	}
	if len(resp.Orders) != 2 || resp.Orders[0].TradeType != types.TradeTypeSpot {
		t.Fatalf("orders = %+v", resp.Orders)
	}
}

func TestListOrdersOmitsEmptyFilters(t *testing.T) {
	t.Parallel()
	var got orderCapture
	srv := newOrderServer(t, http.StatusOK, "orders_list_mixed.json", &got)
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListOrders(context.Background(), types.OrderListQuery{}); err != nil {
		t.Fatal(err)
	}
	if got.rawQuery != "" {
		t.Fatalf("empty query should omit params, got %q", got.rawQuery)
	}
}

func TestListOrdersAPIKeyREADHeaders(t *testing.T) {
	t.Parallel()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := base64.URLEncoding.EncodeToString(priv.Seed())

	var got orderCapture
	srv := newOrderServer(t, http.StatusOK, "orders_list_margin.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("tradex", "0.1.0"),
		client.WithAPIKey("pub-key", secret),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListMarginOrders(context.Background(), types.OrderListQuery{Details: 2}); err != nil {
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
	if parseQuery(got.rawQuery)["tradeType"] != "margin" {
		t.Fatalf("query = %q", got.rawQuery)
	}
}

func TestListOrdersRequiresAuth(t *testing.T) {
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
	_, err = c.ListMarginOrders(context.Background(), types.OrderListQuery{})
	if err == nil || !strings.Contains(err.Error(), "requires Token or API-key (READ)") {
		t.Fatalf("err = %v", err)
	}
	if called {
		t.Fatal("HTTP request was sent without credentials")
	}
}

func TestListOrdersRejectsInvalidQuery(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid query should not be sent")
	}))
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ListOrders(context.Background(), types.OrderListQuery{Page: 1, FromID: 10})
	if err == nil || !strings.Contains(err.Error(), "fromId") {
		t.Fatalf("err = %v", err)
	}
}

func TestListOrdersFailedBody(t *testing.T) {
	t.Parallel()
	srv := newOrderServer(t, http.StatusOK, "orders_list_failed.json", nil)
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ListOrders(context.Background(), types.OrderListQuery{})
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.Code != "UnAuthenticated" {
		t.Fatalf("code = %q", e.Code)
	}
}

func TestCancelOrderByIDTokenAuth(t *testing.T) {
	t.Parallel()
	var got orderCapture
	srv := newOrderServer(t, http.StatusOK, "orders_cancel_by_id.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("secret-token"),
	)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.CancelOrderByID(context.Background(), 5684)
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPost || got.path != "/market/orders/update-status" {
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
	var body map[string]any
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "canceled" || int(body["order"].(float64)) != 5684 {
		t.Fatalf("body = %s", got.body)
	}
	if _, ok := body["clientOrderId"]; ok {
		t.Fatalf("id cancel should omit clientOrderId: %s", got.body)
	}
	if resp.Order == nil || resp.Order.ID != 5684 || !resp.UpdatedStatus.IsCanceled() {
		t.Fatalf("resp = %+v", resp)
	}
	if !resp.Order.Status.IsCanceled() || resp.Order.ClientOrderID != nil {
		t.Fatalf("order = %+v", resp.Order)
	}
}

func TestCancelOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	var got orderCapture
	srv := newOrderServer(t, http.StatusOK, "orders_cancel_by_client.json", &got)
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.CancelOrderByClientOrderID(context.Background(), "order1")
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "canceled" || body["clientOrderId"] != "order1" {
		t.Fatalf("body = %s", got.body)
	}
	if _, ok := body["order"]; ok {
		t.Fatalf("client cancel should omit order id: %s", got.body)
	}
	if resp.Order == nil || resp.Order.ClientOrderIDValue() != "order1" || resp.Order.ID != 173546223 {
		t.Fatalf("order = %+v", resp.Order)
	}
	if !resp.Order.IsMargin() || !resp.UpdatedStatus.IsCanceled() {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestCancelOrderAPIKeyTRADEHeaders(t *testing.T) {
	t.Parallel()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := base64.URLEncoding.EncodeToString(priv.Seed())

	var got orderCapture
	srv := newOrderServer(t, http.StatusOK, "orders_cancel_by_id.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("tradex", "0.1.0"),
		client.WithAPIKey("pub-key", secret),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.CancelOrder(context.Background(), types.NewCancelOrderByID(5684)); err != nil {
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
	if got.path != "/market/orders/update-status" {
		t.Fatalf("path = %s", got.path)
	}
}

func TestCancelOrderRequiresAuthTRADE(t *testing.T) {
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
	_, err = c.CancelOrderByID(context.Background(), 5684)
	if err == nil || !strings.Contains(err.Error(), "requires Token or API-key (TRADE)") {
		t.Fatalf("err = %v", err)
	}
	if called {
		t.Fatal("HTTP request was sent without credentials")
	}
}

func TestCancelOrderRejectsMissingIdentifier(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid cancel should not be sent")
	}))
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CancelOrder(context.Background(), types.CancelOrderRequest{})
	if err == nil || !strings.Contains(err.Error(), "order id or clientOrderId") {
		t.Fatalf("err = %v", err)
	}
}

func TestCancelOrderFailedMissingIdentifierBody(t *testing.T) {
	t.Parallel()
	srv := newOrderServer(t, http.StatusOK, "orders_cancel_failed.json", nil)
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CancelOrderByID(context.Background(), 1)
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.Code != "NullIdAndClientOrderId" {
		t.Fatalf("code = %q", e.Code)
	}
}

func TestCancelOrderNotAppliedFailedBody(t *testing.T) {
	t.Parallel()
	srv := newOrderServer(t, http.StatusOK, "orders_cancel_not_applied.json", nil)
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CancelOrderByID(context.Background(), 5684)
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if !e.Status.IsFailed() || !strings.Contains(string(e.RawBody), `"updatedStatus": "Done"`) {
		t.Fatalf("err = %+v body=%s", e, e.RawBody)
	}
}

func TestCancelOrderHTTPUnauthorized(t *testing.T) {
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
	_, err = c.CancelOrderByClientOrderID(context.Background(), "order1")
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.Code != "UnAuthenticated" || e.StatusCode != http.StatusUnauthorized {
		t.Fatalf("code=%q http=%d", e.Code, e.StatusCode)
	}
}

func TestOrderListAndCancelFixturesPresent(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"orders_list_margin.json",
		"orders_list_mixed.json",
		"orders_list_failed.json",
		"orders_cancel_by_id.json",
		"orders_cancel_by_client.json",
		"orders_cancel_failed.json",
		"orders_cancel_not_applied.json",
	} {
		b := testdata(t, name)
		if len(b) == 0 || !strings.Contains(string(b), "status") {
			t.Fatalf("fixture %s looks empty", name)
		}
	}
}
