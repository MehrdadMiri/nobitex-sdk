package types

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// Sentinel errors for order decimal validation.
var (
	ErrEmptyMarket    = errors.New("types: market symbol is required")
	ErrUnknownMarket  = errors.New("types: unknown market precision")
	ErrInvalidDecimal = errors.New("types: invalid decimal")
	ErrNotMultiple    = errors.New("types: value is not a multiple of the market step")
	ErrNonPositive    = errors.New("types: value must be positive")
	ErrInvalidStep    = errors.New("types: precision step must be positive")
)

// Scale returns the number of digits after the decimal point in m as written.
// "0.000001" → 6, "10" → 0, "0.01" → 2.
func (m Money) Scale() int {
	s := strings.TrimSpace(string(m))
	i := strings.IndexByte(s, '.')
	if i < 0 {
		return 0
	}
	return len(s) - i - 1
}

// FitsStep reports whether m is an integer multiple of step.
// Nobitex documents amountPrecisions / pricePrecisions as the smallest allowed
// increment (for example BTCIRT amount 0.000001, BTCIRT price 10).
func (m Money) FitsStep(step Money) error {
	v, err := parseDecimal(m)
	if err != nil {
		return err
	}
	s, err := parseDecimal(step)
	if err != nil {
		return err
	}
	if s.Sign() <= 0 {
		return fmt.Errorf("%w: %s", ErrInvalidStep, step)
	}
	q := new(big.Rat).Quo(v, s)
	if !q.IsInt() {
		return fmt.Errorf("%w: %s vs step %s", ErrNotMultiple, m, step)
	}
	return nil
}

// TruncateToStep returns the multiple of step closest to m toward zero.
func (m Money) TruncateToStep(step Money) (Money, error) {
	v, err := parseDecimal(m)
	if err != nil {
		return "", err
	}
	s, err := parseDecimal(step)
	if err != nil {
		return "", err
	}
	if s.Sign() <= 0 {
		return "", fmt.Errorf("%w: %s", ErrInvalidStep, step)
	}
	q := new(big.Rat).Quo(v, s)
	n := new(big.Int).Quo(q.Num(), q.Denom()) // toward zero
	out := new(big.Rat).Mul(new(big.Rat).SetInt(n), s)
	return formatRat(out, step.Scale()), nil
}

func requirePositive(m Money) error {
	v, err := parseDecimal(m)
	if err != nil {
		return err
	}
	if v.Sign() <= 0 {
		return fmt.Errorf("%w: %s", ErrNonPositive, m)
	}
	return nil
}

func parseDecimal(m Money) (*big.Rat, error) {
	s := strings.TrimSpace(string(m))
	if s == "" {
		return nil, fmt.Errorf("%w: empty", ErrInvalidDecimal)
	}
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return nil, fmt.Errorf("%w: %q", ErrInvalidDecimal, s)
	}
	return r, nil
}

func formatRat(r *big.Rat, scale int) Money {
	if scale < 0 {
		scale = 0
	}
	s := r.FloatString(scale)
	if scale > 0 {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	if s == "" || s == "-" {
		s = "0"
	}
	return Money(s)
}
