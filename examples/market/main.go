// Public P1 market-data example (no token).
//
//	go run ./examples/market
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

	c, err := client.New(client.WithApp("MyBot", "1.0.0"))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	stats, err := c.MarketStats(ctx, types.MarketStatsQuery{SrcCurrency: "btc", DstCurrency: "rls"})
	if err != nil {
		log.Fatal(err)
	}
	if st, ok := stats.Stat("BTCIRT"); ok {
		log.Printf("BTCIRT latest=%s change=%v", types.MoneyValue(st.Latest), st.DayChange)
	}

	trades, err := c.MarketTrades(ctx, "BTCIRT")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("BTCIRT trades=%d", len(trades.Trades))
}
