package errors_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestCheckNilOnOK(t *testing.T) {
	t.Parallel()
	body := []byte(`{"status":"ok","lastTradePrice":"1"}`)
	if err := sdkerr.Check(http.StatusOK, body); err != nil {
		t.Fatalf("Check(200, ok) = %v", err)
	}
}

func TestCheckFailedBodyOnHTTP200(t *testing.T) {
	t.Parallel()
	body := []byte(`{"status":"failed","code":"InvalidSymbol","message":"bad symbol"}`)
	err := sdkerr.Check(http.StatusOK, body)
	if err == nil {
		t.Fatal("expected API error")
	}
	var e *sdkerr.Error
	if !sdkerr.As(err, &e) {
		t.Fatalf("As failed: %v", err)
	}
	if e.Kind != sdkerr.KindAPI {
		t.Errorf("Kind = %q", e.Kind)
	}
	if e.Status != types.StatusFailed {
		t.Errorf("Status = %q", e.Status)
	}
	if e.Code != "InvalidSymbol" {
		t.Errorf("Code = %q", e.Code)
	}
	if e.Message != "bad symbol" {
		t.Errorf("Message = %q", e.Message)
	}
	if !sdkerr.IsAPI(err) {
		t.Error("IsAPI = false")
	}
	if !strings.Contains(e.Error(), "InvalidSymbol") {
		t.Errorf("Error() = %q", e.Error())
	}
}

func TestCheckHTTPError(t *testing.T) {
	t.Parallel()
	body := []byte(`{"status":"failed","code":"UnAuthenticated","message":"nope"}`)
	err := sdkerr.Check(http.StatusUnauthorized, body)
	var e *sdkerr.Error
	if !sdkerr.As(err, &e) {
		t.Fatalf("As failed: %v", err)
	}
	if e.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d", e.StatusCode)
	}
}

func TestParseTooManyRequests(t *testing.T) {
	t.Parallel()
	body := []byte(`{"status":"failed","code":"TooManyRequests","message":"wait","backOff":12,"limit":60}`)
	err := sdkerr.Check(http.StatusTooManyRequests, body)
	if !sdkerr.IsRateLimited(err) {
		t.Fatalf("IsRateLimited = false for %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.BackOff != 12 || e.Limit != 60 {
		t.Errorf("BackOff=%d Limit=%d", e.BackOff, e.Limit)
	}
}

func TestFromResponseNonJSON(t *testing.T) {
	t.Parallel()
	err := sdkerr.Check(http.StatusBadGateway, []byte("bad gateway"))
	var e *sdkerr.Error
	if !sdkerr.As(err, &e) {
		t.Fatal("expected error")
	}
	if e.Message != "bad gateway" {
		t.Errorf("Message = %q", e.Message)
	}
}

func TestTransportAndDecode(t *testing.T) {
	t.Parallel()
	tr := sdkerr.Transport(fmt.Errorf("connection refused"))
	if !sdkerr.IsTransport(tr) {
		t.Error("IsTransport = false")
	}
	if !strings.Contains(tr.Error(), "connection refused") {
		t.Errorf("Error() = %q", tr.Error())
	}

	dec := sdkerr.Decode(200, []byte("{"), fmt.Errorf("unexpected EOF"))
	if !sdkerr.IsDecode(dec) {
		t.Error("IsDecode = false")
	}
}

func TestEnvelopeRoundTrip(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"status":"failed","code":"X","message":"m","backOff":1,"limit":2}`)
	var env types.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if !env.Status.IsFailed() {
		t.Fatal("expected failed")
	}
}
