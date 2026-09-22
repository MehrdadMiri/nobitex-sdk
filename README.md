# nobitex-sdk

Typed Go client for the [Nobitex API](https://apidocs.nobitex.ir) (`https://apiv2.nobitex.ir`).

Repository: https://github.com/MehrdadMiri/nobitex-sdk

This module is the typed Go client for Tradex services: shared HTTP core (Token and API-key auth, `TraderBot/<name>-<version>` User-Agent, typed errors) plus **order book v3**, **system options / market precisions**, and **margin order placement**. Positions and cancel land in follow-up tickets.

## Status

| Area | Status |
|------|--------|
| Go module + packages (`client`, `auth`, `errors`, `types`) | Done |
| Base URL override (default `https://apiv2.nobitex.ir`) | Done |
| Token header + Ed25519 API-key signing | Done |
| User-Agent `TraderBot/<name>-<version>` on every request | Done |
| Typed transport + API errors (`status=failed`, HTTP 4xx/5xx, `backOff`) | Done |
| Env/config secrets (nothing hardcoded) | Done |
| Order book v3 (`GET /v3/orderbook/:symbol`, including `all`) | Done |
| System options / market precisions (`GET /v2/options`) | Done |
| Place margin order (`POST /margin/orders/add`) | Done |
| Positions / cancel | **Out of scope** — follow-up PRs |

## Install

```bash
go get github.com/MehrdadMiri/nobitex-sdk
```

Go 1.22+.

## Quick start

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func main() {
	c, err := client.New(
		client.WithApp("MyBot", "1.0.0"), // User-Agent: TraderBot/MyBot-1.0.0
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Public: no token. Specific market (asks/bids are [price, amount] strings).
	book, err := c.OrderBook(ctx, "BTCIRT")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("BTCIRT last=%s bids=%d asks=%d", book.LastTradePrice, len(book.Bids), len(book.Asks))
	if len(book.Bids) > 0 {
		log.Printf("best bid %s x %s", book.Bids[0].Price, book.Bids[0].Amount)
	}

	// Public: consolidated books for every market.
	all, err := c.OrderBookAll(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if btc, ok := all.Book("BTCIRT"); ok {
		log.Printf("all-markets BTCIRT last=%s", btc.LastTradePrice)
	}

	// Public: market amount/price steps for later order decimal validation.
	opts, err := c.SystemOptions(ctx)
	if err != nil {
		log.Fatal(err)
	}
	amount, _ := opts.AmountPrecision("BTCIRT")
	price, _ := opts.PricePrecision("BTCIRT")
	log.Printf("BTCIRT amount step=%s price step=%s", amount, price)
	if err := opts.ValidateOrderDecimals("BTCIRT", "0.001", "35650565900"); err != nil {
		log.Fatal(err)
	}

	// Authenticated: Token header (or API-key TRADE signing). Load secrets from env.
	authed, err := client.New(
		client.WithApp("MyBot", "1.0.0"),
		client.WithToken(os.Getenv("NOBITEX_TOKEN")),
	)
	if err != nil {
		log.Fatal(err)
	}
	placed, err := authed.AddMarginOrder(ctx, types.NewMarginLimitOrder(
		types.OrderSideSell, "btc", "usdt", "0.01", "13400000000",
	))
	if err != nil {
		log.Fatal(err)
	}
	for _, o := range placed.PlacedOrders() {
		log.Printf("margin order id=%d clientOrderId=%s", o.ID, o.ClientOrderIDValue())
	}
}
```

Construct without the environment:

```go
c, err := client.New(
	client.WithBaseURL("https://apiv2.nobitex.ir"), // default if omitted
	client.WithApp("tradex", "0.1.0"),
	client.WithToken(os.Getenv("NOBITEX_TOKEN")),
)
```

## Secrets (env / config only)

**Never commit tokens or keys.** Load them from the process environment, a secret manager, or your service config. A template is in [`.env.example`](.env.example).

| Variable | Purpose |
|----------|---------|
| `NOBITEX_BASE_URL` | Optional API root; default `https://apiv2.nobitex.ir` |
| `NOBITEX_APP_NAME` | User-Agent name segment |
| `NOBITEX_APP_VERSION` | User-Agent version segment |
| `NOBITEX_TOKEN` | Session token → `Authorization: Token …` |
| `NOBITEX_API_KEY` | Public API key → `Nobitex-Key` |
| `NOBITEX_API_SECRET` | URL-safe Base64 Ed25519 private key |

`client.NewFromEnv` prefers API-key signing when **both** key and secret are set; otherwise it uses `NOBITEX_TOKEN`. Public calls work with no credentials.

## Auth

Official docs: [authentication](https://apidocs.nobitex.ir) · [API key guide](https://apidocs.nobitex.ir/api_key/api-key-guide)

1. **Token** — `Authorization: Token <token>`. Use `client.WithToken` or `auth.NewTokenAuth`.
2. **API key** — headers `Nobitex-Key`, `Nobitex-Signature`, `Nobitex-Timestamp`. Payload is `timestamp + METHOD + full_path + raw_body`, signed with Ed25519 (URL-safe Base64). Use `client.WithAPIKey` or `auth.NewAPIKeyAuth`.

Keys that place or cancel orders need the **TRADE** permission. Timestamp must stay within ~30s of server time in production.

## User-Agent

Nobitex asks bots to send `TraderBot/<name-and-version>` on every call ([general notes](https://apidocs.nobitex.ir/general_notes)). This client always overwrites `User-Agent` to `TraderBot/<name>-<version>` (example: `TraderBot/MyBot-1.0.0`). Configure name/version with `WithApp`, `WithAppName` / `WithAppVersion`, or `NOBITEX_APP_*`.

## Errors

`client.DoJSON` maps:

- network / HTTP-client failures → `errors.KindTransport`
- HTTP ≥ 400 **or** JSON `{"status":"failed",...}` (including HTTP 200) → `errors.KindAPI`
- JSON decode of a successful response → `errors.KindDecode`

Rate-limit bodies may include `code=TooManyRequests`, `backOff` (seconds), and `limit`. Use `errors.As` / `errors.IsAPI` / `errors.IsRateLimited` rather than string matching.

```go
import sdkerr "github.com/MehrdadMiri/nobitex-sdk/errors"

if sdkerr.IsRateLimited(err) {
    var e *sdkerr.Error
    sdkerr.As(err, &e)
    time.Sleep(time.Duration(e.BackOff) * time.Second)
}
```

## Order book v3

Public `GET /v3/orderbook/:symbol` (no token, 300 req/min). Docs: https://apidocs.nobitex.ir

- `Client.OrderBook(ctx, "BTCIRT")` — one market. Asks/bids parse as `[price, amount]` strings (`types.PriceLevel`).
- `Client.OrderBookAll(ctx)` — `symbol=all`; books keyed by market (`all.Book("BTCIRT")`).
- Calls reuse the shared client (User-Agent on every request, `WithoutAuth`, typed errors). Invalid symbols map to `errors.KindAPI` (`InvalidSymbol`).

Market symbols are uppercased; `all` stays lowercase (the API rejects `ALL`).

## System options / market precisions

Public `GET /v2/options` (no token). Docs: https://apidocs.nobitex.ir/options/get-system-options

- `Client.SystemOptions(ctx)` reuses the shared HTTP client (`User-Agent`, typed errors, `WithoutAuth`).
- Response exposes `nobitex.amountPrecisions` and `nobitex.pricePrecisions` (smallest allowed amount / price increment per market, e.g. BTCIRT).
- Helpers for later order-placement code: `AmountPrecision` / `PricePrecision`, `ValidateAmount` / `ValidatePrice` / `ValidateOrderDecimals`, and `Money.FitsStep` / `TruncateToStep`.

## Place margin order

Authenticated `POST /margin/orders/add` (Token header, or API-key with **TRADE**). Docs: https://apidocs.nobitex.ir · https://apidocs.nobitex.ir/margin_trade/%D8%AF%D8%B1%D8%AC-%D8%B3%D9%81%D8%A7%D8%B1%D8%B4-%D8%AA%D8%B9%D9%87%D8%AF%DB%8C

- `Client.AddMarginOrder(ctx, req)` — reuses the shared client (User-Agent, Token / API-key signing, typed errors). Fails client-side if no authenticator is configured.
- Request executions: `limit`, `market`, `stop_limit`, `stop_market`, and `oco`. OCO is sent as `execution=limit` + `mode=oco` (or set `Execution: types.ExecutionOCO` and `Prepare` rewrites it).
- Helpers: `types.NewMarginLimitOrder`, `NewMarginMarketOrder`, `NewMarginStopLimitOrder`, `NewMarginStopMarketOrder`, `NewMarginOCOOrder`. Optional `Leverage` and `ClientOrderID` can be set on the struct.
- Response identifiers for later cancel/list: `order.id` / `order.clientOrderId` (single), or both legs plus `pairId` for OCO (`resp.PlacedOrders()`, `resp.OrderIDs()`).
- Rate limit: 300 requests / 10 minutes, shared with spot placement.

```go
req := types.NewMarginStopLimitOrder(types.OrderSideSell, "btc", "usdt", "0.01", "12500000000", "12600000000")
req.Leverage = "2"
req.ClientOrderID = "my-order-123"
resp, err := c.AddMarginOrder(ctx, req)
```

## Package layout

```
github.com/MehrdadMiri/nobitex-sdk
├── client/   HTTP core, Do / DoJSON, OrderBook / OrderBookAll, SystemOptions, AddMarginOrder
├── auth/     Token header + API-key Ed25519 signer; env loader
├── errors/   Typed transport + API error model
└── types/    Envelope / status / money / order-book / system options / decimal-step / margin-order shapes
```

Stdlib only (no third-party dependencies).

## How to extend (follow-up endpoint tickets)

1. Add request/response structs (new file or small domain package; money as `types.Money` / `string`, not `float64`).
2. Add a method on `*client.Client` that calls `DoJSON` with the documented path and `WithJSONBody` / `WithQuery`.
3. Use `WithoutAuth()` only for documented public routes (order book and system options already do this). Keep auth on for user/margin/position calls.
4. Rely on existing error mapping — do not treat HTTP 200 as success without checking `status`.
5. Cover the method with `httptest` + a recorded fixture; do **not** commit credentials.

## P0 endpoints

1. `GET /v3/orderbook/:symbol` — including `symbol=all` (**done**)
2. `POST /margin/orders/add` — limit / market / stop_limit / stop_market / oco (**done**)
3. `GET /positions/list` — next
4. `POST /positions/:positionId/close` — next
5. `GET`/`POST /market/orders/list` (margin filter) — next
6. `POST /market/orders/update-status` (cancel) — next
7. `GET /v2/options` (`amountPrecisions`, `pricePrecisions`) (**done**)

## Tests

```bash
go test ./...
```

Unit tests use `httptest` and recorded JSON fixtures. `TestOrderBookLivePublic` and `TestSystemOptionsLivePublic` optionally hit live public APIs (skipped with `-short`, and skipped if the network is down). Margin-order tests are fixture-only — **no authenticated live calls** and no real funds. No secrets in git.

## Non-goals

- Trading bot / strategy engine
- Storing secrets in git
- Positions / cancel in this PR

## License / ownership

Tradex internal SDK under Mehrdad Miri / Tradex PM workstream.
