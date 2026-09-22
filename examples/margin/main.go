// Authenticated margin flow using env-based credentials (no secrets in source).
//
//	export NOBITEX_TOKEN=...                 # or NOBITEX_API_KEY + NOBITEX_API_SECRET
//	go run ./examples/margin                 # lists positions + margin orders (no funds)
//	NOBITEX_EXAMPLE_TRADE=1 go run ./examples/margin  # also places + cancels (TRADE, real funds)
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

	// NewFromEnv reads NOBITEX_TOKEN or NOBITEX_API_KEY+NOBITEX_API_SECRET.
	// Never hardcode tokens — copy .env.example and fill locally.
	c, err := client.NewFromEnv(client.WithApp("MyBot", "1.0.0"))
	if err != nil {
		log.Fatal(err)
	}
	if c.Auth() == nil {
		log.Fatal("set NOBITEX_TOKEN or NOBITEX_API_KEY+NOBITEX_API_SECRET (see .env.example)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req := types.NewMarginLimitOrder(types.OrderSideSell, "btc", "usdt", "0.01", "13400000000")
	req.ClientOrderID = "example-margin-1"

	// Place is TRADE and moves funds. Default is dry-run so CI / accidental
	// `go run` cannot spend. Set NOBITEX_EXAMPLE_TRADE=1 to POST for real.
	if os.Getenv("NOBITEX_EXAMPLE_TRADE") == "1" {
		placed, err := c.AddMarginOrder(ctx, req)
		if err != nil {
			log.Fatal(err)
		}
		for _, o := range placed.PlacedOrders() {
			log.Printf("placed id=%d clientOrderId=%s", o.ID, o.ClientOrderIDValue())
		}
	} else {
		log.Printf("dry-run place: execution=%s %s %s/%s amount=%s price=%s clientOrderId=%s",
			req.Execution, req.Type, req.SrcCurrency, req.DstCurrency, req.Amount, req.Price, req.ClientOrderID)
		log.Printf("set NOBITEX_EXAMPLE_TRADE=1 to POST /margin/orders/add (TRADE, real funds)")
	}

	positions, err := c.ListPositions(ctx, types.PositionListQuery{
		SrcCurrency: "btc",
		Status:      types.PositionListActive,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("open positions=%d", len(positions.Positions))

	listed, err := c.ListMarginOrders(ctx, types.OrderListQuery{
		Status:  types.OrderListStatusOpen,
		Details: types.OrderListDetailsFull,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, o := range listed.Orders {
		log.Printf("margin order id=%d clientOrderId=%s", o.ID, o.ClientOrderIDValue())
	}

	if os.Getenv("NOBITEX_EXAMPLE_TRADE") == "1" && len(listed.Orders) > 0 {
		canceled, err := c.CancelOrderByID(ctx, listed.Orders[0].ID)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("canceled id=%d status=%s", canceled.Order.ID, canceled.UpdatedStatus)
	}
}
