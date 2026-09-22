package types

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Position list query `status` values (https://apidocs.nobitex.ir).
const (
	PositionListActive PositionListStatus = "active" // open / not-yet-settled (API default)
	PositionListPast   PositionListStatus = "past"   // closed, liquidated, or expired after settlement
)

// PositionListStatus is the GET /positions/list `status` query filter.
type PositionListStatus string

// Per-position lifecycle values (PascalCase on the wire).
const (
	PositionStatusOpen       PositionStatus = "Open"
	PositionStatusClosed     PositionStatus = "Closed"
	PositionStatusLiquidated PositionStatus = "Liquidated"
	PositionStatusExpired    PositionStatus = "Expired"
)

// PositionStatus is the current state of one margin position.
type PositionStatus string

// IsOpen reports whether the position is still live (status Open).
func (s PositionStatus) IsOpen() bool { return s == PositionStatusOpen }

// IsSettled reports Closed, Liquidated, or Expired (past / finished positions).
func (s PositionStatus) IsSettled() bool {
	return s == PositionStatusClosed || s == PositionStatusLiquidated || s == PositionStatusExpired
}

// Position direction: buy = long, sell = short.
const (
	PositionDirectionBuy  PositionDirection = "buy"
	PositionDirectionSell PositionDirection = "sell"
)

// PositionDirection is the opening side of a position (`buy` or `sell`).
type PositionDirection string

// Documented marginType values.
const (
	MarginTypeIsolated MarginType = "Isolated Margin"
	MarginTypeCross    MarginType = "Cross Margin"
)

// MarginType is Isolated vs Cross collateral. Isolated is the documented default.
type MarginType string

// Close-order execution on the wire (snake_case). Response bodies use PascalCase.
const (
	CloseExecutionLimit      CloseExecution = "limit"
	CloseExecutionMarket     CloseExecution = "market"
	CloseExecutionStopLimit  CloseExecution = "stop_limit"
	CloseExecutionStopMarket CloseExecution = "stop_market"
	// CloseExecutionOCO is a convenience value accepted by ClosePositionRequest.Prepare.
	// The documented OCO oneOf sends execution=limit and mode=oco.
	CloseExecutionOCO CloseExecution = "oco"
)

// CloseExecution is POST /positions/:id/close `execution`.
type CloseExecution string

// CloseOrderMode selects OCO vs a single close order. Documented values: default, oco.
type CloseOrderMode string

const (
	CloseOrderModeDefault CloseOrderMode = "default"
	CloseOrderModeOCO     CloseOrderMode = "oco"
)

// Response-side execution strings (PascalCase) on close orders.
const (
	CloseExecutionResponseLimit      = "Limit"
	CloseExecutionResponseMarket     = "Market"
	CloseExecutionResponseStopLimit  = "StopLimit"
	CloseExecutionResponseStopMarket = "StopMarket"
)

// Close order tradeType / order status / open-vs-close side as returned by
// POST /positions/:id/close (independent of margin-order-place types).
const (
	CloseTradeTypeMargin     = "Margin"
	CloseOrderStatusNew      = "New"
	CloseOrderStatusActive   = "Active"
	CloseOrderStatusDone     = "Done"
	CloseOrderStatusCanceled = "Canceled"
	CloseOrderStatusInactive = "Inactive"
	CloseOrderSideOpen       = "open"
	CloseOrderSideClose      = "close"
)

var closeClientOrderIDRe = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

const maxCloseClientOrderIDLen = 32

// PositionListQuery is GET /positions/list query filters.
// Empty fields are omitted so the API default applies (status=active, pageSize=50).
type PositionListQuery struct {
	SrcCurrency string             // e.g. btc; omit for every source
	DstCurrency string             // e.g. rls; omit for every destination
	Status      PositionListStatus // active | past
	Page        int                // >= 1
	PageSize    int                // >= 1; documented default 50
}

// Normalize trims/lowercases currencies and list status. It does not default
// Status; leaving it empty lets the API use `active`.
func (q PositionListQuery) Normalize() (PositionListQuery, error) {
	out := q
	out.SrcCurrency = strings.ToLower(strings.TrimSpace(out.SrcCurrency))
	out.DstCurrency = strings.ToLower(strings.TrimSpace(out.DstCurrency))
	out.Status = PositionListStatus(strings.ToLower(strings.TrimSpace(string(out.Status))))

	if out.Status != "" && out.Status != PositionListActive && out.Status != PositionListPast {
		return PositionListQuery{}, fmt.Errorf("types: position list status must be %q or %q", PositionListActive, PositionListPast)
	}
	if out.Page < 0 {
		return PositionListQuery{}, fmt.Errorf("types: position list page must be >= 1")
	}
	if out.PageSize < 0 {
		return PositionListQuery{}, fmt.Errorf("types: position list pageSize must be >= 1")
	}
	return out, nil
}

// Position is one item from GET /positions/list.
//
// ID and Status are the fields Tradex uses to track and close a position.
// Active-only fields (delegatedAmount, liability, unrealizedPNL, …) are nil
// on settled (past) rows; past-only fields (PNL, PNLPercent, exitPrice) are
// typically nil while the position is Open.
type Position struct {
	ID               int64             `json:"id"`
	CreatedAt        time.Time         `json:"createdAt"`
	SrcCurrency      string            `json:"srcCurrency"`
	DstCurrency      string            `json:"dstCurrency"`
	Side             PositionDirection `json:"side"`
	Status           PositionStatus    `json:"status"`
	MarginType       MarginType        `json:"marginType"`
	Collateral       Money             `json:"collateral"`
	Leverage         Money             `json:"leverage"`
	OpenedAt         *time.Time        `json:"openedAt"`
	ClosedAt         *time.Time        `json:"closedAt"`
	LiquidationPrice *Money            `json:"liquidationPrice"`
	EntryPrice       *Money            `json:"entryPrice"`
	ExitPrice        *Money            `json:"exitPrice"`

	// Active-position fields (null on settled rows).
	DelegatedAmount      *Money  `json:"delegatedAmount"`
	Liability            *Money  `json:"liability"`
	TotalAsset           *Money  `json:"totalAsset"`
	MarginRatio          *Money  `json:"marginRatio"`
	LiabilityInOrder     *Money  `json:"liabilityInOrder"`
	AssetInOrder         *Money  `json:"assetInOrder"`
	UnrealizedPNL        *Money  `json:"unrealizedPNL"`
	UnrealizedPNLPercent *Money  `json:"unrealizedPNLPercent"`
	ExpirationDate       *string `json:"expirationDate"` // YYYY-MM-DD
	ExtensionFee         *Money  `json:"extensionFee"`
	MarkPrice            *Money  `json:"markPrice"`

	// Settled-position fields (null while Open).
	PNL        *Money `json:"PNL"`
	PNLPercent *Money `json:"PNLPercent"`
}

// IsOpen reports whether this row is an open (not-yet-settled) position.
func (p Position) IsOpen() bool { return p.Status.IsOpen() }

// IsSettled reports Closed, Liquidated, or Expired.
func (p Position) IsSettled() bool { return p.Status.IsSettled() }

// MoneyValue returns *m or "" when nil. Used for Tradex-facing optional fields.
func MoneyValue(m *Money) Money {
	if m == nil {
		return ""
	}
	return *m
}

// PositionListResponse is GET /positions/list.
type PositionListResponse struct {
	Envelope
	Positions []Position `json:"positions"`
	HasNext   bool       `json:"hasNext"`
}

// Position returns the row with the given id, if present.
func (r *PositionListResponse) Position(id int64) (Position, bool) {
	if r == nil {
		return Position{}, false
	}
	for _, p := range r.Positions {
		if p.ID == id {
			return p, true
		}
	}
	return Position{}, false
}

// ClosePositionRequest is POST /positions/:positionId/close.
//
// Closing always places the opposite-side margin order (sell position → buy
// close, buy position → sell close). Documented oneOf variants:
//   - limit:       execution=limit, amount + price required
//   - market:      execution=market, amount required
//   - stop_limit:  execution=stop_limit, amount + price + stopPrice required
//   - stop_market: execution=stop_market, amount + stopPrice required
//   - oco:         execution=limit, mode=oco, amount + price + stopPrice + stopLimitPrice required
//
// Prepare lowercases execution/mode and, for OCO, rewrites execution=oco into
// the documented execution=limit + mode=oco pair.
type ClosePositionRequest struct {
	Execution      CloseExecution `json:"execution"`
	Mode           CloseOrderMode `json:"mode,omitempty"`
	Amount         Money          `json:"amount"`
	Price          Money          `json:"price,omitempty"`
	StopPrice      Money          `json:"stopPrice,omitempty"`
	StopLimitPrice Money          `json:"stopLimitPrice,omitempty"`
	ClientOrderID  string         `json:"clientOrderId,omitempty"`
}

// NewCloseLimitOrder builds a limit close (priced opposite-side order).
func NewCloseLimitOrder(amount, price Money) ClosePositionRequest {
	return ClosePositionRequest{Execution: CloseExecutionLimit, Amount: amount, Price: price}
}

// NewCloseMarketOrder builds a market close.
func NewCloseMarketOrder(amount Money) ClosePositionRequest {
	return ClosePositionRequest{Execution: CloseExecutionMarket, Amount: amount}
}

// NewCloseStopLimitOrder builds a stop-limit close (stopPrice then limit at price).
func NewCloseStopLimitOrder(amount, price, stopPrice Money) ClosePositionRequest {
	return ClosePositionRequest{
		Execution: CloseExecutionStopLimit,
		Amount:    amount,
		Price:     price,
		StopPrice: stopPrice,
	}
}

// NewCloseStopMarketOrder builds a stop-market close.
func NewCloseStopMarketOrder(amount, stopPrice Money) ClosePositionRequest {
	return ClosePositionRequest{
		Execution: CloseExecutionStopMarket,
		Amount:    amount,
		StopPrice: stopPrice,
	}
}

// NewCloseOCOOrder builds an OCO close (limit + stop-limit pair).
// On the wire this is execution=limit and mode=oco.
func NewCloseOCOOrder(amount, price, stopPrice, stopLimitPrice Money) ClosePositionRequest {
	return ClosePositionRequest{
		Execution:      CloseExecutionLimit,
		Mode:           CloseOrderModeOCO,
		Amount:         amount,
		Price:          price,
		StopPrice:      stopPrice,
		StopLimitPrice: stopLimitPrice,
	}
}

// IsOCO reports whether this request is (or will be sent as) an OCO pair.
func (r ClosePositionRequest) IsOCO() bool {
	mode := CloseOrderMode(strings.ToLower(strings.TrimSpace(string(r.Mode))))
	exec := CloseExecution(strings.ToLower(strings.TrimSpace(string(r.Execution))))
	return mode == CloseOrderModeOCO || exec == CloseExecutionOCO
}

// Prepare returns a copy ready to marshal: trimmed/lowercased fields, OCO
// rewritten to the documented wire shape, and required fields checked.
func (r ClosePositionRequest) Prepare() (ClosePositionRequest, error) {
	out := r
	out.Execution = CloseExecution(strings.ToLower(strings.TrimSpace(string(out.Execution))))
	out.Mode = CloseOrderMode(strings.ToLower(strings.TrimSpace(string(out.Mode))))
	out.ClientOrderID = strings.TrimSpace(out.ClientOrderID)
	out.Amount = Money(strings.TrimSpace(string(out.Amount)))
	out.Price = Money(strings.TrimSpace(string(out.Price)))
	out.StopPrice = Money(strings.TrimSpace(string(out.StopPrice)))
	out.StopLimitPrice = Money(strings.TrimSpace(string(out.StopLimitPrice)))

	if out.Amount == "" {
		return ClosePositionRequest{}, fmt.Errorf("types: close position amount is required")
	}
	if err := validateCloseClientOrderID(out.ClientOrderID); err != nil {
		return ClosePositionRequest{}, err
	}

	isOCO := out.Mode == CloseOrderModeOCO || out.Execution == CloseExecutionOCO
	if isOCO {
		if out.Execution != "" && out.Execution != CloseExecutionLimit && out.Execution != CloseExecutionOCO {
			return ClosePositionRequest{}, fmt.Errorf("types: OCO close requires execution %q (got %q)", CloseExecutionLimit, out.Execution)
		}
		out.Execution = CloseExecutionLimit
		out.Mode = CloseOrderModeOCO
		if out.Price == "" || out.StopPrice == "" || out.StopLimitPrice == "" {
			return ClosePositionRequest{}, fmt.Errorf("types: OCO close requires price, stopPrice, and stopLimitPrice")
		}
		return out, nil
	}

	out.Mode = ""
	if out.Execution == "" {
		out.Execution = CloseExecutionLimit
	}
	switch out.Execution {
	case CloseExecutionLimit:
		if out.Price == "" {
			return ClosePositionRequest{}, fmt.Errorf("types: limit close requires price")
		}
	case CloseExecutionMarket:
		// price is not in the documented market oneOf; leave it only if the caller set it.
	case CloseExecutionStopMarket:
		if out.StopPrice == "" {
			return ClosePositionRequest{}, fmt.Errorf("types: stop_market close requires stopPrice")
		}
	case CloseExecutionStopLimit:
		if out.Price == "" || out.StopPrice == "" {
			return ClosePositionRequest{}, fmt.Errorf("types: stop_limit close requires price and stopPrice")
		}
	default:
		return ClosePositionRequest{}, fmt.Errorf("types: unsupported close execution %q (want limit, market, stop_limit, stop_market, or oco)", out.Execution)
	}
	return out, nil
}

func validateCloseClientOrderID(id string) error {
	if id == "" {
		return nil
	}
	if len(id) > maxCloseClientOrderIDLen {
		return fmt.Errorf("types: clientOrderId exceeds %d characters", maxCloseClientOrderIDLen)
	}
	if !closeClientOrderIDRe.MatchString(id) {
		return fmt.Errorf("types: clientOrderId must match ^[A-Za-z0-9-]+$")
	}
	return nil
}

// CloseOrder is one opposite-side order returned by POST /positions/:id/close.
type CloseOrder struct {
	ID              int64     `json:"id"`
	Type            string    `json:"type"` // buy | sell (opposite of the position)
	Execution       string    `json:"execution"`
	TradeType       string    `json:"tradeType"`
	SrcCurrency     string    `json:"srcCurrency"`
	DstCurrency     string    `json:"dstCurrency"`
	Price           Money     `json:"price"`
	Amount          Money     `json:"amount"`
	TotalPrice      Money     `json:"totalPrice"`
	TotalOrderPrice Money     `json:"totalOrderPrice"`
	MatchedAmount   Money     `json:"matchedAmount"`
	UnmatchedAmount Money     `json:"unmatchedAmount"`
	ClientOrderID   *string   `json:"clientOrderId"`
	Param1          string    `json:"param1,omitempty"`
	PairID          *int64    `json:"pairId"`
	Leverage        Money     `json:"leverage,omitempty"`
	Side            string    `json:"side,omitempty"` // open | close
	Status          string    `json:"status"`
	Partial         bool      `json:"partial"`
	Fee             Money     `json:"fee"`
	CreatedAt       time.Time `json:"created_at"`
	AveragePrice    Money     `json:"averagePrice"`
}

// ClientOrderIDValue returns the clientOrderId string, or "" when null/absent.
func (o CloseOrder) ClientOrderIDValue() string {
	if o.ClientOrderID == nil {
		return ""
	}
	return *o.ClientOrderID
}

// ClosePositionResponse is the success envelope: a single close order, or two OCO legs.
type ClosePositionResponse struct {
	Envelope
	Order  *CloseOrder  `json:"order,omitempty"`
	Orders []CloseOrder `json:"orders,omitempty"`
}

// CloseOrders returns the order(s) that were accepted: the single `order`
// object, or both OCO legs from `orders`.
func (r *ClosePositionResponse) CloseOrders() []CloseOrder {
	if r == nil {
		return nil
	}
	if len(r.Orders) > 0 {
		return r.Orders
	}
	if r.Order != nil {
		return []CloseOrder{*r.Order}
	}
	return nil
}

// OrderIDs returns every server order id in the response (one for a single
// close, two for OCO). Empty if the body had no orders.
func (r *ClosePositionResponse) OrderIDs() []int64 {
	orders := r.CloseOrders()
	ids := make([]int64, 0, len(orders))
	for _, o := range orders {
		if o.ID != 0 {
			ids = append(ids, o.ID)
		}
	}
	return ids
}
