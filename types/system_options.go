package types

import (
	"fmt"
	"strings"
)

// SystemOptions is the GET /v2/options body.
//
// Official docs: https://apidocs.nobitex.ir/options/get-system-options
// (OpenAPI: https://apidocs.nobitex.ir/openapi/options.yaml). Public, no token.
type SystemOptions struct {
	Envelope
	Features Features       `json:"features"`
	Coins    []CoinOptions  `json:"coins"`
	Nobitex  NobitexOptions `json:"nobitex"`
}

// Features is the documented features object on GET /v2/options.
type Features struct {
	FCMEnabled      bool     `json:"fcmEnabled"`
	Chat            string   `json:"chat"`
	WalletsToNet    bool     `json:"walletsToNet"`
	AutoKYC         bool     `json:"autoKYC"`
	EnabledFeatures []string `json:"enabledFeatures"`
	BetaFeatures    []string `json:"betaFeatures"`
}

// CoinOptions is one entry of the documented coins array.
type CoinOptions struct {
	Coin             string                    `json:"coin"`
	Name             string                    `json:"name"`
	DefaultNetwork   string                    `json:"defaultNetwork"`
	DisplayPrecision Money                     `json:"displayPrecision"`
	StdName          string                    `json:"stdName"`
	NetworkList      map[string]NetworkOptions `json:"networkList"`
}

// NetworkOptions is one coin network from GET /v2/options.
type NetworkOptions struct {
	Network                 string `json:"network"`
	Name                    string `json:"name"`
	IsDefault               bool   `json:"isDefault"`
	Beta                    bool   `json:"beta"`
	AddressRegex            string `json:"addressRegex,omitempty"`
	MemoRequired            bool   `json:"memoRequired,omitempty"`
	MemoRegex               string `json:"memoRegex,omitempty"`
	DepositEnable           bool   `json:"depositEnable"`
	MinConfirm              int    `json:"minConfirm"`
	WithdrawEnable          bool   `json:"withdrawEnable"`
	WithdrawIntegerMultiple Money  `json:"withdrawIntegerMultiple,omitempty"`
	WithdrawFee             Money  `json:"withdrawFee,omitempty"`
	WithdrawMin             Money  `json:"withdrawMin,omitempty"`
	WithdrawMax             Money  `json:"withdrawMax,omitempty"`
}

// NobitexOptions is the documented nobitex object. AmountPrecisions and
// PricePrecisions are the market step sizes used to validate order decimals.
type NobitexOptions struct {
	AllCurrencies     []string         `json:"allCurrencies"`
	ActiveCurrencies  []string         `json:"activeCurrencies"`
	XchangeCurrencies []string         `json:"xchangeCurrencies"`
	TopCurrencies     []string         `json:"topCurrencies"`
	TestingCurrencies []string         `json:"testingCurrencies"`
	MinOrders         map[string]Money `json:"minOrders"`
	AmountPrecisions  PrecisionMap     `json:"amountPrecisions"`
	PricePrecisions   PrecisionMap     `json:"pricePrecisions"`
	GiftCard          *GiftCard        `json:"giftCard"`
}

// GiftCard is the documented gift-card subsection of nobitex.
type GiftCard struct {
	PhysicalFee Money `json:"physicalFee"`
}

// PrecisionMap is market symbol → smallest allowed increment (a Money string).
// Keys are documented as uppercase with no separator (BTCIRT, BTCUSDT). Some
// live markets also include an underscore (100K_FLOKIIRT).
type PrecisionMap map[string]Money

// MarketPrecision is the amount and price step for one market.
type MarketPrecision struct {
	Market string
	Amount Money
	Price  Money
}

// AmountPrecisions returns nobitex.amountPrecisions (may be nil).
func (o *SystemOptions) AmountPrecisions() PrecisionMap {
	if o == nil {
		return nil
	}
	return o.Nobitex.AmountPrecisions
}

// PricePrecisions returns nobitex.pricePrecisions (may be nil).
func (o *SystemOptions) PricePrecisions() PrecisionMap {
	if o == nil {
		return nil
	}
	return o.Nobitex.PricePrecisions
}

// AmountPrecision looks up the amount step for market.
func (o *SystemOptions) AmountPrecision(market string) (Money, bool) {
	if o == nil {
		return "", false
	}
	return o.Nobitex.AmountPrecisions.Lookup(market)
}

// PricePrecision looks up the price step for market.
func (o *SystemOptions) PricePrecision(market string) (Money, bool) {
	if o == nil {
		return "", false
	}
	return o.Nobitex.PricePrecisions.Lookup(market)
}

// MarketPrecision returns both amount and price steps for market.
func (o *SystemOptions) MarketPrecision(market string) (MarketPrecision, error) {
	if strings.TrimSpace(market) == "" {
		return MarketPrecision{}, fmt.Errorf("%w", ErrEmptyMarket)
	}
	amount, ok := o.AmountPrecision(market)
	if !ok {
		return MarketPrecision{}, fmt.Errorf("%w: %s amount", ErrUnknownMarket, strings.TrimSpace(market))
	}
	price, ok := o.PricePrecision(market)
	if !ok {
		return MarketPrecision{}, fmt.Errorf("%w: %s price", ErrUnknownMarket, strings.TrimSpace(market))
	}
	return MarketPrecision{
		Market: NormalizeMarket(market),
		Amount: amount,
		Price:  price,
	}, nil
}

// ValidateAmount checks that amount is a positive multiple of the market's
// amountPrecisions step. Intended for later order-placement code.
func (o *SystemOptions) ValidateAmount(market string, amount Money) error {
	return o.AmountPrecisions().ValidatePositive(market, amount)
}

// ValidatePrice checks that price is a positive multiple of the market's
// pricePrecisions step. Intended for later order-placement code (limit / stop).
func (o *SystemOptions) ValidatePrice(market string, price Money) error {
	return o.PricePrecisions().ValidatePositive(market, price)
}

// ValidateOrderDecimals validates amount and (if non-empty) price against the
// market's documented step sizes. An empty price is skipped so market orders
// can reuse this helper without a limit price.
func (o *SystemOptions) ValidateOrderDecimals(market string, amount, price Money) error {
	if err := o.ValidateAmount(market, amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}
	if strings.TrimSpace(string(price)) == "" {
		return nil
	}
	if err := o.ValidatePrice(market, price); err != nil {
		return fmt.Errorf("price: %w", err)
	}
	return nil
}

// Lookup returns the step for market. Exact key first, then trimmed, then
// uppercased, then a case-insensitive scan.
func (p PrecisionMap) Lookup(market string) (Money, bool) {
	if p == nil {
		return "", false
	}
	if v, ok := p[market]; ok {
		return v, true
	}
	trimmed := strings.TrimSpace(market)
	if trimmed == "" {
		return "", false
	}
	if v, ok := p[trimmed]; ok {
		return v, true
	}
	upper := strings.ToUpper(trimmed)
	if v, ok := p[upper]; ok {
		return v, true
	}
	for k, v := range p {
		if strings.EqualFold(k, trimmed) {
			return v, true
		}
	}
	return "", false
}

// Validate reports whether value is an integer multiple of the market step.
// Zero is a multiple of any positive step; use ValidatePositive for orders.
func (p PrecisionMap) Validate(market string, value Money) error {
	if strings.TrimSpace(market) == "" {
		return ErrEmptyMarket
	}
	step, ok := p.Lookup(market)
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownMarket, strings.TrimSpace(market))
	}
	return value.FitsStep(step)
}

// ValidatePositive is Validate plus a requirement that value > 0.
func (p PrecisionMap) ValidatePositive(market string, value Money) error {
	if err := requirePositive(value); err != nil {
		return err
	}
	return p.Validate(market, value)
}

// Truncate snaps value down (toward zero) to the market step.
func (p PrecisionMap) Truncate(market string, value Money) (Money, error) {
	if strings.TrimSpace(market) == "" {
		return "", ErrEmptyMarket
	}
	step, ok := p.Lookup(market)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownMarket, strings.TrimSpace(market))
	}
	return value.TruncateToStep(step)
}

// NormalizeMarket trims and uppercases a market symbol (BTCIRT). Underscores
// in live keys such as 100K_FLOKIIRT are preserved.
func NormalizeMarket(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}
