package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// UnmarshalJSON accepts a JSON string, number, or null.
// Nobitex documents monetary fields as strings; a few public payloads
// (UDF OHLC arrays, some integer lastUpdate fields) still use numbers.
func (m *Money) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*m = ""
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*m = Money(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("types: money: %w", err)
	}
	*m = Money(n.String())
	return nil
}

// JSONInt64 unmarshals a JSON number or a numeric string (depth lastUpdate
// is documented and returned as a string of unix milliseconds).
type JSONInt64 int64

// UnmarshalJSON accepts a JSON number, numeric string, or null (zero).
func (n *JSONInt64) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*n = 0
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*n = 0
			return nil
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("types: json int: %w", err)
		}
		*n = JSONInt64(v)
		return nil
	}
	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("types: json int: %w", err)
	}
	*n = JSONInt64(v)
	return nil
}

// Int64 returns the numeric value.
func (n JSONInt64) Int64() int64 { return int64(n) }

// FlexibleString unmarshals a JSON string or number as text (userLevel).
type FlexibleString string

// UnmarshalJSON accepts a JSON string, number, or null.
func (s *FlexibleString) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*s = ""
		return nil
	}
	if data[0] == '"' {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*s = FlexibleString(v)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("types: flexible string: %w", err)
	}
	*s = FlexibleString(n.String())
	return nil
}

func (s FlexibleString) String() string { return string(s) }
