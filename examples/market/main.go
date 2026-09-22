// Public P1 market-data + WebSocket overview (no token).
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

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stats, err := c.MarketStats(ctx, types.MarketStatsQuery{SrcCurrency: "btc", DstCurrency: "rls"})
	if err != nil {
		log.Fatal(err)
	}
	if st, ok := stats.Stat("BTCIRT"); ok {
		log.Printf("stats BTCIRT latest=%s change=%v", types.MoneyValue(st.Latest), st.DayChange)
	}

	trades, err := c.MarketTrades(ctx, "BTCIRT")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("trades=%d", len(trades.Trades))

	depth, err := c.MarketDepth(ctx, "BTCIRT")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("depth bids=%d asks=%d last=%s", len(depth.Bids), len(depth.Asks), depth.LastTradePrice)

	ohlc, err := c.MarketOHLC(ctx, types.UDFHistoryQuery{
		Symbol:     "BTCIRT",
		Resolution: types.Resolution60m,
		To:         time.Now().Unix(),
		Countback:  3,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ohlc bars=%d s=%s", len(ohlc.Candles()), ohlc.S)

	markets, err := c.MarginMarkets(ctx, false)
	if err != nil {
		log.Fatal(err)
	}
	if m, ok := markets.Market("BTCIRT"); ok {
		log.Printf("margin market BTCIRT maxLeverage=%s", m.MaxLeverage)
	}

	ov := client.WebSocketOverview()
	log.Printf("ws=%s orderbook=%s", ov.ProductionURL, types.PublicOrderBookChannel("BTCIRT"))
}
