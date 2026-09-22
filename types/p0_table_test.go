package types_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func clientFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "client", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPriceLevelUnmarshalTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		raw       string
		wantPrice types.Money
		wantAmt   types.Money
		wantErr   string
	}{
		{name: "documented pair", raw: `["1476091000","1.016"]`, wantPrice: "1476091000", wantAmt: "1.016"},
		{name: "empty strings", raw: `["",""]`, wantPrice: "", wantAmt: ""},
		{name: "empty array", raw: `[]`, wantErr: "price level"},
		{name: "one value", raw: `["only-price"]`, wantErr: "price level"},
		{name: "three values", raw: `["1","2","3"]`, wantErr: "price level"},
		{name: "object", raw: `{"price":"1","amount":"2"}`, wantErr: ""}, // json.Unmarshal into []Money fails
		{name: "json numbers", raw: `[1476091000,1.016]`, wantErr: ""},   // Money is a JSON string
		{name: "null", raw: `null`, wantErr: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var p types.PriceLevel
			err := json.Unmarshal([]byte(tc.raw), &p)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if tc.name == "object" || tc.name == "json numbers" || tc.name == "null" {
				if err == nil {
					t.Fatalf("documented encoding is [string,string]; accepted %s as %+v", tc.raw, p)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if p.Price != tc.wantPrice || p.Amount != tc.wantAmt {
				t.Fatalf("got %+v", p)
			}
		})
	}
}

func TestOrderBookAllReservedKeysAndLookup(t *testing.T) {
	t.Parallel()
	const body = `{
	  "status":"ok",
	  "code":"ignored",
	  "message":"ignored",
	  "backOff":12,
	  "limit":60,
	  "BTCIRT":{"lastTradePrice":"1","asks":[["1","2"]],"bids":[]},
	  "usdtIRT":{"lastTradePrice":"3","asks":[],"bids":[["4","5"]]}
	}`
	var all types.OrderBookAll
	if err := json.Unmarshal([]byte(body), &all); err != nil {
		t.Fatal(err)
	}
	if !all.Status.IsOK() || all.Code != "ignored" || all.BackOff != 12 {
		t.Fatalf("envelope = %+v", all.Envelope)
	}
	if _, ok := all.Books["code"]; ok {
		t.Fatal("reserved key code should not be a book")
	}
	if _, ok := all.Books["message"]; ok {
		t.Fatal("reserved key message should not be a book")
	}
	if _, ok := all.Books["backOff"]; ok {
		t.Fatal("reserved key backOff should not be a book")
	}
	if _, ok := all.Books["limit"]; ok {
		t.Fatal("reserved key limit should not be a book")
	}
	if len(all.Books) != 2 {
		t.Fatalf("books = %d keys=%v", len(all.Books), all.Books)
	}

	lookups := []struct {
		symbol string
		ok     bool
		last   types.Money
	}{
		{"BTCIRT", true, "1"},
		{"btcirt", true, "1"},
		{" BTCIRT ", true, "1"},
		{"usdtIRT", true, "3"},
		{"USDTIRT", false, ""}, // map key is mixed-case usdtIRT; Book only uppercases the query
		{"NOPE", false, ""},
		{"", false, ""},
	}
	for _, tc := range lookups {
		got, ok := all.Book(tc.symbol)
		if ok != tc.ok {
			t.Errorf("Book(%q) ok=%v want %v", tc.symbol, ok, tc.ok)
			continue
		}
		if ok && got.LastTradePrice != tc.last {
			t.Errorf("Book(%q).LastTradePrice = %q want %q", tc.symbol, got.LastTradePrice, tc.last)
		}
	}

	var none *types.OrderBookAll
	if _, ok := none.Book("BTCIRT"); ok {
		t.Fatal("nil receiver should miss")
	}
}

func TestStatusHelpersTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		s        types.Status
		ok, fail bool
	}{
		{types.StatusOK, true, false},
		{types.StatusSuccess, true, false},
		{types.StatusFailed, false, true},
		{"", false, false},
		{"OK", false, false},
	}
	for _, tc := range cases {
		if tc.s.IsOK() != tc.ok || tc.s.IsFailed() != tc.fail {
			t.Errorf("%q IsOK=%v IsFailed=%v want %v/%v", tc.s, tc.s.IsOK(), tc.s.IsFailed(), tc.ok, tc.fail)
		}
	}
}

func TestOrderStatusHelpersTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		s              types.OrderStatus
		open, canceled bool
	}{
		{types.OrderStatusNew, true, false},
		{types.OrderStatusActive, true, false},
		{types.OrderStatusInactive, true, false},
		{types.OrderStatusDone, false, false},
		{types.OrderStatusCanceled, false, true},
		{"", false, false},
	}
	for _, tc := range cases {
		if tc.s.IsOpen() != tc.open || tc.s.IsCanceled() != tc.canceled {
			t.Errorf("%q IsOpen=%v IsCanceled=%v want %v/%v", tc.s, tc.s.IsOpen(), tc.s.IsCanceled(), tc.open, tc.canceled)
		}
	}
}

func TestTradeTypeFilterTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   types.TradeType
		want types.TradeTypeFilter
	}{
		{types.TradeTypeMargin, types.TradeTypeFilterMargin},
		{types.TradeTypeSpot, types.TradeTypeFilterSpot},
		{" Margin ", "margin"},
		{"SPOT", types.TradeTypeFilterSpot},
	}
	for _, tc := range cases {
		if got := tc.in.Filter(); got != tc.want {
			t.Errorf("Filter(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestPositionListQueryNormalizeTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		in      types.PositionListQuery
		wantSrc string
		wantDst string
		wantSt  types.PositionListStatus
		wantErr string
	}{
		{name: "trim lower", in: types.PositionListQuery{SrcCurrency: " BTC ", DstCurrency: "RLS", Status: "ACTIVE"}, wantSrc: "btc", wantDst: "rls", wantSt: types.PositionListActive},
		{name: "past", in: types.PositionListQuery{Status: "Past"}, wantSt: types.PositionListPast},
		{name: "empty leaves API default", in: types.PositionListQuery{}},
		{name: "invalid status", in: types.PositionListQuery{Status: "Open"}, wantErr: "status"},
		{name: "negative page", in: types.PositionListQuery{Page: -1}, wantErr: "page"},
		{name: "negative pageSize", in: types.PositionListQuery{PageSize: -3}, wantErr: "pageSize"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.in.Normalize()
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.SrcCurrency != tc.wantSrc || got.DstCurrency != tc.wantDst || got.Status != tc.wantSt {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestCancelOrderRequestPrepareTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		in      types.CancelOrderRequest
		wantID  int64
		wantCID string
		wantErr string
	}{
		{name: "id only defaults status", in: types.CancelOrderRequest{Order: 5684}, wantID: 5684},
		{name: "client id trimmed", in: types.CancelOrderRequest{ClientOrderID: " order1 "}, wantCID: "order1"},
		{name: "both kept", in: types.CancelOrderRequest{Order: 1, ClientOrderID: "abc"}, wantID: 1, wantCID: "abc"},
		{name: "empty", in: types.CancelOrderRequest{}, wantErr: "order id or clientOrderId"},
		{name: "negative id", in: types.CancelOrderRequest{Order: -1}, wantErr: "order id"},
		{name: "bad client", in: types.CancelOrderRequest{ClientOrderID: "bad id"}, wantErr: "clientOrderId"},
		{name: "wrong status", in: types.CancelOrderRequest{Order: 1, Status: "done"}, wantErr: "canceled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.in.Prepare()
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != types.OrderCancelStatus || got.Order != tc.wantID || got.ClientOrderID != tc.wantCID {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestValidateOrderDecimalsTable(t *testing.T) {
	t.Parallel()
	var opts types.SystemOptions
	if err := json.Unmarshal([]byte(documentedSystemOptions), &opts); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, market, amount, price string
		want                        error
	}{
		{name: "valid limit", market: "BTCIRT", amount: "0.001", price: "35650565900"},
		{name: "valid market empty price", market: "BTCIRT", amount: "0.001", price: ""},
		{name: "valid BTCUSDT", market: "BTCUSDT", amount: "0.01", price: "68000.01"},
		{name: "amount not multiple", market: "BTCIRT", amount: "0.00000015", price: "10", want: types.ErrNotMultiple},
		{name: "price not multiple", market: "BTCIRT", amount: "0.001", price: "123", want: types.ErrNotMultiple},
		{name: "unknown market", market: "NOPE", amount: "1", price: "1", want: types.ErrUnknownMarket},
		{name: "zero amount", market: "BTCIRT", amount: "0", price: "10", want: types.ErrNonPositive},
		{name: "empty market", market: "  ", amount: "1", price: "1", want: types.ErrEmptyMarket},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := opts.ValidateOrderDecimals(tc.market, types.Money(tc.amount), types.Money(tc.price))
			if tc.want == nil {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v want %v", err, tc.want)
			}
		})
	}
}

func TestP0SuccessFixturesParse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		file string
		new  func() any
		ok   func(t *testing.T, dest any)
	}{
		{
			name: "orderbook symbol",
			file: "orderbook_btcirt.json",
			new:  func() any { return &types.OrderBook{} },
			ok: func(t *testing.T, dest any) {
				ob := dest.(*types.OrderBook)
				if !ob.Status.IsOK() || ob.LastTradePrice != "35650565900" || len(ob.Asks) != 2 {
					t.Fatalf("%+v", ob)
				}
			},
		},
		{
			name: "orderbook all",
			file: "orderbook_all.json",
			new:  func() any { return &types.OrderBookAll{} },
			ok: func(t *testing.T, dest any) {
				all := dest.(*types.OrderBookAll)
				if !all.Status.IsOK() || len(all.Books) != 2 {
					t.Fatalf("%+v", all)
				}
				if _, ok := all.Book("BTCIRT"); !ok {
					t.Fatal("missing BTCIRT")
				}
			},
		},
		{
			name: "system options",
			file: "system_options.json",
			new:  func() any { return &types.SystemOptions{} },
			ok: func(t *testing.T, dest any) {
				opts := dest.(*types.SystemOptions)
				if !opts.Status.IsOK() || opts.AmountPrecisions()["BTCIRT"] == "" {
					t.Fatalf("%+v", opts.Nobitex)
				}
			},
		},
		{
			name: "margin limit",
			file: "margin_order_limit.json",
			new:  func() any { return &types.MarginOrderAddResponse{} },
			ok: func(t *testing.T, dest any) {
				r := dest.(*types.MarginOrderAddResponse)
				if r.Order == nil || r.Order.ID == 0 || !r.Status.IsOK() {
					t.Fatalf("%+v", r)
				}
			},
		},
		{
			name: "margin market",
			file: "margin_order_market.json",
			new:  func() any { return &types.MarginOrderAddResponse{} },
			ok: func(t *testing.T, dest any) {
				r := dest.(*types.MarginOrderAddResponse)
				if r.Order == nil || r.Order.Execution != types.ExecutionResponseMarket {
					t.Fatalf("%+v", r.Order)
				}
			},
		},
		{
			name: "margin oco",
			file: "margin_order_oco.json",
			new:  func() any { return &types.MarginOrderAddResponse{} },
			ok: func(t *testing.T, dest any) {
				r := dest.(*types.MarginOrderAddResponse)
				if n := len(r.PlacedOrders()); n != 2 {
					t.Fatalf("placed = %d", n)
				}
			},
		},
		{
			name: "positions list",
			file: "positions_list.json",
			new:  func() any { return &types.PositionListResponse{} },
			ok: func(t *testing.T, dest any) {
				r := dest.(*types.PositionListResponse)
				if !r.Status.IsOK() || len(r.Positions) == 0 {
					t.Fatalf("%+v", r)
				}
				if _, ok := r.Position(r.Positions[0].ID); !ok {
					t.Fatal("Position lookup missed first id")
				}
			},
		},
		{
			name: "close limit",
			file: "close_position_limit.json",
			new:  func() any { return &types.ClosePositionResponse{} },
			ok: func(t *testing.T, dest any) {
				r := dest.(*types.ClosePositionResponse)
				if r.Order == nil || r.Order.ID == 0 {
					t.Fatalf("%+v", r)
				}
			},
		},
		{
			name: "close oco",
			file: "close_position_oco.json",
			new:  func() any { return &types.ClosePositionResponse{} },
			ok: func(t *testing.T, dest any) {
				r := dest.(*types.ClosePositionResponse)
				if n := len(r.CloseOrders()); n != 2 {
					t.Fatalf("close orders = %d", n)
				}
			},
		},
		{
			name: "orders list margin",
			file: "orders_list_margin.json",
			new:  func() any { return &types.OrderListResponse{} },
			ok: func(t *testing.T, dest any) {
				r := dest.(*types.OrderListResponse)
				if !r.Status.IsOK() || len(r.MarginOrders()) == 0 {
					t.Fatalf("%+v", r)
				}
			},
		},
		{
			name: "cancel by id",
			file: "orders_cancel_by_id.json",
			new:  func() any { return &types.CancelOrderResponse{} },
			ok: func(t *testing.T, dest any) {
				r := dest.(*types.CancelOrderResponse)
				if r.Order == nil || !r.UpdatedStatus.IsCanceled() {
					t.Fatalf("%+v", r)
				}
			},
		},
		{
			name: "cancel by clientOrderId",
			file: "orders_cancel_by_client.json",
			new:  func() any { return &types.CancelOrderResponse{} },
			ok: func(t *testing.T, dest any) {
				r := dest.(*types.CancelOrderResponse)
				if r.Order == nil || r.Order.ClientOrderIDValue() == "" {
					t.Fatalf("%+v", r.Order)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dest := tc.new()
			if err := json.Unmarshal(clientFixture(t, tc.file), dest); err != nil {
				t.Fatal(err)
			}
			tc.ok(t, dest)
		})
	}
}

func TestP0FailedFixturesMapToAPIError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		file       string
		statusCode int
		wantCode   string
	}{
		{"orderbook_invalid.json", 400, "InvalidSymbol"},
		{"system_options_failed.json", 200, "Maintenance"},
		{"margin_order_failed.json", 200, "MissingStopPrice"},
		{"positions_list_failed.json", 200, "ParseError"},
		{"close_position_failed.json", 200, "ExceedLiability"},
		{"close_position_no_open.json", 404, "NoOpenPosition"},
		{"orders_list_failed.json", 200, "UnAuthenticated"},
		{"orders_cancel_failed.json", 200, "NullIdAndClientOrderId"},
		{"orders_cancel_not_applied.json", 200, ""}, // failed body, code may be empty
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			t.Parallel()
			err := sdkerr.Check(tc.statusCode, clientFixture(t, tc.file))
			if !sdkerr.IsAPI(err) {
				t.Fatalf("err = %v", err)
			}
			var e *sdkerr.Error
			sdkerr.As(err, &e)
			if tc.wantCode != "" && e.Code != tc.wantCode {
				t.Fatalf("code = %q want %q", e.Code, tc.wantCode)
			}
			if !e.Status.IsFailed() {
				t.Fatalf("status = %q", e.Status)
			}
		})
	}
}

func TestClientOrderIDAcceptsMaxHyphenated(t *testing.T) {
	t.Parallel()
	req := types.NewMarginMarketOrder(types.OrderSideBuy, "btc", "usdt", "1")
	req.ClientOrderID = strings.Repeat("A", 16) + "-" + strings.Repeat("9", 15) // 32 chars
	if len(req.ClientOrderID) != 32 {
		t.Fatalf("len = %d", len(req.ClientOrderID))
	}
	if _, err := req.Prepare(); err != nil {
		t.Fatal(err)
	}
}
