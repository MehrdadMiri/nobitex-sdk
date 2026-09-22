package types_test

import (
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestStatusHelpers(t *testing.T) {
	t.Parallel()
	if !types.StatusOK.IsOK() || types.StatusOK.IsFailed() {
		t.Fatal("ok")
	}
	if !types.StatusSuccess.IsOK() {
		t.Fatal("success should be OK")
	}
	if !types.StatusFailed.IsFailed() || types.StatusFailed.IsOK() {
		t.Fatal("failed")
	}
}
