package errors_test

import (
	"net/http"
	"testing"

	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
)

func TestCheckTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		status      int
		body        string
		wantNil     bool
		wantAPI     bool
		wantRate    bool
		wantCode    string
		wantBackOff int
	}{
		{name: "200 ok", status: 200, body: `{"status":"ok"}`, wantNil: true},
		{name: "200 success", status: 200, body: `{"status":"success"}`, wantNil: true},
		{name: "204 empty", status: 204, body: "", wantNil: true},
		{name: "200 non-json", status: 200, body: "not-json", wantNil: true},
		{name: "200 failed", status: 200, body: `{"status":"failed","code":"InvalidSymbol","message":"bad"}`, wantAPI: true, wantCode: "InvalidSymbol"},
		{name: "400 json", status: 400, body: `{"status":"failed","code":"ParseError"}`, wantAPI: true, wantCode: "ParseError"},
		{name: "401", status: 401, body: `{"status":"failed","code":"UnAuthenticated"}`, wantAPI: true, wantCode: "UnAuthenticated"},
		{name: "404", status: 404, body: `{"status":"failed","code":"NoOpenPosition"}`, wantAPI: true, wantCode: "NoOpenPosition"},
		{name: "429", status: 429, body: `{"status":"failed","code":"TooManyRequests","backOff":12,"limit":60}`, wantAPI: true, wantRate: true, wantCode: "TooManyRequests", wantBackOff: 12},
		{name: "502 text", status: 502, body: "bad gateway", wantAPI: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := sdkerr.Check(tc.status, []byte(tc.body))
			if tc.wantNil {
				if err != nil {
					t.Fatalf("Check = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if tc.wantAPI && !sdkerr.IsAPI(err) {
				t.Fatalf("IsAPI = false, err=%v", err)
			}
			if tc.wantRate != sdkerr.IsRateLimited(err) {
				t.Fatalf("IsRateLimited = %v want %v", sdkerr.IsRateLimited(err), tc.wantRate)
			}
			var e *sdkerr.Error
			if !sdkerr.As(err, &e) {
				t.Fatalf("As failed: %v", err)
			}
			if e.StatusCode != tc.status {
				t.Fatalf("StatusCode = %d", e.StatusCode)
			}
			if tc.wantCode != "" && e.Code != tc.wantCode {
				t.Fatalf("Code = %q want %q", e.Code, tc.wantCode)
			}
			if tc.wantBackOff != 0 && e.BackOff != tc.wantBackOff {
				t.Fatalf("BackOff = %d", e.BackOff)
			}
			if e.Kind != sdkerr.KindAPI {
				t.Fatalf("Kind = %q", e.Kind)
			}
		})
	}
}

func TestIsRateLimitedHTTPStatusWithoutCode(t *testing.T) {
	t.Parallel()
	err := sdkerr.Check(http.StatusTooManyRequests, []byte("slow down"))
	if !sdkerr.IsRateLimited(err) {
		t.Fatalf("HTTP 429 without JSON code should still be rate-limited, err=%v", err)
	}
}
