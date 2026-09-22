package types

import (
	"fmt"
	"strings"
	"time"
)

// SpotOrderRequest is POST /market/orders/add (spot venue).
//
// Documented executions match margin: limit, market, stop_limit, stop_market,
// and oco (sent as execution=limit + mode=oco). This is a spot helper — it
// does not list or cancel orders (those belong to a separate P0 ticket).
type SpotOrderRequest struct {
	Execution      Execution `json:"execution"`
	Mode           OrderMode `json:"mode,omitempty"`
	Type           OrderSide `json:"type"`
	SrcCurrency    string    `json:"srcCurrency"`
	DstCurrency    string    `json:"dstCurrency"`
	Amount         Money     `json:"amount"`
	Price          Money     `json:"price,omitempty"`
	StopPrice      Money     `json:"stopPrice,omitempty"`
	StopLimitPrice Money     `json:"stopLimitPrice,omitempty"`
	ClientOrderID  string    `json:"clientOrderId,omitempty"`
	Pro            bool      `json:"pro,omitempty"`
}

// NewSpotLimitOrder builds a spot limit request.
func NewSpotLimitOrder(side OrderSide, src, dst string, amount, price Money) SpotOrderRequest {
	return SpotOrderRequest{
		Execution:   ExecutionLimit,
		Type:        side,
		SrcCurrency: src,
		DstCurrency: dst,
		Amount:      amount,
		Price:       price,
	}
}

// NewSpotMarketOrder builds a spot market request.
func NewSpotMarketOrder(side OrderSide, src, dst string, amount Money) SpotOrderRequest {
	return SpotOrderRequest{
		Execution:   ExecutionMarket,
		Type:        side,
		SrcCurrency: src,
		DstCurrency: dst,
		Amount:      amount,
	}
}

// NewSpotStopLimitOrder builds a spot stop-limit request.
func NewSpotStopLimitOrder(side OrderSide, src, dst string, amount, price, stopPrice Money) SpotOrderRequest {
	return SpotOrderRequest{
		Execution:   ExecutionStopLimit,
		Type:        side,
		SrcCurrency: src,
		DstCurrency: dst,
		Amount:      amount,
		Price:       price,
		StopPrice:   stopPrice,
	}
}

// NewSpotStopMarketOrder builds a spot stop-market request.
func NewSpotStopMarketOrder(side OrderSide, src, dst string, amount, stopPrice Money) SpotOrderRequest {
	return SpotOrderRequest{
		Execution:   ExecutionStopMarket,
		Type:        side,
		SrcCurrency: src,
		DstCurrency: dst,
		Amount:      amount,
		StopPrice:   stopPrice,
	}
}

// NewSpotOCOOrder builds a spot OCO pair (limit + stop-limit).
func NewSpotOCOOrder(side OrderSide, src, dst string, amount, price, stopPrice, stopLimitPrice Money) SpotOrderRequest {
	return SpotOrderRequest{
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
func (r SpotOrderRequest) IsOCO() bool {
	mode := OrderMode(strings.ToLower(strings.TrimSpace(string(r.Mode))))
	exec := Execution(strings.ToLower(strings.TrimSpace(string(r.Execution))))
	return mode == OrderModeOCO || exec == ExecutionOCO
}

// Prepare returns a copy ready to marshal. OCO is rewritten to execution=limit
// + mode=oco as documented.
func (r SpotOrderRequest) Prepare() (SpotOrderRequest, error) {
	out := r
	out.SrcCurrency = strings.ToLower(strings.TrimSpace(out.SrcCurrency))
	out.DstCurrency = strings.ToLower(strings.TrimSpace(out.DstCurrency))
	out.Execution = Execution(strings.ToLower(strings.TrimSpace(string(out.Execution))))
	out.Mode = OrderMode(strings.ToLower(strings.TrimSpace(string(out.Mode))))
	out.Type = OrderSide(strings.ToLower(strings.TrimSpace(string(out.Type))))
	out.ClientOrderID = strings.TrimSpace(out.ClientOrderID)
	out.Amount = Money(strings.TrimSpace(string(out.Amount)))
	out.Price = Money(strings.TrimSpace(string(out.Price)))
	out.StopPrice = Money(strings.TrimSpace(string(out.StopPrice)))
	out.StopLimitPrice = Money(strings.TrimSpace(string(out.StopLimitPrice)))

	if out.SrcCurrency == "" {
		return SpotOrderRequest{}, fmt.Errorf("types: spot order srcCurrency is required")
	}
	if out.DstCurrency == "" {
		return SpotOrderRequest{}, fmt.Errorf("types: spot order dstCurrency is required")
	}
	if out.Amount == "" {
		return SpotOrderRequest{}, fmt.Errorf("types: spot order amount is required")
	}
	if out.Type != OrderSideBuy && out.Type != OrderSideSell {
		return SpotOrderRequest{}, fmt.Errorf("types: spot order type must be %q or %q", OrderSideBuy, OrderSideSell)
	}
	if err := validateClientOrderID(out.ClientOrderID); err != nil {
		return SpotOrderRequest{}, err
	}

	isOCO := out.Mode == OrderModeOCO || out.Execution == ExecutionOCO
	if isOCO {
		if out.Execution != "" && out.Execution != ExecutionLimit && out.Execution != ExecutionOCO {
			return SpotOrderRequest{}, fmt.Errorf("types: OCO orders require execution %q (got %q)", ExecutionLimit, out.Execution)
		}
		out.Execution = ExecutionLimit
		out.Mode = OrderModeOCO
		if out.Price == "" || out.StopPrice == "" || out.StopLimitPrice == "" {
			return SpotOrderRequest{}, fmt.Errorf("types: OCO orders require price, stopPrice, and stopLimitPrice")
		}
		return out, nil
	}

	out.Mode = ""
	switch out.Execution {
	case ExecutionLimit:
		if out.Price == "" {
			return SpotOrderRequest{}, fmt.Errorf("types: limit orders require price")
		}
	case ExecutionMarket:
		// price is optional on market and used only as a bound when set.
	case ExecutionStopMarket:
		if out.StopPrice == "" {
			return SpotOrderRequest{}, fmt.Errorf("types: stop_market orders require stopPrice")
		}
	case ExecutionStopLimit:
		if out.Price == "" || out.StopPrice == "" {
			return SpotOrderRequest{}, fmt.Errorf("types: stop_limit orders require price and stopPrice")
		}
	case "":
		return SpotOrderRequest{}, fmt.Errorf("types: spot order execution is required")
	default:
		return SpotOrderRequest{}, fmt.Errorf("types: unsupported spot execution %q (want limit, market, stop_limit, stop_market, or oco)", out.Execution)
	}
	return out, nil
}

// SpotOrder is one spot order as returned by add / status.
type SpotOrder struct {
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
	ClientOrderID   *string     `json:"clientOrderId"`
	Param1          string      `json:"param1,omitempty"`
	PairID          *int64      `json:"pairId"`
	Status          OrderStatus `json:"status"`
	Partial         bool        `json:"partial"`
	Fee             Money       `json:"fee"`
	CreatedAt       time.Time   `json:"created_at"`
	AveragePrice    Money       `json:"averagePrice"`
}

// ClientOrderIDValue returns the clientOrderId string, or "" when null/absent.
func (o SpotOrder) ClientOrderIDValue() string {
	if o.ClientOrderID == nil {
		return ""
	}
	return *o.ClientOrderID
}

// SpotOrderAddResponse is POST /market/orders/add: a single order, or two OCO legs.
type SpotOrderAddResponse struct {
	Envelope
	Order  *SpotOrder  `json:"order,omitempty"`
	Orders []SpotOrder `json:"orders,omitempty"`
}

// PlacedOrders returns the accepted order or both OCO legs.
func (r *SpotOrderAddResponse) PlacedOrders() []SpotOrder {
	if r == nil {
		return nil
	}
	if len(r.Orders) > 0 {
		return r.Orders
	}
	if r.Order != nil {
		return []SpotOrder{*r.Order}
	}
	return nil
}

// OrderIDs returns every server order id in the response.
func (r *SpotOrderAddResponse) OrderIDs() []int64 {
	orders := r.PlacedOrders()
	ids := make([]int64, 0, len(orders))
	for _, o := range orders {
		if o.ID != 0 {
			ids = append(ids, o.ID)
		}
	}
	return ids
}

// SpotOrderStatusRequest is POST /market/orders/status.
// At least one of ID / ClientOrderID is required; the server prefers ID.
type SpotOrderStatusRequest struct {
	ID            int64  `json:"id,omitempty"`
	ClientOrderID string `json:"clientOrderId,omitempty"`
}

// Prepare trims clientOrderId and checks that an identifier is present.
func (r SpotOrderStatusRequest) Prepare() (SpotOrderStatusRequest, error) {
	out := r
	out.ClientOrderID = strings.TrimSpace(out.ClientOrderID)
	if out.ID <= 0 && out.ClientOrderID == "" {
		return SpotOrderStatusRequest{}, fmt.Errorf("types: order status requires id or clientOrderId")
	}
	if out.ClientOrderID != "" {
		if err := validateClientOrderID(out.ClientOrderID); err != nil {
			return SpotOrderStatusRequest{}, err
		}
	}
	return out, nil
}

// SpotOrderStatusResponse is POST /market/orders/status.
type SpotOrderStatusResponse struct {
	Envelope
	Order *SpotOrder `json:"order"`
}

// UserTrade is one row of GET /market/trades/list (spot fills, last 180 days).
type UserTrade struct {
	ID          int64     `json:"id"`
	OrderID     *int64    `json:"orderId"`
	SrcCurrency string    `json:"srcCurrency"`
	DstCurrency string    `json:"dstCurrency"`
	Market      string    `json:"market"`
	Timestamp   time.Time `json:"timestamp"`
	Type        OrderSide `json:"type"`
	Price       Money     `json:"price"`
	Amount      Money     `json:"amount"`
	Total       Money     `json:"total"`
	Fee         Money     `json:"fee"`
}

// UserTradesQuery is GET /market/trades/list.
// srcCurrency and dstCurrency must both be set or both empty.
type UserTradesQuery struct {
	SrcCurrency string
	DstCurrency string
	FromID      int64
	TradeType   OrderSide // buy | sell
	TradeOrder  string    // asc | desc
	Page        int
	PageSize    int
}

// Normalize trims/lowercases filters and checks the src/dst pairing rule.
func (q UserTradesQuery) Normalize() (UserTradesQuery, error) {
	out := q
	out.SrcCurrency = strings.ToLower(strings.TrimSpace(out.SrcCurrency))
	out.DstCurrency = strings.ToLower(strings.TrimSpace(out.DstCurrency))
	out.TradeType = OrderSide(strings.ToLower(strings.TrimSpace(string(out.TradeType))))
	out.TradeOrder = strings.ToLower(strings.TrimSpace(out.TradeOrder))
	if (out.SrcCurrency == "") != (out.DstCurrency == "") {
		return UserTradesQuery{}, fmt.Errorf("types: user trades srcCurrency and dstCurrency must both be set or both empty")
	}
	if out.TradeType != "" && out.TradeType != OrderSideBuy && out.TradeType != OrderSideSell {
		return UserTradesQuery{}, fmt.Errorf("types: user trades tradeType must be buy or sell")
	}
	if out.TradeOrder != "" && out.TradeOrder != "asc" && out.TradeOrder != "desc" {
		return UserTradesQuery{}, fmt.Errorf("types: user trades tradeOrder must be asc or desc")
	}
	if out.FromID < 0 || out.Page < 0 || out.PageSize < 0 {
		return UserTradesQuery{}, fmt.Errorf("types: user trades fromId/page/pageSize must be >= 0")
	}
	return out, nil
}

// UserTradesResponse is GET /market/trades/list.
type UserTradesResponse struct {
	Envelope
	Trades  []UserTrade `json:"trades"`
	HasNext bool        `json:"hasNext"`
}
