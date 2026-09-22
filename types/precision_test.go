package types_test

import (
	"errors"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestMoneyScale(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   types.Money
		want int
	}{
		{"0.000001", 6},
		{"10", 0},
		{"0.01", 2},
		{"0.0000001", 7},
		{" 10.0 ", 1},
	}
	for _, tc := range cases {
		if got := tc.in.Scale(); got != tc.want {
			t.Errorf("Scale(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestMoneyFitsStep(t *testing.T) {
	t.Parallel()
	cases := []struct {
		value, step types.Money
		ok          bool
	}{
		{"0.000001", "0.000001", true},
		{"0.001", "0.000001", true},
		{"0.0000015", "0.000001", false},
		{"120", "10", true},
		{"10.0", "10", true},
		{"123", "10", false},
		{"1.00", "0.01", true},
		{"1.001", "0.01", false},
		{"68000.01", "0.01", true},
		{"0.0000001", "0.0000001", true},
		{"0.00000015", "0.0000001", false},
		{"0", "0.01", true},
		{"", "0.01", false},
		{"abc", "0.01", false},
		{"1", "0", false},
		{"1", "-0.01", false},
	}
	for _, tc := range cases {
		err := tc.value.FitsStep(tc.step)
		if tc.ok && err != nil {
			t.Errorf("FitsStep(%q, %q) = %v, want nil", tc.value, tc.step, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("FitsStep(%q, %q) = nil, want error", tc.value, tc.step)
		}
	}
}

func TestMoneyTruncateToStep(t *testing.T) {
	t.Parallel()
	cases := []struct {
		value, step, want types.Money
	}{
		{"123", "10", "120"},
		{"1.239", "0.01", "1.23"},
		{"0.0000015", "0.000001", "0.000001"},
		{"10", "10", "10"},
		{"0.019", "0.01", "0.01"},
		{"35650565905", "10", "35650565900"},
	}
	for _, tc := range cases {
		got, err := tc.value.TruncateToStep(tc.step)
		if err != nil {
			t.Errorf("TruncateToStep(%q, %q): %v", tc.value, tc.step, err)
			continue
		}
		if got != tc.want {
			t.Errorf("TruncateToStep(%q, %q) = %q, want %q", tc.value, tc.step, got, tc.want)
		}
		if err := got.FitsStep(tc.step); err != nil {
			t.Errorf("truncated %q does not fit step %q: %v", got, tc.step, err)
		}
	}
}

func TestFitsStepSentinels(t *testing.T) {
	t.Parallel()
	if err := types.Money("1.001").FitsStep("0.01"); !errors.Is(err, types.ErrNotMultiple) {
		t.Fatalf("ErrNotMultiple: %v", err)
	}
	if err := types.Money("1").FitsStep("0"); !errors.Is(err, types.ErrInvalidStep) {
		t.Fatalf("ErrInvalidStep: %v", err)
	}
	if err := types.Money("nope").FitsStep("0.01"); !errors.Is(err, types.ErrInvalidDecimal) {
		t.Fatalf("ErrInvalidDecimal: %v", err)
	}
}
