package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestOrderBookSymbolPathTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		symbol   string
		wantPath string
		wantErr  string
	}{
		{name: "upper", symbol: "BTCIRT", wantPath: "/v3/orderbook/BTCIRT"},
		{name: "trim lower", symbol: "  btcirt  ", wantPath: "/v3/orderbook/BTCIRT"},
		{name: "usdt", symbol: "USDTIRT", wantPath: "/v3/orderbook/USDTIRT"},
		{name: "empty", symbol: "", wantErr: "required"},
		{name: "whitespace", symbol: "  ", wantErr: "required"},
		{name: "all", symbol: "all", wantErr: "OrderBookAll"},
		{name: "ALL", symbol: "ALL", wantErr: "OrderBookAll"},
		{name: "All", symbol: "All", wantErr: "OrderBookAll"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotPath, gotAuth, gotKey string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotAuth = r.Header.Get("Authorization")
				gotKey = r.Header.Get("Nobitex-Key")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(testdata(t, "orderbook_btcirt.json"))
			}))
			defer srv.Close()

			c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("must-not-be-sent"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.OrderBook(context.Background(), tc.symbol)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v want substring %q", err, tc.wantErr)
				}
				if gotPath != "" {
					t.Fatalf("invalid symbol hit the network: %s", gotPath)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotPath != tc.wantPath {
				t.Fatalf("path = %q want %q", gotPath, tc.wantPath)
			}
			if gotAuth != "" || gotKey != "" {
				t.Fatalf("public call sent auth Authorization=%q Nobitex-Key=%q", gotAuth, gotKey)
			}
		})
	}
}

func TestPublicP0StripsAuthTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		fixture  string
		wantPath string
		call     func(context.Context, *client.Client) error
	}{
		{
			name:     "OrderBook",
			fixture:  "orderbook_btcirt.json",
			wantPath: "/v3/orderbook/BTCIRT",
			call: func(ctx context.Context, c *client.Client) error {
				_, err := c.OrderBook(ctx, "BTCIRT")
				return err
			},
		},
		{
			name:     "OrderBookAll",
			fixture:  "orderbook_all.json",
			wantPath: "/v3/orderbook/all",
			call: func(ctx context.Context, c *client.Client) error {
				_, err := c.OrderBookAll(ctx)
				return err
			},
		},
		{
			name:     "SystemOptions",
			fixture:  "system_options.json",
			wantPath: "/v2/options",
			call: func(ctx context.Context, c *client.Client) error {
				_, err := c.SystemOptions(ctx)
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotPath, gotUA, gotAuth, gotKey string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotUA = r.Header.Get("User-Agent")
				gotAuth = r.Header.Get("Authorization")
				gotKey = r.Header.Get("Nobitex-Key")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(testdata(t, tc.fixture))
			}))
			defer srv.Close()

			c, err := client.New(
				client.WithBaseURL(srv.URL),
				client.WithApp("MyBot", "1.0.0"),
				client.WithToken("secret-token"),
			)
			if err != nil {
				t.Fatal(err)
			}
			if err := tc.call(context.Background(), c); err != nil {
				t.Fatal(err)
			}
			if gotPath != tc.wantPath {
				t.Fatalf("path = %q want %q", gotPath, tc.wantPath)
			}
			if gotUA != "TraderBot/MyBot-1.0.0" {
				t.Fatalf("User-Agent = %q", gotUA)
			}
			if gotAuth != "" || gotKey != "" {
				t.Fatalf("public %s sent auth Authorization=%q Nobitex-Key=%q", tc.name, gotAuth, gotKey)
			}
		})
	}
}

func TestP0AuthenticatedMethodsRequireCredentials(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		permission string
		call       func(*client.Client) error
	}{
		{
			name:       "AddMarginOrder",
			permission: "TRADE",
			call: func(c *client.Client) error {
				_, err := c.AddMarginOrder(context.Background(), types.NewMarginMarketOrder(types.OrderSideBuy, "btc", "usdt", "0.01"))
				return err
			},
		},
		{
			name:       "ListPositions",
			permission: "TRADE",
			call: func(c *client.Client) error {
				_, err := c.ListPositions(context.Background(), types.PositionListQuery{})
				return err
			},
		},
		{
			name:       "ClosePosition",
			permission: "TRADE",
			call: func(c *client.Client) error {
				_, err := c.ClosePosition(context.Background(), 128, types.NewCloseMarketOrder("0.01"))
				return err
			},
		},
		{
			name:       "ListOrders",
			permission: "READ",
			call: func(c *client.Client) error {
				_, err := c.ListOrders(context.Background(), types.OrderListQuery{})
				return err
			},
		},
		{
			name:       "ListOrdersPost",
			permission: "READ",
			call: func(c *client.Client) error {
				_, err := c.ListOrdersPost(context.Background(), types.OrderListQuery{})
				return err
			},
		},
		{
			name:       "ListMarginOrders",
			permission: "READ",
			call: func(c *client.Client) error {
				_, err := c.ListMarginOrders(context.Background(), types.OrderListQuery{})
				return err
			},
		},
		{
			name:       "ListMarginOrdersPost",
			permission: "READ",
			call: func(c *client.Client) error {
				_, err := c.ListMarginOrdersPost(context.Background(), types.OrderListQuery{})
				return err
			},
		},
		{
			name:       "CancelOrder",
			permission: "TRADE",
			call: func(c *client.Client) error {
				_, err := c.CancelOrder(context.Background(), types.NewCancelOrderByID(1))
				return err
			},
		},
		{
			name:       "CancelOrderByID",
			permission: "TRADE",
			call: func(c *client.Client) error {
				_, err := c.CancelOrderByID(context.Background(), 1)
				return err
			},
		},
		{
			name:       "CancelOrderByClientOrderID",
			permission: "TRADE",
			call: func(c *client.Client) error {
				_, err := c.CancelOrderByClientOrderID(context.Background(), "order1")
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			called := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				t.Errorf("%s must not hit the network: %s %s", tc.name, r.Method, r.URL.Path)
				w.WriteHeader(http.StatusTeapot)
			}))
			defer srv.Close()

			c, err := client.New(client.WithBaseURL(srv.URL), client.WithApp("MyBot", "1.0.0"))
			if err != nil {
				t.Fatal(err)
			}
			err = tc.call(c)
			if err == nil || !strings.Contains(err.Error(), tc.permission) {
				t.Fatalf("err = %v want permission %q", err, tc.permission)
			}
			if called {
				t.Fatal("HTTP request was sent without credentials")
			}
		})
	}
}

func TestListOrdersQueryEncodingTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		q     types.OrderListQuery
		want  map[string]string
		empty bool
	}{
		{name: "empty omits all", q: types.OrderListQuery{}, empty: true},
		{
			name: "margin filter + paging",
			q: types.OrderListQuery{
				Status:      types.OrderListStatusOpen,
				TradeType:   types.TradeTypeFilterMargin,
				SrcCurrency: " BTC ",
				DstCurrency: "usdt",
				Details:     types.OrderListDetailsFull,
				Sort:        types.OrderListSortIDDesc,
				Page:        2,
				PageSize:    50,
			},
			want: map[string]string{
				"status": "open", "tradeType": "margin", "srcCurrency": "btc",
				"dstCurrency": "usdt", "details": "2", "order": "-id", "page": "2", "pageSize": "50",
			},
		},
		{
			name: "fromId without page",
			q:    types.OrderListQuery{FromID: 99, Execution: types.ExecutionStopLimit, Type: types.OrderSideBuy},
			want: map[string]string{"fromId": "99", "execution": "stop_limit", "type": "buy"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got orderCapture
			srv := newOrderServer(t, http.StatusOK, "orders_list_mixed.json", &got)
			defer srv.Close()
			c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.ListOrders(context.Background(), tc.q); err != nil {
				t.Fatal(err)
			}
			if got.path != "/market/orders/list" || got.method != http.MethodGet {
				t.Fatalf("request %s %s", got.method, got.path)
			}
			if tc.empty {
				if got.rawQuery != "" {
					t.Fatalf("expected empty query, got %q", got.rawQuery)
				}
				return
			}
			parsed := parseQuery(got.rawQuery)
			for k, v := range tc.want {
				if parsed[k] != v {
					t.Fatalf("query[%s]=%q want %q raw=%q", k, parsed[k], v, got.rawQuery)
				}
			}
		})
	}
}

func TestListPositionsQueryEncodingTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		q     types.PositionListQuery
		want  map[string]string
		empty bool
	}{
		{name: "empty omits all", q: types.PositionListQuery{}, empty: true},
		{
			name: "src + past",
			q:    types.PositionListQuery{SrcCurrency: "ETH", Status: types.PositionListPast},
			want: map[string]string{"srcCurrency": "eth", "status": "past"},
		},
		{
			name: "dst + page only",
			q:    types.PositionListQuery{DstCurrency: "rls", Page: 3},
			want: map[string]string{"dstCurrency": "rls", "page": "3"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got posCapture
			srv := newPosServer(t, http.StatusOK, "positions_list.json", &got)
			defer srv.Close()
			c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.ListPositions(context.Background(), tc.q); err != nil {
				t.Fatal(err)
			}
			if tc.empty {
				if got.rawQuery != "" {
					t.Fatalf("expected empty query, got %q", got.rawQuery)
				}
				return
			}
			parsed := parseQuery(got.rawQuery)
			for k, v := range tc.want {
				if parsed[k] != v {
					t.Fatalf("query[%s]=%q want %q raw=%q", k, parsed[k], v, got.rawQuery)
				}
			}
			if parsed["pageSize"] != "" && tc.want["pageSize"] == "" {
				t.Fatalf("unexpected pageSize in %v", parsed)
			}
		})
	}
}

func TestNewFromEnvPublicWithoutSecrets(t *testing.T) {
	t.Setenv("NOBITEX_BASE_URL", "")
	t.Setenv("NOBITEX_TOKEN", "")
	t.Setenv("NOBITEX_API_KEY", "")
	t.Setenv("NOBITEX_API_SECRET", "")
	t.Setenv("NOBITEX_APP_NAME", "MyBot")
	t.Setenv("NOBITEX_APP_VERSION", "1.0.0")

	c, err := client.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if c.Auth() != nil {
		t.Fatal("public NewFromEnv should have no authenticator")
	}
	if c.UserAgent() != "TraderBot/MyBot-1.0.0" {
		t.Fatalf("UserAgent = %q", c.UserAgent())
	}
	if c.BaseURL() != client.DefaultBaseURL {
		t.Fatalf("BaseURL = %q", c.BaseURL())
	}
}

func TestREADMEUsageContract(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	needles := []string{
		"https://apidocs.nobitex.ir",
		"https://github.com/MehrdadMiri/nobitex-sdk",
		"NOBITEX_TOKEN",
		"NewFromEnv",
		"OrderBook",
		"OrderBookAll",
		"AddMarginOrder",
		"ListPositions",
		"ListMarginOrders",
		"CancelOrderByID",
		"examples/orderbook",
		"examples/margin",
		"examples/market",
		"examples/account",
		"MarketStats",
		"MarketDepth",
		"MarketOHLC",
		"UserProfile",
		"ListWallets",
		"ListWithdraws",
		"WebSocketOverview",
		"WebSocketToken",
		"MarginMarkets",
	}
	for _, n := range needles {
		if !strings.Contains(s, n) {
			t.Errorf("README missing %q", n)
		}
	}
	if strings.Contains(s, "sk_live") || strings.Contains(s, "Authorization: Token ntx-") {
		t.Error("README looks like it embeds a real token")
	}
}
