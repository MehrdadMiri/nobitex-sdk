package types_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestPositionListQueryNormalize(t *testing.T) {
	t.Parallel()
	got, err := types.PositionListQuery{
		SrcCurrency: " BTC ",
		DstCurrency: "RLS",
		Status:      "ACTIVE",
		Page:        2,
		PageSize:    10,
	}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if got.SrcCurrency != "btc" || got.DstCurrency != "rls" || got.Status != types.PositionListActive {
		t.Fatalf("got %+v", got)
	}
	if got.Page != 2 || got.PageSize != 10 {
		t.Fatalf("paging %+v", got)
	}

	empty, err := types.PositionListQuery{}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if empty.Status != "" {
		t.Fatalf("empty query should leave status unset, got %q", empty.Status)
	}
}

func TestPositionListQueryRejectsInvalid(t *testing.T) {
	t.Parallel()
	if _, err := (types.PositionListQuery{Status: "Open"}).Normalize(); err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("err = %v", err)
	}
	if _, err := (types.PositionListQuery{Page: -1}).Normalize(); err == nil || !strings.Contains(err.Error(), "page") {
		t.Fatalf("err = %v", err)
	}
	if _, err := (types.PositionListQuery{PageSize: -3}).Normalize(); err == nil || !strings.Contains(err.Error(), "pageSize") {
		t.Fatalf("err = %v", err)
	}
}

func TestPositionStatusHelpers(t *testing.T) {
	t.Parallel()
	if !types.PositionStatusOpen.IsOpen() || types.PositionStatusOpen.IsSettled() {
		t.Fatal("Open")
	}
	for _, s := range []types.PositionStatus{
		types.PositionStatusClosed, types.PositionStatusLiquidated, types.PositionStatusExpired,
	} {
		if s.IsOpen() || !s.IsSettled() {
			t.Fatalf("%q", s)
		}
	}
}

func TestUnmarshalPositionsListFixture(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("..", "client", "testdata", "positions_list.json"))
	if err != nil {
		t.Fatal(err)
	}
	var list types.PositionListResponse
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if !list.Status.IsOK() || list.HasNext || len(list.Positions) != 2 {
		t.Fatalf("list = %+v", list)
	}

	open, ok := list.Position(128)
	if !ok || !open.IsOpen() || open.Side != types.PositionDirectionSell {
		t.Fatalf("open = %+v ok=%v", open, ok)
	}
	if open.ID != 128 || open.Status != types.PositionStatusOpen {
		t.Fatalf("id/status = %d %q", open.ID, open.Status)
	}
	if open.SrcCurrency != "btc" || open.DstCurrency != "rls" {
		t.Fatalf("market = %s/%s", open.SrcCurrency, open.DstCurrency)
	}
	if open.MarginType != types.MarginTypeIsolated || open.Leverage != "2" {
		t.Fatalf("margin = %+v", open)
	}
	if open.ClosedAt != nil || open.ExitPrice != nil || open.PNL != nil {
		t.Fatalf("open row should have null closed/exit/PNL: %+v", open)
	}
	if types.MoneyValue(open.EntryPrice) != "6400000000" {
		t.Fatalf("entryPrice = %q", types.MoneyValue(open.EntryPrice))
	}
	if types.MoneyValue(open.UnrealizedPNL) != "-576435" {
		t.Fatalf("unrealizedPNL = %q", types.MoneyValue(open.UnrealizedPNL))
	}
	if open.ExpirationDate == nil || *open.ExpirationDate != "2022-11-20" {
		t.Fatalf("expirationDate = %v", open.ExpirationDate)
	}
	if open.OpenedAt == nil || open.OpenedAt.Year() != 2022 {
		t.Fatalf("openedAt = %v", open.OpenedAt)
	}

	past, ok := list.Position(32)
	if !ok || !past.IsSettled() || past.Status != types.PositionStatusClosed {
		t.Fatalf("past = %+v ok=%v", past, ok)
	}
	if past.ClosedAt == nil || types.MoneyValue(past.PNL) != "118.46" {
		t.Fatalf("past settled fields = %+v", past)
	}
	if past.Liability != nil || past.UnrealizedPNL != nil {
		t.Fatalf("past row should omit active-only fields: %+v", past)
	}
	if _, ok := list.Position(999); ok {
		t.Fatal("missing id should be absent")
	}
}

func TestPrepareCloseLimitMarketStopAndOCO(t *testing.T) {
	t.Parallel()

	limit, err := types.NewCloseLimitOrder("0.0100150225", "6200000000").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if limit.Execution != types.CloseExecutionLimit || limit.Price != "6200000000" || limit.Mode != "" {
		t.Fatalf("limit = %+v", limit)
	}

	market, err := types.NewCloseMarketOrder("0.01").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if market.Execution != types.CloseExecutionMarket || market.Price != "" {
		t.Fatalf("market = %+v", market)
	}

	sl, err := types.NewCloseStopLimitOrder("0.01", "6200000000", "6100000000").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if sl.Execution != types.CloseExecutionStopLimit || sl.StopPrice != "6100000000" {
		t.Fatalf("stop_limit = %+v", sl)
	}

	sm, err := types.NewCloseStopMarketOrder("0.01", "6100000000").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if sm.Execution != types.CloseExecutionStopMarket || sm.StopPrice != "6100000000" {
		t.Fatalf("stop_market = %+v", sm)
	}

	oco, err := types.NewCloseOCOOrder("0.01", "12600000000", "13600000000", "13610000000").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if oco.Execution != types.CloseExecutionLimit || oco.Mode != types.CloseOrderModeOCO {
		t.Fatalf("oco wire = %+v", oco)
	}
}

func TestPrepareCloseOCOConvenienceExecution(t *testing.T) {
	t.Parallel()
	req := types.ClosePositionRequest{
		Execution:      types.CloseExecutionOCO,
		Amount:         "0.01",
		Price:          "12600000000",
		StopPrice:      "13600000000",
		StopLimitPrice: "13610000000",
	}
	if !req.IsOCO() {
		t.Fatal("CloseExecutionOCO should report IsOCO")
	}
	got, err := req.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if got.Execution != types.CloseExecutionLimit || got.Mode != types.CloseOrderModeOCO {
		t.Fatalf("got %+v", got)
	}
}

func TestPrepareCloseDefaultsEmptyExecutionToLimit(t *testing.T) {
	t.Parallel()
	got, err := types.ClosePositionRequest{Amount: "0.01", Price: "1"}.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if got.Execution != types.CloseExecutionLimit {
		t.Fatalf("got %+v", got)
	}
}

func TestPrepareCloseRejectsInvalid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		req  types.ClosePositionRequest
		want string
	}{
		{name: "missing amount", req: types.NewCloseLimitOrder("", "1"), want: "amount"},
		{name: "limit missing price", req: types.NewCloseLimitOrder("1", ""), want: "price"},
		{name: "stop_market missing stop", req: types.NewCloseStopMarketOrder("1", ""), want: "stopPrice"},
		{name: "oco missing stopLimitPrice", req: types.NewCloseOCOOrder("1", "1", "2", ""), want: "stopLimitPrice"},
		{name: "bad execution", req: types.ClosePositionRequest{Execution: "iceberg", Amount: "1"}, want: "unsupported close execution"},
		{
			name: "oco with market execution",
			req: types.ClosePositionRequest{
				Execution: types.CloseExecutionMarket, Mode: types.CloseOrderModeOCO,
				Amount: "1", Price: "1", StopPrice: "2", StopLimitPrice: "3",
			},
			want: "OCO close requires execution",
		},
		{
			name: "clientOrderId punctuation",
			req: func() types.ClosePositionRequest {
				r := types.NewCloseMarketOrder("1")
				r.ClientOrderID = "bad id!"
				return r
			}(),
			want: "clientOrderId",
		},
		{
			name: "clientOrderId too long",
			req: func() types.ClosePositionRequest {
				r := types.NewCloseMarketOrder("1")
				r.ClientOrderID = strings.Repeat("a", 33)
				return r
			}(),
			want: "clientOrderId",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.req.Prepare()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestPrepareCloseOmitsEmptyOptionalJSON(t *testing.T) {
	t.Parallel()
	req, err := types.NewCloseLimitOrder("0.01", "6200000000").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, unexpected := range []string{"stopPrice", "stopLimitPrice", "mode", "clientOrderId"} {
		if strings.Contains(s, unexpected) {
			t.Fatalf("limit JSON unexpectedly includes %s: %s", unexpected, s)
		}
	}
	if !strings.Contains(s, `"execution":"limit"`) || !strings.Contains(s, `"amount":"0.01"`) {
		t.Fatalf("json = %s", s)
	}
}

func TestPrepareCloseOCOJSON(t *testing.T) {
	t.Parallel()
	req, err := types.NewCloseOCOOrder("0.01", "12600000000", "13600000000", "13610000000").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	req.ClientOrderID = "close-position-128"
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["execution"] != "limit" || m["mode"] != "oco" {
		t.Fatalf("wire = %s", raw)
	}
	if m["stopPrice"] != "13600000000" || m["stopLimitPrice"] != "13610000000" {
		t.Fatalf("wire = %s", raw)
	}
	if m["clientOrderId"] != "close-position-128" {
		t.Fatalf("wire = %s", raw)
	}
}

func TestUnmarshalCloseLimitAndOCOFixtures(t *testing.T) {
	t.Parallel()

	limitRaw, err := os.ReadFile(filepath.Join("..", "client", "testdata", "close_position_limit.json"))
	if err != nil {
		t.Fatal(err)
	}
	var limit types.ClosePositionResponse
	if err := json.Unmarshal(limitRaw, &limit); err != nil {
		t.Fatal(err)
	}
	if limit.Order == nil || limit.Order.ID != 28 || limit.Order.Side != types.CloseOrderSideClose {
		t.Fatalf("limit = %+v", limit.Order)
	}
	if limit.Order.Execution != types.CloseExecutionResponseLimit {
		t.Fatalf("execution = %q", limit.Order.Execution)
	}
	if limit.Order.ClientOrderIDValue() != "close-position-128" {
		t.Fatalf("clientOrderId = %q", limit.Order.ClientOrderIDValue())
	}
	if limit.Order.CreatedAt.Year() != 2022 {
		t.Fatalf("created_at = %v", limit.Order.CreatedAt)
	}
	if ids := limit.OrderIDs(); len(ids) != 1 || ids[0] != 28 {
		t.Fatalf("OrderIDs = %v", ids)
	}

	ocoRaw, err := os.ReadFile(filepath.Join("..", "client", "testdata", "close_position_oco.json"))
	if err != nil {
		t.Fatal(err)
	}
	var oco types.ClosePositionResponse
	if err := json.Unmarshal(ocoRaw, &oco); err != nil {
		t.Fatal(err)
	}
	orders := oco.CloseOrders()
	if len(orders) != 2 || orders[0].ID != 29 || orders[1].ID != 30 {
		t.Fatalf("orders = %+v", orders)
	}
	if orders[1].Execution != types.CloseExecutionResponseStopLimit || orders[1].Param1 != "13600000000" {
		t.Fatalf("stop leg = %+v", orders[1])
	}
	if orders[0].PairID == nil || *orders[0].PairID != 30 {
		t.Fatalf("pairId = %v", orders[0].PairID)
	}
}
