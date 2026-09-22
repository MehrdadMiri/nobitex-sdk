package client_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestListWithdrawsReadOnly(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "withdraws_list.json", &got)
	defer srv.Close()
	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("secret-token"),
	)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.ListWithdraws(context.Background(), types.WithdrawListQuery{
		Wallet:   "all",
		From:     "2024-01-01",
		To:       "2024-02-01",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.path != "/users/wallets/withdraws/list" {
		t.Fatalf("%s %s", got.method, got.path)
	}
	if got.ua != "TraderBot/MyBot-1.0.0" || got.auth != "Token secret-token" {
		t.Fatalf("ua=%q auth=%q", got.ua, got.auth)
	}
	q := parseQuery(got.rawQuery)
	if q["wallet"] != "all" || q["from"] != "2024-01-01" || q["to"] != "2024-02-01" {
		t.Fatalf("query=%v", q)
	}
	w, ok := resp.WithdrawByID(432)
	if !ok || w.Status != types.WithdrawDone || w.Currency != "usdt" || w.Network != "ETH" {
		t.Fatalf("%+v ok=%v", w, ok)
	}
	rial, ok := resp.WithdrawByID(503)
	if !ok || rial.Network != "FIAT_MONEY" {
		t.Fatalf("rial=%+v", rial)
	}
}

func TestGetWithdraw(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "withdraw_detail.json", &got)
	defer srv.Close()
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.GetWithdraw(context.Background(), 432)
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/withdraws/432" {
		t.Fatalf("path=%q", got.path)
	}
	if resp.Withdraw == nil || resp.Withdraw.ID != 432 || resp.Withdraw.Amount != "0.05000000" {
		t.Fatalf("%+v", resp.Withdraw)
	}

	if _, err := c.GetWithdraw(context.Background(), 0); err == nil {
		t.Fatal("expected invalid id")
	}
	plain, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plain.ListWithdraws(context.Background(), types.WithdrawListQuery{}); err == nil || !strings.Contains(err.Error(), "READ") {
		t.Fatalf("err=%v", err)
	}
}
