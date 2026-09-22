package types

import (
	"fmt"
	"strings"
)

// MarketStat is one row of GET /market/stats (keyed as "btc-rls", "btc-usdt", …).
// Monetary fields stay strings. Docs: https://apidocs.nobitex.ir
type MarketStat struct {
	IsClosed       bool    `json:"isClosed"`
	IsClosedReason *string `json:"isClosedReason"`
	BestSell       *Money  `json:"bestSell"`
	BestBuy        *Money  `json:"bestBuy"`
	VolumeSrc      *Money  `json:"volumeSrc"`
	VolumeDst      *Money  `json:"volumeDst"`
	Latest         *Money  `json:"latest"`
	Mark           *Money  `json:"mark"`
	DayLow         *Money  `json:"dayLow"`
	DayHigh        *Money  `json:"dayHigh"`
	DayOpen        *Money  `json:"dayOpen"`
	DayClose       *Money  `json:"dayClose"`
	DayChange      *string `json:"dayChange"`
}

// MarketStatsQuery is GET /market/stats. Both currencies are optional; omit
// both for every market. Comma-separated lists are allowed (btc,eth).
type MarketStatsQuery struct {
	SrcCurrency string
	DstCurrency string
}

// Normalize trims and lowercases currency filters. Empty stays empty.
func (q MarketStatsQuery) Normalize() MarketStatsQuery {
	return MarketStatsQuery{
		SrcCurrency: normalizeCurrencyList(q.SrcCurrency),
		DstCurrency: normalizeCurrencyList(q.DstCurrency),
	}
}

func normalizeCurrencyList(s string) string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ",")
}

// MarketStatsResponse is GET /market/stats. The deprecated `global` Binance
// block is ignored.
type MarketStatsResponse struct {
	Envelope
	Stats map[string]MarketStat `json:"stats"`
}

// Stat looks up a market. Accepts stats keys (btc-rls) or order-book symbols
// (BTCIRT, BTCUSDT).
func (r *MarketStatsResponse) Stat(symbol string) (MarketStat, bool) {
	if r == nil || r.Stats == nil {
		return MarketStat{}, false
	}
	if s, ok := r.Stats[symbol]; ok {
		return s, true
	}
	key := StatsKey(symbol)
	if s, ok := r.Stats[key]; ok {
		return s, true
	}
	if key != symbol {
		if s, ok := r.Stats[strings.ToLower(symbol)]; ok {
			return s, true
		}
	}
	return MarketStat{}, false
}

// StatsKey maps an order-book symbol (BTCIRT, BTCUSDT) to the stats object
// key (btc-rls, btc-usdt). Already-hyphenated keys are lowercased.
func StatsKey(symbol string) string {
	s := strings.ToLower(strings.TrimSpace(symbol))
	s = strings.ReplaceAll(s, "_", "-")
	if s == "" {
		return ""
	}
	if strings.Contains(s, "-") {
		return s
	}
	if strings.HasSuffix(s, "irt") {
		return strings.TrimSuffix(s, "irt") + "-rls"
	}
	if strings.HasSuffix(s, "usdt") && s != "usdt" {
		return strings.TrimSuffix(s, "usdt") + "-usdt"
	}
	return s
}

// MarketTrade is one public recent trade from GET /v2/trades/:symbol.
type MarketTrade struct {
	Time   int64     `json:"time"` // unix milliseconds
	Price  Money     `json:"price"`
	Volume Money     `json:"volume"`
	Type   OrderSide `json:"type"` // buy | sell
}

// MarketTradesResponse is GET /v2/trades/:symbol (at most 20 trades).
type MarketTradesResponse struct {
	Envelope
	Symbol string        `json:"-"`
	Trades []MarketTrade `json:"trades"`
}

// MarketDepth is GET /v2/depth/:symbol (depth-chart shape; same [price,amount]
// levels as the order book). lastUpdate is a unix-ms string on the wire.
type MarketDepth struct {
	Envelope
	Symbol         string       `json:"-"`
	Bids           []PriceLevel `json:"bids"`
	Asks           []PriceLevel `json:"asks"`
	LastUpdate     JSONInt64    `json:"lastUpdate"`
	LastTradePrice Money        `json:"lastTradePrice"`
}

// UDF resolutions documented for GET /market/udf/history and the candle WS channel.
const (
	Resolution1m   = "1"
	Resolution5m   = "5"
	Resolution15m  = "15"
	Resolution30m  = "30"
	Resolution60m  = "60"
	Resolution180m = "180"
	Resolution240m = "240"
	Resolution360m = "360"
	Resolution720m = "720"
	ResolutionD    = "D"
	Resolution1D   = "1D"
	Resolution2D   = "2D"
	Resolution3D   = "3D"
)

// UDFHistoryQuery is GET /market/udf/history (TradingView UDF).
// `to` is required. `countback` overrides `from` when set. Max 500 candles.
type UDFHistoryQuery struct {
	Symbol     string
	Resolution string
	From       int64 // unix seconds; optional when Countback is set
	To         int64 // unix seconds; required
	Countback  int
	Page       int
}

// Normalize trims symbol (uppercased) and resolution.
func (q UDFHistoryQuery) Normalize() (UDFHistoryQuery, error) {
	out := q
	out.Symbol = strings.ToUpper(strings.TrimSpace(out.Symbol))
	out.Resolution = strings.TrimSpace(out.Resolution)
	if out.Symbol == "" {
		return UDFHistoryQuery{}, fmt.Errorf("types: udf history symbol is required")
	}
	if strings.EqualFold(out.Symbol, "all") {
		return UDFHistoryQuery{}, fmt.Errorf("types: udf history does not support symbol %q", "all")
	}
	if out.Resolution == "" {
		return UDFHistoryQuery{}, fmt.Errorf("types: udf history resolution is required")
	}
	if out.To == 0 {
		return UDFHistoryQuery{}, fmt.Errorf("types: udf history to (unix seconds) is required")
	}
	return out, nil
}

// UDFHistory is the TradingView UDF body. Success uses `s=ok` (not `status`).
// o/h/l/c/v are JSON numbers; they unmarshal into Money via Money.UnmarshalJSON.
type UDFHistory struct {
	S      string  `json:"s"`
	T      []int64 `json:"t"`
	O      []Money `json:"o"`
	H      []Money `json:"h"`
	L      []Money `json:"l"`
	C      []Money `json:"c"`
	V      []Money `json:"v"`
	ErrMsg string  `json:"errmsg,omitempty"`
}

// IsOK reports s=ok.
func (h UDFHistory) IsOK() bool { return strings.EqualFold(h.S, "ok") }

// HasNoData reports s=no_data (valid empty range).
func (h UDFHistory) HasNoData() bool { return strings.EqualFold(h.S, "no_data") }

// IsError reports s=error.
func (h UDFHistory) IsError() bool { return strings.EqualFold(h.S, "error") }

// Candle is one OHLC bar assembled from parallel UDF arrays.
type Candle struct {
	Time   int64
	Open   Money
	High   Money
	Low    Money
	Close  Money
	Volume Money
}

// Candles zips the UDF arrays. Length is min of the present series.
func (h UDFHistory) Candles() []Candle {
	n := len(h.T)
	n = minLen(n, len(h.O))
	n = minLen(n, len(h.H))
	n = minLen(n, len(h.L))
	n = minLen(n, len(h.C))
	n = minLen(n, len(h.V))
	if n <= 0 {
		return nil
	}
	out := make([]Candle, n)
	for i := 0; i < n; i++ {
		out[i] = Candle{
			Time:   h.T[i],
			Open:   atMoney(h.O, i),
			High:   atMoney(h.H, i),
			Low:    atMoney(h.L, i),
			Close:  atMoney(h.C, i),
			Volume: atMoney(h.V, i),
		}
	}
	return out
}

func atMoney(s []Money, i int) Money {
	if i < 0 || i >= len(s) {
		return ""
	}
	return s[i]
}

func minLen(a, b int) int {
	if a <= 0 {
		return 0
	}
	if b < a {
		return b
	}
	return a
}
