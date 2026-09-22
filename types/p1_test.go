package types_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestMoneyUnmarshalStringAndNumber(t *testing.T) {
	t.Parallel()
	var s struct {
		A types.Money `json:"a"`
		B types.Money `json:"b"`
		C types.Money `json:"c"`
	}
	if err := json.Unmarshal([]byte(`{"a":"1.50","b":18.221362316,"c":null}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.A != "1.50" || s.B != "18.221362316" || s.C != "" {
		t.Fatalf("got A=%q B=%q C=%q", s.A, s.B, s.C)
	}
}

func TestJSONInt64StringOrNumber(t *testing.T) {
	t.Parallel()
	var n types.JSONInt64
	if err := json.Unmarshal([]byte(`"1790066825745"`), &n); err != nil {
		t.Fatal(err)
	}
	if n.Int64() != 1790066825745 {
		t.Fatalf("n=%d", n.Int64())
	}
	if err := json.Unmarshal([]byte(`42`), &n); err != nil {
		t.Fatal(err)
	}
	if n.Int64() != 42 {
		t.Fatalf("n=%d", n.Int64())
	}
}

func TestStatsKey(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"BTCIRT":   "btc-rls",
		"btcirt":   "btc-rls",
		"BTCUSDT":  "btc-usdt",
		"btc-rls":  "btc-rls",
		"BTC-USDT": "btc-usdt",
	}
	for in, want := range cases {
		if got := types.StatsKey(in); got != want {
			t.Errorf("StatsKey(%q)=%q want %q", in, got, want)
		}
	}
}

func TestMarketStatsLookup(t *testing.T) {
	t.Parallel()
	var resp types.MarketStatsResponse
	body := []byte(`{"status":"ok","stats":{"btc-rls":{"isClosed":false,"latest":"100","dayChange":"3.15"}}}`)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	st, ok := resp.Stat("BTCIRT")
	if !ok || types.MoneyValue(st.Latest) != "100" {
		t.Fatalf("stat=%+v ok=%v", st, ok)
	}
	if st.DayChange == nil || *st.DayChange != "3.15" {
		t.Fatalf("dayChange=%v", st.DayChange)
	}
}

func TestUDFHistoryCandles(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"s":"ok","t":[1,2],"o":[10.5,11],"h":[12,13],"l":[9,10],"c":[11,12.5],"v":[1.25,0.5]}`)
	var h types.UDFHistory
	if err := json.Unmarshal(raw, &h); err != nil {
		t.Fatal(err)
	}
	if !h.IsOK() {
		t.Fatal(h.S)
	}
	cs := h.Candles()
	if len(cs) != 2 || cs[0].Open != "10.5" || cs[1].Volume != "0.5" {
		t.Fatalf("candles=%+v", cs)
	}
}

func TestUDFHistoryQueryNormalize(t *testing.T) {
	t.Parallel()
	q, err := (types.UDFHistoryQuery{Symbol: " btcirt ", Resolution: " D ", To: 10}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if q.Symbol != "BTCIRT" || q.Resolution != "D" {
		t.Fatalf("%+v", q)
	}
	if _, err := (types.UDFHistoryQuery{Symbol: "BTCIRT", Resolution: "D"}).Normalize(); err == nil {
		t.Fatal("expected error for missing to")
	}
	if _, err := (types.UDFHistoryQuery{Symbol: "all", Resolution: "D", To: 1}).Normalize(); err == nil {
		t.Fatal("expected error for all")
	}
}

func TestMarketStatsQueryCurrencyList(t *testing.T) {
	t.Parallel()
	q := types.MarketStatsQuery{SrcCurrency: " BTC, ETH ", DstCurrency: "RLS"}.Normalize()
	if q.SrcCurrency != "btc,eth" || q.DstCurrency != "rls" {
		t.Fatalf("%+v", q)
	}
}

func TestSpotOrderPrepareOCOAndLimit(t *testing.T) {
	t.Parallel()
	lim := types.NewSpotLimitOrder(types.OrderSideBuy, "BTC", "RLS", "0.6", "520000000")
	lim.ClientOrderID = "order1"
	wire, err := lim.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if wire.SrcCurrency != "btc" || wire.Execution != types.ExecutionLimit || wire.Mode != "" {
		t.Fatalf("%+v", wire)
	}

	oco := types.NewSpotOCOOrder(types.OrderSideBuy, "btc", "usdt", "0.01", "42390", "42700", "42715")
	wire, err = oco.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if wire.Execution != types.ExecutionLimit || wire.Mode != types.OrderModeOCO {
		t.Fatalf("%+v", wire)
	}

	bad := types.NewSpotLimitOrder(types.OrderSideBuy, "btc", "rls", "0.6", "")
	if _, err := bad.Prepare(); err == nil || !strings.Contains(err.Error(), "price") {
		t.Fatalf("err=%v", err)
	}
}

func TestSpotOrderStatusRequiresIdentifier(t *testing.T) {
	t.Parallel()
	if _, err := (types.SpotOrderStatusRequest{}).Prepare(); err == nil {
		t.Fatal("expected error")
	}
	wire, err := (types.SpotOrderStatusRequest{ID: 5684}).Prepare()
	if err != nil || wire.ID != 5684 {
		t.Fatalf("%+v %v", wire, err)
	}
}

func TestUserTradesQueryPairing(t *testing.T) {
	t.Parallel()
	if _, err := (types.UserTradesQuery{SrcCurrency: "btc"}).Normalize(); err == nil {
		t.Fatal("expected pairing error")
	}
	q, err := (types.UserTradesQuery{SrcCurrency: "USDT", DstCurrency: "RLS", TradeOrder: "ASC"}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if q.SrcCurrency != "usdt" || q.DstCurrency != "rls" || q.TradeOrder != "asc" {
		t.Fatalf("%+v", q)
	}
}

func TestWalletTransferPrepare(t *testing.T) {
	t.Parallel()
	wire, err := types.NewSpotToMarginTransfer(" USDT ", "0.01").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if wire.Currency != "usdt" || wire.Src != types.WalletSpot || wire.Dst != types.WalletMargin {
		t.Fatalf("%+v", wire)
	}
	same := types.WalletTransferRequest{Currency: "btc", Amount: "1", Src: types.WalletSpot, Dst: types.WalletSpot}
	if _, err := same.Prepare(); err == nil {
		t.Fatal("expected same src/dst error")
	}
}

func TestDelegationLimitFor(t *testing.T) {
	t.Parallel()
	resp := types.DelegationLimitResponse{
		Limits: types.DelegationLimits{
			Sell: []types.LeverageLimit{{Leverage: "2", Limit: "0.37"}},
			Buy:  []types.LeverageLimit{{Leverage: "2", Limit: "416000000"}},
		},
	}
	lim, ok := resp.LimitFor(types.OrderSideSell, "2")
	if !ok || lim != "0.37" {
		t.Fatalf("sell=%q ok=%v", lim, ok)
	}
	if _, ok := resp.LimitFor(types.OrderSideBuy, "9"); ok {
		t.Fatal("unexpected leverage")
	}
}

func TestWebSocketChannelHelpers(t *testing.T) {
	t.Parallel()
	ov := types.DefaultWebSocketOverview()
	if ov.ProductionURL != types.DefaultWebSocketURL || ov.TokenPath != types.WebSocketTokenPath {
		t.Fatalf("%+v", ov)
	}
	if got := types.PublicOrderBookChannel("btcirt"); got != "public:orderbook-BTCIRT" {
		t.Fatal(got)
	}
	if got := types.PublicCandleChannel("BTCIRT", "15"); got != "public:candle-BTCIRT-15" {
		t.Fatal(got)
	}
	if got := types.PublicTradesChannel("BTCIRT"); got != "public:trades-BTCIRT" {
		t.Fatal(got)
	}
	if got := types.PublicMarketStatsChannel("all"); got != "public:market-stats-all" {
		t.Fatal(got)
	}
	if got := types.PublicMarketStatsChannel("BTCIRT"); got != "public:market-stats-BTCIRT" {
		t.Fatal(got)
	}
	if got := types.PrivateOrdersChannel("abc"); got != "private:orders#abc" {
		t.Fatal(got)
	}
	if got := types.PrivateTradesChannel("abc"); got != "private:trades#abc" {
		t.Fatal(got)
	}
	if len(ov.PublicChannels) != 5 || len(ov.PrivateChannels) != 2 {
		t.Fatalf("channels public=%d private=%d", len(ov.PublicChannels), len(ov.PrivateChannels))
	}
}

func TestMarginMarketsLookup(t *testing.T) {
	t.Parallel()
	var resp types.MarginMarketsResponse
	body := []byte(`{"status":"ok","markets":{"BTCIRT":{"srcCurrency":"btc","dstCurrency":"rls","maxLeverage":"5","sellEnabled":true,"buyEnabled":true}}}`)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	m, ok := resp.Market("btcirt")
	if !ok || m.SrcCurrency != "btc" || m.MaxLeverage != "5" {
		t.Fatalf("%+v ok=%v", m, ok)
	}
}

func TestWalletBalanceAndListPrepare(t *testing.T) {
	t.Parallel()
	b, err := (types.WalletBalanceRequest{Currency: " BTC "}).Prepare()
	if err != nil || b.Currency != "btc" {
		t.Fatalf("%+v %v", b, err)
	}
	if _, err := (types.WalletBalanceRequest{}).Prepare(); err == nil {
		t.Fatal("expected currency required")
	}
	w, err := (types.WalletListRequest{Type: " MARGIN "}).Prepare()
	if err != nil || w.Type != types.WalletMargin {
		t.Fatalf("%+v %v", w, err)
	}
	v2, err := (types.WalletsV2Request{Currencies: "BTC, RLS", Type: "spot"}).Prepare()
	if err != nil || v2.Currencies != "btc,rls" {
		t.Fatalf("%+v %v", v2, err)
	}
}
