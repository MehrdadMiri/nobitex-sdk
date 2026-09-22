package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestMarginMarketsPublic(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "margin_markets.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithApp("MyBot", "1.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.MarginMarkets(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.path != "/margin/markets/list" {
		t.Fatalf("%s %s", got.method, got.path)
	}
	if got.auth != "" {
		t.Fatalf("unauthenticated call sent Authorization=%q", got.auth)
	}
	m, ok := resp.Market("btcirt")
	if !ok || m.SrcCurrency != "btc" || m.MaxLeverage != "5" {
		t.Fatalf("%+v ok=%v", m, ok)
	}
}

func TestMarginMarketsDetailsBodyAndAuth(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "margin_markets.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("secret-token"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.MarginMarkets(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if got.auth != "Token secret-token" {
		t.Fatalf("auth=%q", got.auth)
	}
	var body map[string]any
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["details"] != true {
		t.Fatalf("body=%v", body)
	}
	m, ok := resp.Market("BTCUSDT")
	if !ok || types.MoneyValue(m.BuyMaxDelegationInSrcCurrency) != "0.002" {
		t.Fatalf("%+v ok=%v", m, ok)
	}
}

func TestMarginDelegationLimit(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "margin_delegation.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.MarginDelegationLimit(context.Background(), " btcirt ")
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/margin/v2/delegation-limit" {
		t.Fatalf("path=%q", got.path)
	}
	if parseQuery(got.rawQuery)["market"] != "BTCIRT" {
		t.Fatalf("query=%q", got.rawQuery)
	}
	lim, ok := resp.LimitFor(types.OrderSideSell, "2")
	if !ok || lim != "0.37" {
		t.Fatalf("limit=%q ok=%v", lim, ok)
	}

	plain, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plain.MarginDelegationLimit(context.Background(), "BTCIRT"); err == nil || !strings.Contains(err.Error(), "READ") {
		t.Fatalf("err=%v", err)
	}
}

func TestTransferWalletSpotToMargin(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "margin_transfer.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.TransferSpotToMargin(context.Background(), " USDT ", "0.01")
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPost || got.path != "/wallets/transfer" {
		t.Fatalf("%s %s", got.method, got.path)
	}
	var body map[string]string
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["currency"] != "usdt" || body["src"] != "spot" || body["dst"] != "margin" || body["amount"] != "0.01" {
		t.Fatalf("body=%v", body)
	}
	if resp.DstWallet.Type != types.WalletSpot || resp.SrcWallet.Type != types.WalletMargin {
		t.Fatalf("%+v", resp)
	}
}

func TestTransferWalletFailed(t *testing.T) {
	t.Parallel()
	srv := newP1Server(t, http.StatusOK, "margin_transfer_failed.json", nil)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.TransferMarginToSpot(context.Background(), "btc", "0.01")
	if err == nil || !sdkerr.IsAPI(err) {
		t.Fatalf("err=%v", err)
	}
}
