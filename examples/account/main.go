// Authenticated P1 READ flow (env token). No TRADE — does not place orders,
// transfer wallets, or submit withdrawals.
//
//	export NOBITEX_TOKEN=...                 # or NOBITEX_API_KEY + NOBITEX_API_SECRET (READ)
//	go run ./examples/account
//
// Docs: https://apidocs.nobitex.ir
// Repo: https://github.com/MehrdadMiri/nobitex-sdk
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func main() {
	log.SetOutput(os.Stderr)

	c, err := client.NewFromEnv(client.WithApp("MyBot", "1.0.0"))
	if err != nil {
		log.Fatal(err)
	}
	if c.Auth() == nil {
		log.Fatal("set NOBITEX_TOKEN or NOBITEX_API_KEY+NOBITEX_API_SECRET (see .env.example)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	profile, err := c.UserProfile(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("profile nick=%s verified=%v", profile.Profile.DisplayNickname(), profile.Profile.Verified)

	wallets, err := c.ListWallets(ctx, types.WalletListRequest{Type: types.WalletSpot})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("spot wallets=%d", len(wallets.Wallets))
	if btc, ok := wallets.WalletByCurrency("btc"); ok {
		log.Printf("btc balance=%s", btc.Balance)
	}

	bal, err := c.WalletBalance(ctx, "usdt")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("usdt balance=%s", bal.Balance)

	deposits, err := c.ListDeposits(ctx, types.DepositListQuery{PageSize: 10})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("deposits=%d", len(deposits.Deposits))

	withdraws, err := c.ListWithdraws(ctx, types.WithdrawListQuery{PageSize: 10})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("withdraws=%d (read-only; submit/confirm/cancel are out of scope)", len(withdraws.Withdraws))

	trades, err := c.ListUserTrades(ctx, types.UserTradesQuery{})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("user trades=%d", len(trades.Trades))

	if lim, err := c.MarginDelegationLimit(ctx, "BTCIRT"); err != nil {
		log.Printf("delegation-limit: %v", err)
	} else if remaining, ok := lim.LimitFor(types.OrderSideSell, "2"); ok {
		log.Printf("BTCIRT sell leverage=2 remaining=%s", remaining)
	}

	tok, err := c.WebSocketToken(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := tok.ValidateToken(); err != nil {
		log.Fatal(err)
	}
	log.Printf("ws token present=%v private=%s", tok.Token != "", types.PrivateOrdersChannel(profile.Profile.WebsocketAuthParam))
}
