package types

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PriceLevel is one asks/bids row: a documented [price, amount] string pair.
// See https://apidocs.nobitex.ir (GET /v3/orderbook/:symbol).
type PriceLevel struct {
	Price  Money
	Amount Money
}

// UnmarshalJSON accepts the documented JSON array ["price","amount"].
func (p *PriceLevel) UnmarshalJSON(data []byte) error {
	var pair []Money
	if err := json.Unmarshal(data, &pair); err != nil {
		return err
	}
	if len(pair) != 2 {
		return fmt.Errorf("types: price level must be [price, amount], got %d values", len(pair))
	}
	p.Price = pair[0]
	p.Amount = pair[1]
	return nil
}

// MarshalJSON emits [price, amount] as documented by Nobitex.
func (p PriceLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]Money{p.Price, p.Amount})
}

// OrderBook is a single-market GET /v3/orderbook/:symbol body.
// Asks are sell orders (typically lowest price first); bids are buy orders
// (typically highest price first). Monetary fields stay strings (Money).
type OrderBook struct {
	Envelope
	Symbol         string       `json:"-"`
	Asks           []PriceLevel `json:"asks"`
	Bids           []PriceLevel `json:"bids"`
	LastTradePrice Money        `json:"lastTradePrice"`
	LastUpdate     *int64       `json:"lastUpdate"` // unix milliseconds; docs allow null
}

// OrderBookAll is the consolidated GET /v3/orderbook/all body: status plus one
// object per market symbol (BTCIRT, USDTIRT, …).
type OrderBookAll struct {
	Envelope
	Books map[string]OrderBook
}

var orderBookReserved = map[string]struct{}{
	"status":  {},
	"code":    {},
	"message": {},
	"backOff": {},
	"limit":   {},
}

// UnmarshalJSON splits the all-markets object into Envelope + per-symbol books.
func (o *OrderBookAll) UnmarshalJSON(data []byte) error {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	o.Envelope = env
	o.Books = make(map[string]OrderBook, len(raw))
	for k, v := range raw {
		if _, skip := orderBookReserved[k]; skip {
			continue
		}
		var book OrderBook
		if err := json.Unmarshal(v, &book); err != nil {
			return fmt.Errorf("types: orderbook %s: %w", k, err)
		}
		book.Symbol = k
		o.Books[k] = book
	}
	return nil
}

// Book returns the depth for symbol (exact key, then uppercased).
func (o *OrderBookAll) Book(symbol string) (OrderBook, bool) {
	if o == nil || o.Books == nil {
		return OrderBook{}, false
	}
	if b, ok := o.Books[symbol]; ok {
		return b, true
	}
	alt := strings.ToUpper(strings.TrimSpace(symbol))
	if alt != symbol {
		if b, ok := o.Books[alt]; ok {
			return b, true
		}
	}
	return OrderBook{}, false
}
