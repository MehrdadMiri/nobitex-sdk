package types_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestOrderListQueryPrepareMarginFilter(t *testing.T) {
	t.Parallel()
	q, err := types.NewMarginOrderListQuery().Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if q.TradeType != types.TradeTypeFilterMargin {
		t.Fatalf("tradeType = %q", q.TradeType)
	}

	q, err = types.OrderListQuery{
		Status:      " OPEN ",
		Type:        "SELL",
		Execution:   "Stop_Limit",
		TradeType:   types.TradeTypeMargin.Filter(),
		SrcCurrency: " BTC ",
		DstCurrency: "USDT",
		Details:     types.OrderListDetailsFull,
		Sort:        "-ID",
		Page:        2,
		PageSize:    50,
	}.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if q.Status != types.OrderListStatusOpen || q.Type != types.OrderSideSell {
		t.Fatalf("status/type = %+v", q)
	}
	if q.Execution != types.ExecutionStopLimit || q.TradeType != types.TradeTypeFilterMargin {
		t.Fatalf("exec/tradeType = %+v", q)
	}
	if q.SrcCurrency != "btc" || q.DstCurrency != "usdt" || q.Sort != types.OrderListSortIDDesc {
		t.Fatalf("normalized = %+v", q)
	}
}

func TestOrderListQueryWithMarginKeepsFilters(t *testing.T) {
	t.Parallel()
	q := types.OrderListQuery{Status: types.OrderListStatusAll, Details: 2, TradeType: types.TradeTypeFilterSpot}.WithMargin()
	if q.TradeType != types.TradeTypeFilterMargin || q.Status != types.OrderListStatusAll || q.Details != 2 {
		t.Fatalf("q = %+v", q)
	}
}

func TestOrderListQueryPrepareRejects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		q    types.OrderListQuery
		want string
	}{
		{name: "status", q: types.OrderListQuery{Status: "Active"}, want: "status"},
		{name: "type", q: types.OrderListQuery{Type: "long"}, want: "type"},
		{name: "execution oco", q: types.OrderListQuery{Execution: types.ExecutionOCO}, want: "execution"},
		{name: "tradeType", q: types.OrderListQuery{TradeType: "credit"}, want: "tradeType"},
		{name: "details", q: types.OrderListQuery{Details: 3}, want: "details"},
		{name: "page and fromId", q: types.OrderListQuery{Page: 1, FromID: 100}, want: "fromId"},
		{name: "pageSize", q: types.OrderListQuery{PageSize: 1001}, want: "pageSize"},
		{name: "sort", q: types.OrderListQuery{Sort: "amount"}, want: "sort"},
		{name: "fromId negative", q: types.OrderListQuery{FromID: -1}, want: "fromId"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.q.Prepare()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v want substring %q", err, tc.want)
			}
		})
	}
}

func TestOrderListQueryJSONOmitsEmptyAndRenamesSort(t *testing.T) {
	t.Parallel()
	q, err := types.OrderListQuery{
		TradeType: types.TradeTypeFilterMargin,
		Details:   2,
		Sort:      types.OrderListSortCreatedAtDesc,
	}.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["tradeType"] != "margin" || m["order"] != "-created_at" {
		t.Fatalf("wire = %s", raw)
	}
	if _, ok := m["status"]; ok {
		t.Fatalf("empty status should be omitted: %s", raw)
	}
	if int(m["details"].(float64)) != 2 {
		t.Fatalf("details = %v", m["details"])
	}
}

func TestOrderListResponseParsesMarginAndSpot(t *testing.T) {
	t.Parallel()
	const body = `{
  "status": "ok",
  "orders": [
    {
      "id": 173546223,
      "type": "sell",
      "execution": "Limit",
      "tradeType": "Spot",
      "market": "BTC-USDT",
      "srcCurrency": "Bitcoin",
      "dstCurrency": "Tether",
      "price": "9750.01",
      "amount": "0.0123",
      "totalPrice": "0",
      "totalOrderPrice": "119.925123",
      "matchedAmount": "0",
      "unmatchedAmount": "0.0123",
      "status": "Active",
      "partial": false,
      "fee": "0",
      "created_at": "2020-07-15T11:32:38.326809+00:00",
      "averagePrice": "0",
      "clientOrderId": "order1"
    },
    {
      "id": 173546224,
      "type": "buy",
      "execution": "StopLimit",
      "tradeType": "Margin",
      "market": "BTC-USDT",
      "srcCurrency": "Bitcoin",
      "dstCurrency": "Tether",
      "price": "9600",
      "amount": "0.02",
      "status": "Inactive",
      "partial": false,
      "fee": "0",
      "created_at": "2020-07-15T11:40:00.000000+00:00",
      "averagePrice": "0",
      "clientOrderId": null,
      "pairId": 173546226,
      "param1": "9500"
    }
  ],
  "hasNext": true
}`
	var resp types.OrderListResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Status.IsOK() || !resp.HasNext || len(resp.Orders) != 2 {
		t.Fatalf("resp = %+v", resp)
	}
	spot := resp.Orders[0]
	if spot.ID != 173546223 || spot.IsMargin() || spot.ClientOrderIDValue() != "order1" {
		t.Fatalf("spot = %+v", spot)
	}
	if !spot.Status.IsOpen() {
		t.Fatalf("spot status should be open: %q", spot.Status)
	}
	margin := resp.Orders[1]
	if !margin.IsMargin() || margin.PairID == nil || *margin.PairID != 173546226 || margin.Param1 != "9500" {
		t.Fatalf("margin = %+v", margin)
	}
	if margin.ClientOrderID != nil {
		t.Fatalf("margin clientOrderId should be null, got %v", margin.ClientOrderID)
	}
	wantCreated, err := time.Parse(time.RFC3339Nano, "2020-07-15T11:32:38.326809+00:00")
	if err != nil {
		t.Fatal(err)
	}
	if !spot.CreatedAt.Equal(wantCreated) {
		t.Fatalf("created_at = %v", spot.CreatedAt)
	}
	ids := resp.OrderIDs()
	if len(ids) != 2 || ids[0] != 173546223 || ids[1] != 173546224 {
		t.Fatalf("OrderIDs = %v", ids)
	}
	onlyMargin := resp.MarginOrders()
	if len(onlyMargin) != 1 || onlyMargin[0].ID != 173546224 {
		t.Fatalf("MarginOrders = %+v", onlyMargin)
	}
}

func TestCancelOrderRequestPrepareByIDAndClientOrderID(t *testing.T) {
	t.Parallel()
	byID, err := types.NewCancelOrderByID(5684).Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if byID.Order != 5684 || byID.ClientOrderID != "" || byID.Status != types.OrderCancelStatus {
		t.Fatalf("byID = %+v", byID)
	}
	raw, err := json.Marshal(byID)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"order":5684,"status":"canceled"}` {
		t.Fatalf("byID wire = %s", raw)
	}

	byClient, err := types.NewCancelOrderByClientOrderID(" order1 ").Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if byClient.Order != 0 || byClient.ClientOrderID != "order1" {
		t.Fatalf("byClient = %+v", byClient)
	}
	raw, err = json.Marshal(byClient)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"clientOrderId":"order1","status":"canceled"}` {
		t.Fatalf("byClient wire = %s", raw)
	}

	both, err := types.CancelOrderRequest{Order: 5684, ClientOrderID: "order1"}.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(both)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"order":5684`) || !strings.Contains(string(raw), `"clientOrderId":"order1"`) {
		t.Fatalf("both wire = %s", raw)
	}
}

func TestCancelOrderRequestPrepareRejects(t *testing.T) {
	t.Parallel()
	if _, err := (types.CancelOrderRequest{}).Prepare(); err == nil || !strings.Contains(err.Error(), "order id or clientOrderId") {
		t.Fatalf("empty err = %v", err)
	}
	if _, err := types.NewCancelOrderByID(-1).Prepare(); err == nil || !strings.Contains(err.Error(), "order id") {
		t.Fatalf("neg id err = %v", err)
	}
	if _, err := types.NewCancelOrderByClientOrderID("bad id").Prepare(); err == nil || !strings.Contains(err.Error(), "clientOrderId") {
		t.Fatalf("bad client err = %v", err)
	}
	if _, err := (types.CancelOrderRequest{Order: 1, Status: "done"}).Prepare(); err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("status err = %v", err)
	}
}

func TestUserOrderCancelRequestPrefersID(t *testing.T) {
	t.Parallel()
	cid := "order1"
	o := types.UserOrder{ID: 99, ClientOrderID: &cid, Status: types.OrderStatusActive}
	req := o.CancelRequest()
	if req.Order != 99 || req.ClientOrderID != "" || req.Status != types.OrderCancelStatus {
		t.Fatalf("req = %+v", req)
	}
	noID := types.UserOrder{ClientOrderID: &cid}
	req = noID.CancelRequest()
	if req.Order != 0 || req.ClientOrderID != "order1" {
		t.Fatalf("noID req = %+v", req)
	}
	if !types.OrderStatusCanceled.IsCanceled() || types.OrderStatusDone.IsOpen() {
		t.Fatal("status helpers")
	}
}

func TestCancelOrderResponseParsesDocumentedBody(t *testing.T) {
	t.Parallel()
	const body = `{
  "status": "ok",
  "updatedStatus": "Canceled",
  "order": {
    "amount": "60",
    "averagePrice": "0",
    "clientOrderId": null,
    "created_at": "2025-10-28T09:25:17.774332+00:00",
    "dstCurrency": "﷼",
    "execution": "Market",
    "fee": "0",
    "id": 5684,
    "market": "USDT-RLS",
    "matchedAmount": "0",
    "partial": false,
    "price": "market",
    "srcCurrency": "Tether",
    "status": "Canceled",
    "totalOrderPrice": "2550000",
    "totalPrice": "0",
    "tradeType": "Spot",
    "type": "sell",
    "unmatchedAmount": "60"
  }
}`
	var resp types.CancelOrderResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Status.IsOK() || resp.UpdatedStatus != types.OrderStatusCanceled || resp.Order == nil {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.Order.ID != 5684 || !resp.Order.Status.IsCanceled() || resp.Order.ClientOrderID != nil {
		t.Fatalf("order = %+v", resp.Order)
	}
	if resp.Order.Price != "market" || resp.Order.Market != "USDT-RLS" {
		t.Fatalf("order = %+v", resp.Order)
	}
}
