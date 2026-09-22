package types

import (
	"fmt"
	"strings"
	"time"
)

// Order list query `status` values (https://apidocs.nobitex.ir).
// These are not the same as OrderStatus (PascalCase lifecycle on each row).
const (
	OrderListStatusAll   OrderListStatus = "all"
	OrderListStatusOpen  OrderListStatus = "open"  // New, Active, Inactive (API default)
	OrderListStatusDone  OrderListStatus = "done"  // at least partially filled
	OrderListStatusClose OrderListStatus = "close" // Done or Canceled
)

// OrderListStatus is the GET/POST /market/orders/list `status` filter.
type OrderListStatus string

// Request-side tradeType filter (lowercase). Response bodies use TradeType
// PascalCase (Spot, Margin, …).
const (
	TradeTypeFilterSpot   TradeTypeFilter = "spot"
	TradeTypeFilterMargin TradeTypeFilter = "margin"
)

// TradeTypeFilter is /market/orders/list `tradeType`.
type TradeTypeFilter string

// Filter returns the lowercase list-query form of a response tradeType
// ("Margin" → "margin").
func (t TradeType) Filter() TradeTypeFilter {
	return TradeTypeFilter(strings.ToLower(strings.TrimSpace(string(t))))
}

// Order list `details` query: 2 includes id, status, fee, created_at, averagePrice.
const (
	OrderListDetailsBasic = 1
	OrderListDetailsFull  = 2
	orderListMaxPageSize  = 1000
)

// Documented `order` (sort) query values. Prefix `-` is descending.
const (
	OrderListSortIDAsc         = "id"
	OrderListSortIDDesc        = "-id"
	OrderListSortCreatedAtAsc  = "created_at"
	OrderListSortCreatedAtDesc = "-created_at"
	OrderListSortPriceAsc      = "price"
	OrderListSortPriceDesc     = "-price"
)

// OrderCancelStatus is the only supported POST /market/orders/update-status
// destination. API keys need TRADE for this call.
const OrderCancelStatus = "canceled"

// OrderListQuery is GET or POST /market/orders/list.
//
// Empty fields are omitted so API defaults apply (status=open, details=1,
// page=1, pageSize=100). Set TradeType to TradeTypeFilterMargin to list
// margin (تعهدی) orders. details=2 is required for id/status/fee/created_at
// /averagePrice — use that when the caller will cancel by order id.
//
// Do not send Page and FromID together. With FromID, only one page is
// returned and PageSize may be up to 1000.
type OrderListQuery struct {
	Status      OrderListStatus `json:"status,omitempty"`
	Type        OrderSide       `json:"type,omitempty"`
	Execution   Execution       `json:"execution,omitempty"`
	TradeType   TradeTypeFilter `json:"tradeType,omitempty"`
	SrcCurrency string          `json:"srcCurrency,omitempty"`
	DstCurrency string          `json:"dstCurrency,omitempty"`
	Details     int             `json:"details,omitempty"`
	FromID      int64           `json:"fromId,omitempty"`
	Sort        string          `json:"order,omitempty"`
	Page        int             `json:"page,omitempty"`
	PageSize    int             `json:"pageSize,omitempty"`
}

// NewMarginOrderListQuery returns a query that filters tradeType=margin.
func NewMarginOrderListQuery() OrderListQuery {
	return OrderListQuery{TradeType: TradeTypeFilterMargin}
}

// WithMargin returns a copy with tradeType=margin (other fields kept).
func (q OrderListQuery) WithMargin() OrderListQuery {
	q.TradeType = TradeTypeFilterMargin
	return q
}

// Prepare trims/lowercases filters and checks documented enums. It does not
// fill API defaults; omitting a field lets the server apply its default.
func (q OrderListQuery) Prepare() (OrderListQuery, error) {
	out := q
	out.Status = OrderListStatus(strings.ToLower(strings.TrimSpace(string(out.Status))))
	out.Type = OrderSide(strings.ToLower(strings.TrimSpace(string(out.Type))))
	out.Execution = Execution(strings.ToLower(strings.TrimSpace(string(out.Execution))))
	out.TradeType = TradeTypeFilter(strings.ToLower(strings.TrimSpace(string(out.TradeType))))
	out.SrcCurrency = strings.ToLower(strings.TrimSpace(out.SrcCurrency))
	out.DstCurrency = strings.ToLower(strings.TrimSpace(out.DstCurrency))
	out.Sort = strings.ToLower(strings.TrimSpace(out.Sort))

	switch out.Status {
	case "", OrderListStatusAll, OrderListStatusOpen, OrderListStatusDone, OrderListStatusClose:
	default:
		return OrderListQuery{}, fmt.Errorf("types: order list status must be all, open, done, or close")
	}
	if out.Type != "" && out.Type != OrderSideBuy && out.Type != OrderSideSell {
		return OrderListQuery{}, fmt.Errorf("types: order list type must be %q or %q", OrderSideBuy, OrderSideSell)
	}
	switch out.Execution {
	case "", ExecutionLimit, ExecutionMarket, ExecutionStopLimit, ExecutionStopMarket:
	default:
		return OrderListQuery{}, fmt.Errorf("types: order list execution must be limit, market, stop_limit, or stop_market")
	}
	switch out.TradeType {
	case "", TradeTypeFilterSpot, TradeTypeFilterMargin:
	default:
		return OrderListQuery{}, fmt.Errorf("types: order list tradeType must be %q or %q", TradeTypeFilterSpot, TradeTypeFilterMargin)
	}
	if out.Details != 0 && out.Details != OrderListDetailsBasic && out.Details != OrderListDetailsFull {
		return OrderListQuery{}, fmt.Errorf("types: order list details must be 1 or 2")
	}
	if out.FromID < 0 {
		return OrderListQuery{}, fmt.Errorf("types: order list fromId must be >= 1")
	}
	if out.Page < 0 {
		return OrderListQuery{}, fmt.Errorf("types: order list page must be >= 1")
	}
	if out.PageSize < 0 {
		return OrderListQuery{}, fmt.Errorf("types: order list pageSize must be >= 1")
	}
	if out.PageSize > orderListMaxPageSize {
		return OrderListQuery{}, fmt.Errorf("types: order list pageSize must be <= %d", orderListMaxPageSize)
	}
	if out.Page > 0 && out.FromID > 0 {
		return OrderListQuery{}, fmt.Errorf("types: order list page and fromId cannot be sent together")
	}
	switch out.Sort {
	case "", OrderListSortIDAsc, OrderListSortIDDesc, OrderListSortCreatedAtAsc, OrderListSortCreatedAtDesc, OrderListSortPriceAsc, OrderListSortPriceDesc:
	default:
		return OrderListQuery{}, fmt.Errorf("types: order list order (sort) must be id, -id, created_at, -created_at, price, or -price")
	}
	return out, nil
}

// UserOrder is one row from GET/POST /market/orders/list or the `order`
// object returned by POST /market/orders/update-status.
//
// ID and ClientOrderID are the identifiers used to cancel. With list
// details=1 the server may omit id, status, fee, created_at, and averagePrice.
type UserOrder struct {
	ID              int64       `json:"id"`
	Type            OrderSide   `json:"type"`
	Execution       string      `json:"execution"`
	TradeType       TradeType   `json:"tradeType"`
	Market          string      `json:"market"`
	SrcCurrency     string      `json:"srcCurrency"`
	DstCurrency     string      `json:"dstCurrency"`
	Price           Money       `json:"price"`
	Amount          Money       `json:"amount"`
	TotalPrice      Money       `json:"totalPrice"`
	TotalOrderPrice Money       `json:"totalOrderPrice"`
	MatchedAmount   Money       `json:"matchedAmount"`
	UnmatchedAmount Money       `json:"unmatchedAmount"`
	Status          OrderStatus `json:"status"`
	Partial         bool        `json:"partial"`
	Fee             Money       `json:"fee"`
	CreatedAt       time.Time   `json:"created_at"`
	AveragePrice    Money       `json:"averagePrice"`
	ClientOrderID   *string     `json:"clientOrderId"`
	PairID          *int64      `json:"pairId"`
	Param1          string      `json:"param1,omitempty"`
}

// ClientOrderIDValue returns the clientOrderId string, or "" when null/absent.
func (o UserOrder) ClientOrderIDValue() string {
	if o.ClientOrderID == nil {
		return ""
	}
	return *o.ClientOrderID
}

// IsMargin reports whether this row is a margin (تعهدی) order.
func (o UserOrder) IsMargin() bool {
	return o.TradeType == TradeTypeMargin || strings.EqualFold(string(o.TradeType), string(TradeTypeFilterMargin))
}

// CancelRequest builds POST /market/orders/update-status from this row.
// Order id is preferred; clientOrderId is used when id is missing (details=1).
func (o UserOrder) CancelRequest() CancelOrderRequest {
	if o.ID > 0 {
		return NewCancelOrderByID(o.ID)
	}
	return NewCancelOrderByClientOrderID(o.ClientOrderIDValue())
}

// OrderListResponse is GET/POST /market/orders/list.
type OrderListResponse struct {
	Envelope
	Orders  []UserOrder `json:"orders"`
	HasNext bool        `json:"hasNext"`
}

// OrderIDs returns every non-zero order id in the page.
func (r *OrderListResponse) OrderIDs() []int64 {
	if r == nil {
		return nil
	}
	ids := make([]int64, 0, len(r.Orders))
	for _, o := range r.Orders {
		if o.ID != 0 {
			ids = append(ids, o.ID)
		}
	}
	return ids
}

// MarginOrders returns rows whose tradeType is Margin.
func (r *OrderListResponse) MarginOrders() []UserOrder {
	if r == nil {
		return nil
	}
	out := make([]UserOrder, 0, len(r.Orders))
	for _, o := range r.Orders {
		if o.IsMargin() {
			out = append(out, o)
		}
	}
	return out
}

// CancelOrderRequest is POST /market/orders/update-status.
// At least one of Order or ClientOrderID is required. If both are set, the
// server prefers Order. Status is always "canceled".
type CancelOrderRequest struct {
	Order         int64  `json:"order,omitempty"`
	ClientOrderID string `json:"clientOrderId,omitempty"`
	Status        string `json:"status"`
}

// NewCancelOrderByID cancels by server order id.
func NewCancelOrderByID(id int64) CancelOrderRequest {
	return CancelOrderRequest{Order: id, Status: OrderCancelStatus}
}

// NewCancelOrderByClientOrderID cancels by user-supplied clientOrderId
// (open orders only, per docs).
func NewCancelOrderByClientOrderID(clientOrderID string) CancelOrderRequest {
	return CancelOrderRequest{ClientOrderID: clientOrderID, Status: OrderCancelStatus}
}

// Prepare trims identifiers, forces status=canceled, and checks that at least
// one of order or clientOrderId is present.
func (r CancelOrderRequest) Prepare() (CancelOrderRequest, error) {
	out := r
	out.ClientOrderID = strings.TrimSpace(out.ClientOrderID)
	out.Status = strings.ToLower(strings.TrimSpace(out.Status))
	if out.Status == "" {
		out.Status = OrderCancelStatus
	}
	if out.Status != OrderCancelStatus {
		return CancelOrderRequest{}, fmt.Errorf("types: cancel status must be %q", OrderCancelStatus)
	}
	if out.Order < 0 {
		return CancelOrderRequest{}, fmt.Errorf("types: cancel order id must be >= 1")
	}
	if err := validateClientOrderID(out.ClientOrderID); err != nil {
		return CancelOrderRequest{}, err
	}
	if out.Order == 0 && out.ClientOrderID == "" {
		return CancelOrderRequest{}, fmt.Errorf("types: cancel requires order id or clientOrderId")
	}
	return out, nil
}

// CancelOrderResponse is POST /market/orders/update-status on success.
// A failed status transition is still HTTP 200 with status=failed; the shared
// error mapper returns that as an API error (updatedStatus/order are in RawBody).
type CancelOrderResponse struct {
	Envelope
	UpdatedStatus OrderStatus `json:"updatedStatus"`
	Order         *UserOrder  `json:"order,omitempty"`
}

// IsOpen reports whether the order can still be canceled (New, Active, Inactive).
func (s OrderStatus) IsOpen() bool {
	return s == OrderStatusNew || s == OrderStatusActive || s == OrderStatusInactive
}

// IsCanceled reports status Canceled.
func (s OrderStatus) IsCanceled() bool { return s == OrderStatusCanceled }
