package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
)

func testdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestOrderBookSpecificSymbol(t *testing.T) {
	t.Parallel()
	fixture := testdata(t, "orderbook_btcirt.json")

	var gotPath, gotUA, gotAuth, gotKey string
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotUA = r.Header.Get("User-Agent")
		gotAuth = r.Header.Get("Authorization")
		gotKey = r.Header.Get("Nobitex-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("should-not-be-sent"),
	)
	if err != nil {
		t.Fatal(err)
	}

	ob, err := c.OrderBook(context.Background(), "  btcirt  ")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q", gotMethod)
	}
	if gotPath != "/v3/orderbook/BTCIRT" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotUA != "TraderBot/MyBot-1.0.0" {
		t.Fatalf("User-Agent = %q", gotUA)
	}
	if gotAuth != "" || gotKey != "" {
		t.Fatalf("public call sent auth Authorization=%q Nobitex-Key=%q", gotAuth, gotKey)
	}
	if ob.Symbol != "BTCIRT" {
		t.Fatalf("symbol = %q", ob.Symbol)
	}
	if !ob.Status.IsOK() {
		t.Fatalf("status = %q", ob.Status)
	}
	if ob.LastTradePrice != "35650565900" {
		t.Fatalf("lastTradePrice = %q", ob.LastTradePrice)
	}
	if len(ob.Asks) != 2 || ob.Asks[0].Price != "1476091000" || ob.Asks[0].Amount != "1.016" {
		t.Fatalf("asks = %+v", ob.Asks)
	}
	if len(ob.Bids) != 2 || ob.Bids[1].Amount != "0.818994" {
		t.Fatalf("bids = %+v", ob.Bids)
	}
}

func TestOrderBookAllConsolidated(t *testing.T) {
	t.Parallel()
	fixture := testdata(t, "orderbook_all.json")

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	all, err := c.OrderBookAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v3/orderbook/all" {
		t.Fatalf("path = %q", gotPath)
	}
	if !all.Status.IsOK() {
		t.Fatalf("status = %q", all.Status)
	}
	if len(all.Books) != 2 {
		t.Fatalf("books = %d", len(all.Books))
	}
	btc, ok := all.Book("BTCIRT")
	if !ok || btc.Asks[0].Amount != "1.016" {
		t.Fatalf("BTCIRT = %+v ok=%v", btc, ok)
	}
	usdt, ok := all.Book("USDTIRT")
	if !ok || usdt.Bids[0].Price != "277960" {
		t.Fatalf("USDTIRT = %+v ok=%v", usdt, ok)
	}
}

func TestOrderBookInvalidSymbol(t *testing.T) {
	t.Parallel()
	fixture := testdata(t, "orderbook_invalid.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.OrderBook(context.Background(), "NOPE")
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.Code != "InvalidSymbol" {
		t.Fatalf("code = %q", e.Code)
	}
}

func TestOrderBookRejectsEmptyAndAll(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.OrderBook(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty symbol")
	}
	_, err = c.OrderBook(context.Background(), "All")
	if err == nil || !strings.Contains(err.Error(), "OrderBookAll") {
		t.Fatalf("expected OrderBookAll redirect, got %v", err)
	}
}

func TestOrderBookLivePublic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live public call")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c, err := client.New(
		client.WithApp("nobitex-sdk-test", "0.1.0"),
		client.WithTimeout(12*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	ob, err := c.OrderBook(ctx, "BTCIRT")
	if err != nil {
		if sdkerr.IsTransport(err) || sdkerr.IsRateLimited(err) {
			t.Skipf("live orderbook unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if !ob.Status.IsOK() || ob.Symbol != "BTCIRT" {
		t.Fatalf("status=%q symbol=%q", ob.Status, ob.Symbol)
	}
	if len(ob.Asks) == 0 || len(ob.Bids) == 0 {
		t.Fatalf("empty depth asks=%d bids=%d", len(ob.Asks), len(ob.Bids))
	}
	if ob.Asks[0].Price == "" || ob.Asks[0].Amount == "" {
		t.Fatalf("asks[0] = %+v", ob.Asks[0])
	}

	all, err := c.OrderBookAll(ctx)
	if err != nil {
		if sdkerr.IsTransport(err) || sdkerr.IsRateLimited(err) {
			t.Skipf("live orderbook all unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if !all.Status.IsOK() || len(all.Books) < 2 {
		t.Fatalf("status=%q books=%d", all.Status, len(all.Books))
	}
	if _, ok := all.Book("BTCIRT"); !ok {
		t.Fatal("live all-markets payload missing BTCIRT")
	}
}

// Ensure testdata files are recorded JSON and not empty.
func TestOrderBookFixturesPresent(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"orderbook_btcirt.json", "orderbook_all.json", "orderbook_invalid.json"} {
		b := testdata(t, name)
		if len(b) == 0 || !strings.Contains(string(b), "status") {
			t.Fatalf("fixture %s looks empty", name)
		}
	}
}
