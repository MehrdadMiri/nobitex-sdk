package client_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func TestWebSocketOverviewStub(t *testing.T) {
	t.Parallel()
	ov := client.WebSocketOverview()
	if ov.ProductionURL != types.DefaultWebSocketURL {
		t.Fatalf("url=%q", ov.ProductionURL)
	}
	if ov.TestnetURL != types.TestnetWebSocketURL {
		t.Fatalf("testnet=%q", ov.TestnetURL)
	}
	if ov.TokenPath != "/auth/ws/token/" || ov.TokenTTL != types.WebSocketTokenTTL {
		t.Fatalf("%+v", ov)
	}
	joined := strings.Join(ov.PublicChannels, " ")
	for _, pat := range []string{
		"public:orderbook-{MARKET_SYMBOL}",
		"public:candle-{MARKET_SYMBOL}-{RESOLUTION}",
		"public:trades-{MARKET_SYMBOL}",
		"public:market-stats-{MARKET_SYMBOL}",
		"public:market-stats-all",
	} {
		if !strings.Contains(joined, pat) {
			t.Fatalf("missing public channel %s in %v", pat, ov.PublicChannels)
		}
	}
	priv := strings.Join(ov.PrivateChannels, " ")
	if !strings.Contains(priv, "private:orders#") || !strings.Contains(priv, "private:trades#") {
		t.Fatalf("private=%v", ov.PrivateChannels)
	}
}

func TestWebSocketToken(t *testing.T) {
	t.Parallel()
	var got p1Capture
	srv := newP1Server(t, http.StatusOK, "ws_token.json", &got)
	defer srv.Close()
	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken("secret-token"),
	)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.WebSocketToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodGet || got.path != "/auth/ws/token/" {
		t.Fatalf("%s %s", got.method, got.path)
	}
	if got.ua != "TraderBot/MyBot-1.0.0" || got.auth != "Token secret-token" {
		t.Fatalf("ua=%q auth=%q", got.ua, got.auth)
	}
	if resp.Token != "eyJhbGciOiJFUzUxMiIsInR5cCI6IkpXVCJ9.example" {
		t.Fatalf("token=%q", resp.Token)
	}
	if err := resp.ValidateToken(); err != nil {
		t.Fatal(err)
	}

	plain, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plain.WebSocketToken(context.Background()); err == nil || !strings.Contains(err.Error(), "READ") {
		t.Fatalf("err=%v", err)
	}
}
