package types

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Execution is the request-side order execution type (snake_case on the wire).
// Response bodies use PascalCase values on MarginOrder.Execution (Limit, Market, …).
type Execution string

const (
	// ExecutionLimit is a priced order (OpenAPI default for this endpoint).
	ExecutionLimit Execution = "limit"
	// ExecutionMarket fills at the best available price; price is not required.
	ExecutionMarket Execution = "market"
	// ExecutionStopLimit activates a limit order when stopPrice is reached.
	ExecutionStopLimit Execution = "stop_limit"
	// ExecutionStopMarket activates a market order when stopPrice is reached.
	ExecutionStopMarket Execution = "stop_market"
	// ExecutionOCO is a convenience value accepted by MarginOrderRequest.Prepare.
	// The documented OCO oneOf sends execution=limit and mode=oco.
	ExecutionOCO Execution = "oco"
)

// OrderMode selects OCO vs a single order. Documented values: default, oco.
type OrderMode string

const (
	OrderModeDefault OrderMode = "default"
	OrderModeOCO     OrderMode = "oco"
)

// OrderSide is buy (open long) or sell (open short) for a margin order.
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderStatus is the documented lifecycle of a placed order.
type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "New"
	OrderStatusActive   OrderStatus = "Active"
	OrderStatusDone     OrderStatus = "Done"
	OrderStatusCanceled OrderStatus = "Canceled"
	OrderStatusInactive OrderStatus = "Inactive"
)

// TradeType is the venue returned on an order (margin orders are "Margin").
type TradeType string

const (
	TradeTypeSpot   TradeType = "Spot"
	TradeTypeMargin TradeType = "Margin"
	TradeTypeCredit TradeType = "Credit"
	TradeTypeDebit  TradeType = "Debit"
)

// PositionSide reports whether the order opens or closes a margin position.
type PositionSide string

const (
	PositionSideOpen  PositionSide = "open"
	PositionSideClose PositionSide = "close"
)

// Response-side execution strings (PascalCase), as returned by POST /margin/orders/add.
const (
	ExecutionResponseLimit      = "Limit"
	ExecutionResponseMarket     = "Market"
	ExecutionResponseStopLimit  = "StopLimit"
	ExecutionResponseStopMarket = "StopMarket"
)

// clientOrderIDRe is the documented clientOrderId pattern (max 32 chars).
var clientOrderIDRe = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

const maxClientOrderIDLen = 32

// MarginOrderRequest is POST /margin/orders/add.
//
// Documented oneOf variants (https://apidocs.nobitex.ir):
//   - limit:       execution=limit, price required
//   - market:      execution=market
//   - stop_limit:  execution=stop_limit, price + stopPrice required
//   - stop_market: execution=stop_market, stopPrice required
//   - oco:         execution=limit, mode=oco, price + stopPrice + stopLimitPrice required
//
// Prepare lowercases currencies/execution and, for OCO, rewrites execution=oco
// into the documented execution=limit + mode=oco pair.
type MarginOrderRequest struct {
	Execution      Execution `json:"execution"`
	Mode           OrderMode `json:"mode,omitempty"`
	SrcCurrency    string    `json:"srcCurrency"`
	DstCurrency    string    `json:"dstCurrency"`
	Type           OrderSide `json:"type"`
	Leverage       Money     `json:"leverage,omitempty"`
	Amount         Money     `json:"amount"`
	Price          Money     `json:"price,omitempty"`
	StopPrice      Money     `json:"stopPrice,omitempty"`
	StopLimitPrice Money     `json:"stopLimitPrice,omitempty"`
	ClientOrderID  string    `json:"clientOrderId,omitempty"`
}

// NewMarginLimitOrder builds a limit request. Set Leverage / ClientOrderID on the result.
func NewMarginLimitOrder(side OrderSide, src, dst string, amount, price Money) MarginOrderRequest {
	return MarginOrderRequest{
		Execution:   ExecutionLimit,
		Type:        side,
		SrcCurrency: src,
		DstCurrency: dst,
		Amount:      amount,
		Price:       price,
	}
}

// NewMarginMarketOrder builds a market request.
func NewMarginMarketOrder(side OrderSide, src, dst string, amount Money) MarginOrderRequest {
	return MarginOrderRequest{
		Execution:   ExecutionMarket,
		Type:        side,
		SrcCurrency: src,
		DstCurrency: dst,
		Amount:      amount,
	}
}

// NewMarginStopLimitOrder builds a stop-limit request (stopPrice then limit at price).
func NewMarginStopLimitOrder(side OrderSide, src, dst string, amount, price, stopPrice Money) MarginOrderRequest {
	return MarginOrderRequest{
		Execution:   ExecutionStopLimit,
		Type:        side,
		SrcCurrency: src,
		DstCurrency: dst,
		Amount:      amount,
		Price:       price,
		StopPrice:   stopPrice,
	}
}

// NewMarginStopMarketOrder builds a stop-market request.
func NewMarginStopMarketOrder(side OrderSide, src, dst string, amount, stopPrice Money) MarginOrderRequest {
	return MarginOrderRequest{
		Execution:   ExecutionStopMarket,
		Type:        side,
		SrcCurrency: src,
		DstCurrency: dst,
		Amount:      amount,
		StopPrice:   stopPrice,
	}
}

// NewMarginOCOOrder builds an OCO request (limit + stop-limit pair).
// On the wire this is execution=limit and mode=oco.
func NewMarginOCOOrder(side OrderSide, src, dst string, amount, price, stopPrice, stopLimitPrice Money) MarginOrderRequest {
	return MarginOrderRequest{
		Execution:      ExecutionLimit,
		Mode:           OrderModeOCO,
		Type:           side,
		SrcCurrency:    src,
		DstCurrency:    dst,
		Amount:         amount,
		Price:          price,
		StopPrice:      stopPrice,
		StopLimitPrice: stopLimitPrice,
	}
}

// IsOCO reports whether this request is (or will be sent as) an OCO pair.
func (r MarginOrderRequest) IsOCO() bool {
	mode := OrderMode(strings.ToLower(strings.TrimSpace(string(r.Mode))))
	exec := Execution(strings.ToLower(strings.TrimSpace(string(r.Execution))))
	return mode == OrderModeOCO || exec == ExecutionOCO
}

// Prepare returns a copy ready to marshal: trimmed/lowercased fields, OCO
// rewritten to the documented wire shape, and required fields checked.
func (r MarginOrderRequest) Prepare() (MarginOrderRequest, error) {
	out := r
	out.SrcCurrency = strings.ToLower(strings.TrimSpace(out.SrcCurrency))
	out.DstCurrency = strings.ToLower(strings.TrimSpace(out.DstCurrency))
	out.Execution = Execution(strings.ToLower(strings.TrimSpace(string(out.Execution))))
	out.Mode = OrderMode(strings.ToLower(strings.TrimSpace(string(out.Mode))))
	out.Type = OrderSide(strings.ToLower(strings.TrimSpace(string(out.Type))))
	out.ClientOrderID = strings.TrimSpace(out.ClientOrderID)
	out.Leverage = Money(strings.TrimSpace(string(out.Leverage)))
	out.Amount = Money(strings.TrimSpace(string(out.Amount)))
	out.Price = Money(strings.TrimSpace(string(out.Price)))
	out.StopPrice = Money(strings.TrimSpace(string(out.StopPrice)))
	out.StopLimitPrice = Money(strings.TrimSpace(string(out.StopLimitPrice)))

	if out.SrcCurrency == "" {
		return MarginOrderRequest{}, fmt.Errorf("types: margin order srcCurrency is required")
	}
	if out.DstCurrency == "" {
		return MarginOrderRequest{}, fmt.Errorf("types: margin order dstCurrency is required")
	}
	if out.Amount == "" {
		return MarginOrderRequest{}, fmt.Errorf("types: margin order amount is required")
	}
	if out.Type != OrderSideBuy && out.Type != OrderSideSell {
		return MarginOrderRequest{}, fmt.Errorf("types: margin order type must be %q or %q", OrderSideBuy, OrderSideSell)
	}
	if err := validateClientOrderID(out.ClientOrderID); err != nil {
		return MarginOrderRequest{}, err
	}

	isOCO := out.Mode == OrderModeOCO || out.Execution == ExecutionOCO
	if isOCO {
		if out.Execution != "" && out.Execution != ExecutionLimit && out.Execution != ExecutionOCO {
			return MarginOrderRequest{}, fmt.Errorf("types: OCO orders require execution %q (got %q)", ExecutionLimit, out.Execution)
		}
		out.Execution = ExecutionLimit
		out.Mode = OrderModeOCO
		if out.Price == "" || out.StopPrice == "" || out.StopLimitPrice == "" {
			return MarginOrderRequest{}, fmt.Errorf("types: OCO orders require price, stopPrice, and stopLimitPrice")
		}
		return out, nil
	}

	out.Mode = ""
	switch out.Execution {
	case ExecutionLimit:
		if out.Price == "" {
			return MarginOrderRequest{}, fmt.Errorf("types: limit orders require price")
		}
	case ExecutionMarket:
		// price is not in the documented market oneOf; leave it only if the caller set it.
	case ExecutionStopMarket:
		if out.StopPrice == "" {
			return MarginOrderRequest{}, fmt.Errorf("types: stop_market orders require stopPrice")
		}
	case ExecutionStopLimit:
		if out.Price == "" || out.StopPrice == "" {
			return MarginOrderRequest{}, fmt.Errorf("types: stop_limit orders require price and stopPrice")
		}
	case "":
		return MarginOrderRequest{}, fmt.Errorf("types: margin order execution is required")
	default:
		return MarginOrderRequest{}, fmt.Errorf("types: unsupported margin execution %q (want limit, market, stop_limit, stop_market, or oco)", out.Execution)
	}
	return out, nil
}

func validateClientOrderID(id string) error {
	if id == "" {
		return nil
	}
	if len(id) > maxClientOrderIDLen {
		return fmt.Errorf("types: clientOrderId exceeds %d characters", maxClientOrderIDLen)
	}
	if !clientOrderIDRe.MatchString(id) {
		return fmt.Errorf("types: clientOrderId must match ^[A-Za-z0-9-]+$")
	}
	return nil
}

// MarginOrder is one placed order as returned by POST /margin/orders/add.
// ID and ClientOrderID are the identifiers used later to cancel or list.
// PairID links the two legs of an OCO pair when present.
type MarginOrder struct {
	ID              int64        `json:"id"`
	Type            OrderSide    `json:"type"`
	Execution       string       `json:"execution"`
	TradeType       TradeType    `json:"tradeType"`
	SrcCurrency     string       `json:"srcCurrency"`
	DstCurrency     string       `json:"dstCurrency"`
	Price           Money        `json:"price"`
	Amount          Money        `json:"amount"`
	TotalPrice      Money        `json:"totalPrice"`
	TotalOrderPrice Money        `json:"totalOrderPrice"`
	MatchedAmount   Money        `json:"matchedAmount"`
	UnmatchedAmount Money        `json:"unmatchedAmount"`
	ClientOrderID   *string      `json:"clientOrderId"`
	Param1          string       `json:"param1,omitempty"`
	PairID          *int64       `json:"pairId"`
	Leverage        Money        `json:"leverage,omitempty"`
	Side            PositionSide `json:"side,omitempty"`
	Status          OrderStatus  `json:"status"`
	Partial         bool         `json:"partial"`
	Fee             Money        `json:"fee"`
	CreatedAt       time.Time    `json:"created_at"`
	AveragePrice    Money        `json:"averagePrice"`
}

// ClientOrderIDValue returns the clientOrderId string, or "" when null/absent.
func (o MarginOrder) ClientOrderIDValue() string {
	if o.ClientOrderID == nil {
		return ""
	}
	return *o.ClientOrderID
}

// MarginOrderAddResponse is the success envelope: a single order, or two OCO legs.
type MarginOrderAddResponse struct {
	Envelope
	Order  *MarginOrder  `json:"order,omitempty"`
	Orders []MarginOrder `json:"orders,omitempty"`
}

// PlacedOrders returns the order(s) that were accepted: the single `order`
// object, or both OCO legs from `orders`. Identifiers on each item (ID,
// ClientOrderID, PairID) are what later cancel/list calls need.
func (r *MarginOrderAddResponse) PlacedOrders() []MarginOrder {
	if r == nil {
		return nil
	}
	if len(r.Orders) > 0 {
		return r.Orders
	}
	if r.Order != nil {
		return []MarginOrder{*r.Order}
	}
	return nil
}

// OrderIDs returns every server order id in the response (one for a single
// order, two for OCO). Empty if the body had no orders.
func (r *MarginOrderAddResponse) OrderIDs() []int64 {
	orders := r.PlacedOrders()
	ids := make([]int64, 0, len(orders))
	for _, o := range orders {
		if o.ID != 0 {
			ids = append(ids, o.ID)
		}
	}
	return ids
}
