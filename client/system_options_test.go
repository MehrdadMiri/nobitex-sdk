package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func systemOptionsFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSystemOptionsFetchesAndParsesPrecisions(t *testing.T) {
	t.Parallel()
	fixture := systemOptionsFixture(t, "system_options.json")

	var gotPath, gotMethod, gotUA, gotAuth, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotUA = r.Header.Get("User-Agent")
		gotAuth = r.Header.Get("Authorization")
		gotKey = r.Header.Get("Nobitex-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("should-not-be-sent"),
	)
	if err != nil {
		t.Fatal(err)
	}

	opts, err := c.SystemOptions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q", gotMethod)
	}
	if gotPath != "/v2/options" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotUA != "TraderBot/MyBot-1.0.0" {
		t.Fatalf("User-Agent = %q", gotUA)
	}
	if gotAuth != "" || gotKey != "" {
		t.Fatalf("public call sent auth Authorization=%q Nobitex-Key=%q", gotAuth, gotKey)
	}
	if !opts.Status.IsOK() {
		t.Fatalf("status = %q", opts.Status)
	}

	if got := opts.AmountPrecisions()["BTCIRT"]; got != "0.000001" {
		t.Fatalf("amountPrecisions BTCIRT = %q", got)
	}
	if got := opts.PricePrecisions()["BTCIRT"]; got != "10" {
		t.Fatalf("pricePrecisions BTCIRT = %q", got)
	}
	if got := opts.PricePrecisions()["BTCUSDT"]; got != "0.01" {
		t.Fatalf("pricePrecisions BTCUSDT = %q", got)
	}

	if err := opts.ValidateOrderDecimals("BTCIRT", "0.001", "35650565900"); err != nil {
		t.Fatalf("helpers rejected valid decimals: %v", err)
	}
	if err := opts.ValidateOrderDecimals("ETHIRT", "0.00001", "10"); err != nil {
		t.Fatal(err)
	}
}

func TestSystemOptionsAPIError(t *testing.T) {
	t.Parallel()
	fixture := systemOptionsFixture(t, "system_options_failed.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.SystemOptions(context.Background())
	if !sdkerr.IsAPI(err) {
		t.Fatalf("err = %v", err)
	}
	var e *sdkerr.Error
	sdkerr.As(err, &e)
	if e.Code != "Maintenance" {
		t.Fatalf("code = %q", e.Code)
	}
}

func TestSystemOptionsFixturesPresent(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"system_options.json", "system_options_failed.json"} {
		b := systemOptionsFixture(t, name)
		if len(b) == 0 || !strings.Contains(string(b), "status") {
			t.Fatalf("fixture %s looks empty", name)
		}
	}
	raw := systemOptionsFixture(t, "system_options.json")
	if !strings.Contains(string(raw), "amountPrecisions") || !strings.Contains(string(raw), "pricePrecisions") {
		t.Fatal("fixture missing precision maps")
	}
}

func TestSystemOptionsLivePublic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live public call")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c, err := client.New(
		client.WithApp("nobitex-sdk-test", "0.1.0"),
		client.WithTimeout(12*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	opts, err := c.SystemOptions(ctx)
	if err != nil {
		if sdkerr.IsTransport(err) || sdkerr.IsRateLimited(err) {
			t.Skipf("live /v2/options unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if !opts.Status.IsOK() {
		t.Fatalf("status = %q", opts.Status)
	}
	if len(opts.AmountPrecisions()) == 0 || len(opts.PricePrecisions()) == 0 {
		t.Fatalf("empty precision maps amount=%d price=%d", len(opts.AmountPrecisions()), len(opts.PricePrecisions()))
	}

	mp, err := opts.MarketPrecision("BTCIRT")
	if err != nil {
		t.Fatalf("BTCIRT missing from live options: %v", err)
	}
	if mp.Amount == "" || mp.Price == "" {
		t.Fatalf("BTCIRT steps empty: %+v", mp)
	}
	if err := types.Money(mp.Amount).FitsStep(mp.Amount); err != nil {
		t.Fatalf("amount step %q is not a multiple of itself: %v", mp.Amount, err)
	}
	if err := opts.ValidateAmount("BTCIRT", mp.Amount); err != nil {
		t.Fatalf("ValidateAmount(step) = %v", err)
	}
	if err := opts.ValidatePrice("BTCIRT", mp.Price); err != nil {
		t.Fatalf("ValidatePrice(step) = %v", err)
	}
	if err := opts.ValidateOrderDecimals("BTCIRT", "0.0000000000001", mp.Price); err == nil {
		t.Fatal("expected tiny amount to fail against live BTCIRT step")
	}
}
