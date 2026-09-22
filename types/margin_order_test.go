package types_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestPrepareLimitMarketStopAndOCO(t *testing.T) {
	t.Parallel()

	limit, err := types.NewMarginLimitOrder(types.OrderSideSell, " BTC ", "USDT", "0.01", "13400000000").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if limit.Execution != types.ExecutionLimit || limit.SrcCurrency != "btc" || limit.DstCurrency != "usdt" {
		t.Fatalf("limit = %+v", limit)
	}
	if limit.Mode != "" {
		t.Fatalf("limit should not send mode, got %q", limit.Mode)
	}

	market, err := types.NewMarginMarketOrder(types.OrderSideBuy, "btc", "usdt", "0.01").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if market.Execution != types.ExecutionMarket || market.Price != "" {
		t.Fatalf("market = %+v", market)
	}

	sl, err := types.NewMarginStopLimitOrder(types.OrderSideSell, "btc", "usdt", "0.01", "12500000000", "12600000000").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if sl.Execution != types.ExecutionStopLimit || sl.Price == "" || sl.StopPrice == "" {
		t.Fatalf("stop_limit = %+v", sl)
	}

	sm, err := types.NewMarginStopMarketOrder(types.OrderSideSell, "btc", "usdt", "0.01", "12600000000").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if sm.Execution != types.ExecutionStopMarket || sm.StopPrice != "12600000000" {
		t.Fatalf("stop_market = %+v", sm)
	}

	oco, err := types.NewMarginOCOOrder(types.OrderSideBuy, "btc", "rls", "0.01", "13400000000", "12600000000", "12500000000").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if oco.Execution != types.ExecutionLimit || oco.Mode != types.OrderModeOCO {
		t.Fatalf("oco wire = %+v", oco)
	}
	if oco.StopLimitPrice != "12500000000" {
		t.Fatalf("stopLimitPrice = %q", oco.StopLimitPrice)
	}
}

func TestPrepareOCOConvenienceExecution(t *testing.T) {
	t.Parallel()
	req := types.MarginOrderRequest{
		Execution:      types.ExecutionOCO,
		Type:           types.OrderSideSell,
		SrcCurrency:    "btc",
		DstCurrency:    "usdt",
		Amount:         "0.01",
		Price:          "13400000000",
		StopPrice:      "12600000000",
		StopLimitPrice: "12500000000",
	}
	if !req.IsOCO() {
		t.Fatal("ExecutionOCO should report IsOCO")
	}
	got, err := req.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if got.Execution != types.ExecutionLimit || got.Mode != types.OrderModeOCO {
		t.Fatalf("got %+v", got)
	}
}

func TestPrepareRejectsInvalid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		req  types.MarginOrderRequest
		want string
	}{
		{
			name: "missing src",
			req:  types.MarginOrderRequest{Execution: types.ExecutionLimit, Type: types.OrderSideBuy, DstCurrency: "usdt", Amount: "1", Price: "1"},
			want: "srcCurrency",
		},
		{
			name: "missing type",
			req:  types.NewMarginLimitOrder("", "btc", "usdt", "1", "1"),
			want: "type must be",
		},
		{
			name: "limit missing price",
			req:  types.NewMarginLimitOrder(types.OrderSideBuy, "btc", "usdt", "1", ""),
			want: "price",
		},
		{
			name: "stop_market missing stop",
			req:  types.NewMarginStopMarketOrder(types.OrderSideBuy, "btc", "usdt", "1", ""),
			want: "stopPrice",
		},
		{
			name: "oco missing stopLimitPrice",
			req:  types.NewMarginOCOOrder(types.OrderSideBuy, "btc", "usdt", "1", "1", "2", ""),
			want: "stopLimitPrice",
		},
		{
			name: "bad execution",
			req:  types.MarginOrderRequest{Execution: "iceberg", Type: types.OrderSideBuy, SrcCurrency: "btc", DstCurrency: "usdt", Amount: "1"},
			want: "unsupported margin execution",
		},
		{
			name: "oco with market execution",
			req: types.MarginOrderRequest{
				Execution: types.ExecutionMarket, Mode: types.OrderModeOCO, Type: types.OrderSideBuy,
				SrcCurrency: "btc", DstCurrency: "usdt", Amount: "1", Price: "1", StopPrice: "2", StopLimitPrice: "3",
			},
			want: "OCO orders require execution",
		},
		{
			name: "clientOrderId punctuation",
			req: func() types.MarginOrderRequest {
				r := types.NewMarginMarketOrder(types.OrderSideBuy, "btc", "usdt", "1")
				r.ClientOrderID = "bad id!"
				return r
			}(),
			want: "clientOrderId",
		},
		{
			name: "clientOrderId too long",
			req: func() types.MarginOrderRequest {
				r := types.NewMarginMarketOrder(types.OrderSideBuy, "btc", "usdt", "1")
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

func TestPrepareOmitsEmptyOptionalJSON(t *testing.T) {
	t.Parallel()
	req, err := types.NewMarginLimitOrder(types.OrderSideSell, "btc", "usdt", "0.01", "13400000000").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, unexpected := range []string{"stopPrice", "stopLimitPrice", "mode", "clientOrderId", "leverage"} {
		if strings.Contains(s, unexpected) {
			t.Fatalf("limit JSON unexpectedly includes %s: %s", unexpected, s)
		}
	}
	if !strings.Contains(s, `"execution":"limit"`) || !strings.Contains(s, `"type":"sell"`) {
		t.Fatalf("json = %s", s)
	}
}

func TestPrepareOCOJSON(t *testing.T) {
	t.Parallel()
	req, err := types.NewMarginOCOOrder(types.OrderSideBuy, "btc", "usdt", "0.01", "42390", "42700", "42715").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	req.ClientOrderID = "order1"
	req.Leverage = "2"
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
	if m["stopPrice"] != "42700" || m["stopLimitPrice"] != "42715" || m["price"] != "42390" {
		t.Fatalf("prices = %s", raw)
	}
}

func TestMarginOrderAddResponseParsesDocumentedLimit(t *testing.T) {
	t.Parallel()
	const body = `{
  "status": "ok",
  "order": {
    "id": 25,
    "type": "sell",
    "execution": "Limit",
    "tradeType": "Margin",
    "srcCurrency": "btc",
    "dstCurrency": "rls",
    "price": "6400000000",
    "amount": "0.01",
    "status": "Active",
    "totalPrice": "0",
    "totalOrderPrice": "64000000",
    "matchedAmount": "0",
    "unmatchedAmount": "0.01",
    "leverage": "2",
    "side": "open",
    "partial": false,
    "fee": "0",
    "created_at": "2022-10-20T11:36:13.592827+00:00",
    "averagePrice": "0"
  }
}`
	var resp types.MarginOrderAddResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Status.IsOK() || resp.Order == nil {
		t.Fatalf("status=%q order=%v", resp.Status, resp.Order)
	}
	if resp.Order.ID != 25 || resp.Order.ClientOrderIDValue() != "" {
		t.Fatalf("ids: id=%d client=%q", resp.Order.ID, resp.Order.ClientOrderIDValue())
	}
	if resp.Order.TradeType != types.TradeTypeMargin || resp.Order.Side != types.PositionSideOpen {
		t.Fatalf("order = %+v", resp.Order)
	}
	wantCreated, err := time.Parse(time.RFC3339Nano, "2022-10-20T11:36:13.592827+00:00")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Order.CreatedAt.Equal(wantCreated) {
		t.Fatalf("created_at = %v", resp.Order.CreatedAt)
	}
	ids := resp.OrderIDs()
	if len(ids) != 1 || ids[0] != 25 {
		t.Fatalf("OrderIDs = %v", ids)
	}
}

func TestMarginOrderAddResponseParsesOCOAndNullClientOrderID(t *testing.T) {
	t.Parallel()
	const body = `{
  "status": "ok",
  "orders": [
    {
      "id": 29,
      "type": "buy",
      "execution": "Limit",
      "tradeType": "Margin",
      "srcCurrency": "btc",
      "dstCurrency": "rls",
      "price": "12600000000",
      "amount": "0.01",
      "status": "Active",
      "totalPrice": "0",
      "totalOrderPrice": "126000000",
      "matchedAmount": "0",
      "unmatchedAmount": "0.01",
      "clientOrderId": "oco-buy-1",
      "pairId": 30,
      "leverage": "2",
      "side": "open",
      "partial": false,
      "fee": "0",
      "created_at": "2022-10-25T09:57:38.560820+00:00",
      "averagePrice": "0"
    },
    {
      "id": 30,
      "type": "buy",
      "execution": "StopLimit",
      "tradeType": "Margin",
      "srcCurrency": "btc",
      "dstCurrency": "rls",
      "price": "13610000000",
      "amount": "0.01",
      "status": "Inactive",
      "totalPrice": "0",
      "totalOrderPrice": "136100000",
      "matchedAmount": "0",
      "unmatchedAmount": "0.01",
      "clientOrderId": null,
      "param1": "13600000000",
      "pairId": 29,
      "leverage": "2",
      "side": "open",
      "partial": false,
      "fee": "0",
      "created_at": "2022-10-25T09:57:38.560820+00:00",
      "averagePrice": "0"
    }
  ]
}`
	var resp types.MarginOrderAddResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatal(err)
	}
	orders := resp.PlacedOrders()
	if len(orders) != 2 {
		t.Fatalf("placed = %d", len(orders))
	}
	if orders[0].ID != 29 || orders[0].PairID == nil || *orders[0].PairID != 30 {
		t.Fatalf("leg0 = %+v", orders[0])
	}
	if orders[0].ClientOrderIDValue() != "oco-buy-1" {
		t.Fatalf("client = %q", orders[0].ClientOrderIDValue())
	}
	if orders[1].ClientOrderID != nil {
		t.Fatalf("leg1 clientOrderId should be null, got %v", orders[1].ClientOrderID)
	}
	if orders[1].Param1 != "13600000000" || orders[1].PairID == nil || *orders[1].PairID != 29 {
		t.Fatalf("leg1 = %+v", orders[1])
	}
	ids := resp.OrderIDs()
	if len(ids) != 2 || ids[0] != 29 || ids[1] != 30 {
		t.Fatalf("OrderIDs = %v", ids)
	}
}
