// Public order book example (no token).
//
//	go run ./examples/orderbook
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
)

func main() {
	log.SetOutput(os.Stderr)

	c, err := client.New(
		client.WithApp("MyBot", "1.0.0"), // User-Agent: TraderBot/MyBot-1.0.0
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// GET /v3/orderbook/BTCIRT — public, no Authorization header.
	book, err := c.OrderBook(ctx, "BTCIRT")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("BTCIRT last=%s bids=%d asks=%d", book.LastTradePrice, len(book.Bids), len(book.Asks))
	if len(book.Bids) > 0 {
		log.Printf("best bid %s x %s", book.Bids[0].Price, book.Bids[0].Amount)
	}

	// GET /v3/orderbook/all — consolidated books keyed by market.
	all, err := c.OrderBookAll(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if btc, ok := all.Book("BTCIRT"); ok {
		log.Printf("all-markets BTCIRT last=%s books=%d", btc.LastTradePrice, len(all.Books))
	}
}
