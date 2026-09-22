package client_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestUserProfileTokenAuth(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "user_profile.json", &got)
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("secret-token"),
	)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.UserProfile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.path != "/users/profile" {
		t.Fatalf("request %s %s", got.method, got.path)
	}
	if got.ua != "TraderBot/MyBot-1.0.0" || got.auth != "Token secret-token" {
		t.Fatalf("ua=%q auth=%q", got.ua, got.auth)
	}
	if resp.Profile.WebsocketAuthParam != "1987577cdf7c7422dee369e188e276ee" {
		t.Fatalf("profile=%+v", resp.Profile)
	}
	if resp.Profile.DisplayNickname() != "trader" || resp.TradeStats.MonthTrades != 12 {
		t.Fatalf("%+v stats=%+v", resp.Profile, resp.TradeStats)
	}
}

func TestUserProfileRequiresAuth(t *testing.T) {
	t.Parallel()
	c, err := client.New(client.WithBaseURL("http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.UserProfile(context.Background())
	if err == nil || !strings.Contains(err.Error(), "READ") {
		t.Fatalf("err=%v", err)
	}
}

func TestUserLimitationsAndWallets(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "user_limitations.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	lim, err := c.UserLimitations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPost || got.path != "/users/limitations" {
		t.Fatalf("%s %s", got.method, got.path)
	}
	if lim.Limitations.UserLevel.String() != "1" {
		t.Fatalf("userLevel=%q", lim.Limitations.UserLevel)
	}

	wSrv := newP1Server(t, http.StatusOK, "user_wallets.json", &got)
	defer wSrv.Close()
	c2, err := client.New(client.WithBaseURL(wSrv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	wallets, err := c2.ListWallets(context.Background(), types.WalletListRequest{Type: " SPOT "})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/users/wallets/list" || got.method != http.MethodPost {
		t.Fatalf("%s %s", got.method, got.path)
	}
	var body map[string]string
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["type"] != "spot" {
		t.Fatalf("body=%v", body)
	}
	btc, ok := wallets.WalletByCurrency("BTC")
	if !ok || btc.Balance != "0.6000000000" || btc.RialBalance.Int64() != 120000000000 {
		t.Fatalf("btc=%+v ok=%v", btc, ok)
	}
}

func TestListWalletsV2AndBalance(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "user_wallets_v2.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.ListWalletsV2(context.Background(), types.WalletsV2Request{Currencies: "BTC, RLS", Type: types.WalletSpot})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v2/wallets" {
		t.Fatalf("path=%q", got.path)
	}
	w, ok := resp.Wallet("BTC")
	if !ok || w.Balance != "0.6" {
		t.Fatalf("%+v ok=%v", w, ok)
	}

	bSrv := newP1Server(t, http.StatusOK, "user_balance.json", &got)
	defer bSrv.Close()
	c2, err := client.New(client.WithBaseURL(bSrv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	bal, err := c2.WalletBalance(context.Background(), " BTC ")
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/users/wallets/balance" {
		t.Fatalf("path=%q", got.path)
	}
	var body map[string]string
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}
	if body["currency"] != "btc" || bal.Balance != "0.6" {
		t.Fatalf("body=%v bal=%q", body, bal.Balance)
	}
}

func TestListDepositsQuery(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "user_deposits.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.ListDeposits(context.Background(), types.DepositListQuery{Wallet: "all", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/users/wallets/deposits/list" {
		t.Fatalf("path=%q", got.path)
	}
	q := parseQuery(got.rawQuery)
	if q["wallet"] != "all" || q["page"] != "1" || q["pageSize"] != "20" {
		t.Fatalf("query=%v", q)
	}
	if len(resp.Deposits) != 1 || resp.Deposits[0].Currency != "btc" {
		t.Fatalf("%+v", resp)
	}
}

func TestUserProfileAPIKeyREADHeaders(t *testing.T) {
	t.Parallel()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := base64.URLEncoding.EncodeToString(priv.Seed())
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "user_profile.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithAPIKey("pub", secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.UserProfile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got.key != "pub" || got.sig == "" || got.ts == "" || got.auth != "" {
		t.Fatalf("key=%q sig=%q ts=%q auth=%q", got.key, got.sig, got.ts, got.auth)
	}
}

func TestUserProfileFailedStatus(t *testing.T) {
	t.Parallel()
	srv := newP1Server(t, http.StatusOK, "unauthenticated.json", nil)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.UserProfile(context.Background())
	if err == nil || !sdkerr.IsAPI(err) {
		t.Fatalf("err=%v", err)
	}
}
