# nobitex-sdk

Typed Go client for the [Nobitex API](https://apidocs.nobitex.ir) (`https://apiv2.nobitex.ir`).

Repository: https://github.com/MehrdadMiri/nobitex-sdk

This module is the typed Go client for Tradex services: shared HTTP core (Token and API-key auth, `TraderBot/<name>-<version>` User-Agent, typed errors) plus **P0** market/margin fundamentals (including user orders list + cancel) and **P1** (nice-to-have) breadth across remaining practical docs categories.

Official docs: https://apidocs.nobitex.ir · Repository: https://github.com/MehrdadMiri/nobitex-sdk

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
| Positions list (`GET /positions/list`) | Done |
| Close position (`POST /positions/:positionId/close`) | Done |
| User orders list (`GET`/`POST /market/orders/list`, margin filter) | Done |
| Cancel order (`POST /market/orders/update-status`) | Done |
| README usage examples (public orderbook + env-token margin flow) | Done |
| Table-driven unit tests for P0 parsers/clients | Done |
| Compilable `examples/` (orderbook, market, margin, account) | Done |
| **P1** remaining market data (stats, trades, depth, OHLC) | Done |
| **P1** user info (profile, wallets, balance, limitations, deposits) | Done |
| **P1** spot helpers (place, status, user trades) | Done |
| **P1** margin helpers (markets list, leverage/delegation, transfer) | Done |
| **P1** withdrawals **read-only** (list + get) | Done |
| **P1** WebSocket overview stub (URL + channel list, connection token) | Done |

## Install

```bash
go get github.com/MehrdadMiri/nobitex-sdk
```

Go 1.22+.

## Usage

Copy-paste examples below. Secrets come from the environment ([`.env.example`](.env.example)); never commit tokens. Runnable copies:

- [`examples/orderbook`](examples/orderbook) — public P0 order book
- [`examples/market`](examples/market) — public P1 market data + WS overview
- [`examples/margin`](examples/margin) — env-token P0 margin flow (list by default)
- [`examples/account`](examples/account) — env-token P1 **READ** (profile, wallets, withdraws, WS token)

Official docs: https://apidocs.nobitex.ir · Repository: https://github.com/MehrdadMiri/nobitex-sdk

### Public order book (no token)

`GET /v3/orderbook/:symbol` is public (300 req/min). No `NOBITEX_TOKEN` required.

```go
package main

import (
	"context"
	"log"

	"github.com/MehrdadMiri/nobitex-sdk/client"
)

func main() {
	c, err := client.New(
		client.WithApp("MyBot", "1.0.0"), // User-Agent: TraderBot/MyBot-1.0.0
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	book, err := c.OrderBook(ctx, "BTCIRT")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("BTCIRT last=%s bids=%d asks=%d", book.LastTradePrice, len(book.Bids), len(book.Asks))
	if len(book.Bids) > 0 {
		log.Printf("best bid %s x %s", book.Bids[0].Price, book.Bids[0].Amount)
	}

	all, err := c.OrderBookAll(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if btc, ok := all.Book("BTCIRT"); ok {
		log.Printf("all-markets BTCIRT last=%s", btc.LastTradePrice)
	}
}
```

```bash
go run ./examples/orderbook
```

### P1 public market data (no token)

Remaining market data from PR #7: stats, trades, depth, OHLC, plus the WebSocket overview stub (no subscriber). `MarginMarkets` also works without a token.

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func main() {
	c, err := client.New(client.WithApp("MyBot", "1.0.0"))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	stats, err := c.MarketStats(ctx, types.MarketStatsQuery{SrcCurrency: "btc", DstCurrency: "rls"})
	if err != nil {
		log.Fatal(err)
	}
	if st, ok := stats.Stat("BTCIRT"); ok {
		log.Printf("latest=%s change=%v", types.MoneyValue(st.Latest), st.DayChange)
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
	log.Printf("depth bids=%d asks=%d", len(depth.Bids), len(depth.Asks))

	ohlc, err := c.MarketOHLC(ctx, types.UDFHistoryQuery{
		Symbol: "BTCIRT", Resolution: types.Resolution60m,
		To: time.Now().Unix(), Countback: 3,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ohlc bars=%d", len(ohlc.Candles()))

	ov := client.WebSocketOverview()
	log.Printf("ws=%s channel=%s", ov.ProductionURL, types.PublicOrderBookChannel("BTCIRT"))
}
```

```bash
go run ./examples/market
```

P1 **TRADE** helpers (`AddSpotOrder`, `TransferWallet`) are not run in examples. Use the same env-token constructor as margin; they move funds.

### Authenticated P1 READ (env token)

API-key **READ** (or session token). No orders, no transfers, no withdraw submit.

```go
package main

import (
	"context"
	"log"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func main() {
	c, err := client.NewFromEnv(client.WithApp("MyBot", "1.0.0"))
	if err != nil {
		log.Fatal(err)
	}
	if c.Auth() == nil {
		log.Fatal("set NOBITEX_TOKEN or NOBITEX_API_KEY+NOBITEX_API_SECRET")
	}
	ctx := context.Background()

	profile, err := c.UserProfile(ctx)
	if err != nil {
		log.Fatal(err)
	}
	wallets, err := c.ListWallets(ctx, types.WalletListRequest{Type: types.WalletSpot})
	if err != nil {
		log.Fatal(err)
	}
	withdraws, err := c.ListWithdraws(ctx, types.WithdrawListQuery{PageSize: 10})
	if err != nil {
		log.Fatal(err)
	}
	tok, err := c.WebSocketToken(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := tok.ValidateToken(); err != nil {
		log.Fatal(err)
	}
	log.Printf("nick=%s wallets=%d withdraws=%d private=%s",
		profile.Profile.DisplayNickname(), len(wallets.Wallets), len(withdraws.Withdraws),
		types.PrivateOrdersChannel(profile.Profile.WebsocketAuthParam))
}
```

```bash
export NOBITEX_TOKEN=  # paste locally; never commit
go run ./examples/account
```

### Authenticated margin flow (env token)

Load `NOBITEX_TOKEN` (or `NOBITEX_API_KEY` + `NOBITEX_API_SECRET` with **TRADE**). Placeholders only — never hardcode secrets. `client.NewFromEnv` is the supported constructor.

```go
package main

import (
	"context"
	"log"

	"github.com/MehrdadMiri/nobitex-sdk/client"
	"github.com/MehrdadMiri/nobitex-sdk/types"
)

func main() {
	c, err := client.NewFromEnv(client.WithApp("MyBot", "1.0.0"))
	if err != nil {
		log.Fatal(err)
	}
	if c.Auth() == nil {
		log.Fatal("set NOBITEX_TOKEN or NOBITEX_API_KEY+NOBITEX_API_SECRET")
	}

	ctx := context.Background()

	placed, err := c.AddMarginOrder(ctx, types.NewMarginLimitOrder(
		types.OrderSideSell, "btc", "usdt", "0.01", "13400000000",
	))
	if err != nil {
		log.Fatal(err)
	}
	for _, o := range placed.PlacedOrders() {
		log.Printf("placed id=%d clientOrderId=%s", o.ID, o.ClientOrderIDValue())
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

	if len(listed.Orders) > 0 {
		canceled, err := c.CancelOrderByID(ctx, listed.Orders[0].ID)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("canceled id=%d status=%s", canceled.Order.ID, canceled.UpdatedStatus)
	}
}
```

```bash
export NOBITEX_TOKEN=  # paste locally; never commit
go run ./examples/margin                 # lists only (no funds)
NOBITEX_EXAMPLE_TRADE=1 go run ./examples/margin  # place + cancel (real funds)
```

Construct without `NewFromEnv`:

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

API-key scopes (see [API key guide](https://apidocs.nobitex.ir/api_key/api-key-guide)):

| Call | API-key permission |
|------|--------------------|
| `GET`/`POST /market/orders/list`, order status, user trades, profile/wallets/deposits/withdraws, WS token | **READ** |
| `POST /market/orders/update-status` (cancel) | **TRADE** |
| `POST /margin/orders/add`, `POST /market/orders/add` | **TRADE** |
| `GET /positions/list`, `POST /positions/:id/close` | **TRADE** |
| `POST /wallets/transfer` (spot ↔ margin) | **TRADE** |

Keys that place, close, or **cancel** orders need the **TRADE** permission. Timestamp must stay within ~30s of server time in production. The client always sends `User-Agent` and applies Token (`Authorization: Token …`) and/or API-key signing (`Nobitex-Key` / `Nobitex-Signature` / `Nobitex-Timestamp`) from the shared HTTP core. Missing credentials fail before any network call on authenticated endpoints.

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

## Positions list + close

Authenticated (Token and/or API-key with **TRADE**). Docs: https://apidocs.nobitex.ir

- `Client.ListPositions(ctx, types.PositionListQuery{...})` — `GET /positions/list`. Query filters: `srcCurrency`, `dstCurrency`, `status` (`active` | `past`; API default `active`), `page`, `pageSize` (default 50). Each row includes the position `id` and `status` (`Open` / `Closed` / `Liquidated` / `Expired`) Tradex needs, plus entry/exit prices, liability, and PNL fields.
- `Client.ClosePosition(ctx, positionID, req)` — `POST /positions/:positionId/close`. Opposite-side close (sell position → buy order, buy position → sell order). Executions: `limit` / `market` / `stop_limit` / `stop_market` / `oco` (OCO is sent as documented `execution=limit` + `mode=oco`). Helpers: `NewCloseLimitOrder`, `NewCloseMarketOrder`, `NewCloseStopLimitOrder`, `NewCloseStopMarketOrder`, `NewCloseOCOOrder`.
- Both methods reuse the shared HTTP client (User-Agent, Token / API-key signing, typed errors). Missing credentials fail before any network call.

```go
list, err := c.ListPositions(ctx, types.PositionListQuery{
	SrcCurrency: "btc",
	Status:      types.PositionListActive,
})
closeResp, err := c.ClosePosition(ctx, 128, types.NewCloseLimitOrder("0.0100150225", "6200000000"))
```

## User orders list + cancel

Authenticated. Docs: https://apidocs.nobitex.ir · https://github.com/MehrdadMiri/nobitex-sdk

- `Client.ListOrders(ctx, query)` — documented `GET /market/orders/list` (API-key **READ**, 30/min). Filters: `status` (`all`/`open`/`done`/`close`), `type`, `execution`, **`tradeType` (`spot`/`margin`)**, `srcCurrency`, `dstCurrency`, `details`, `fromId`, `order` (sort), `page`, `pageSize`. `page` and `fromId` cannot be combined.
- `Client.ListMarginOrders(ctx, query)` — same GET with `tradeType=margin` (the P0 margin filter). Other query fields are kept.
- `Client.ListOrdersPost` / `ListMarginOrdersPost` — POST the same filters as a JSON body (GET is the documented method).
- Set `Details: types.OrderListDetailsFull` (`2`) so each row includes `id` / `status` / `fee` / `created_at` / `averagePrice` for later cancel.
- `Client.CancelOrder(ctx, req)` — `POST /market/orders/update-status` with `status=canceled`. API-key **TRADE** (90/min). At least one of `order` (server id) or `clientOrderId` is required; if both are sent, `order` wins. Search by `clientOrderId` only covers open orders (`New` / `Active` / `Inactive`).
- Helpers: `CancelOrderByID`, `CancelOrderByClientOrderID`, `types.NewCancelOrderByID`, `types.NewCancelOrderByClientOrderID`, `types.NewMarginOrderListQuery`.
- All four list/cancel methods reuse the shared HTTP client (`User-Agent: TraderBot/<name>-<version>`, Token and/or API-key signing, typed errors). No authenticator → error, no HTTP call.

```go
listed, err := c.ListMarginOrders(ctx, types.OrderListQuery{
	Status:  types.OrderListStatusOpen,
	Details: types.OrderListDetailsFull,
})
_, err = c.CancelOrderByID(ctx, listed.Orders[0].ID)
_, err = c.CancelOrderByClientOrderID(ctx, "my-order-123")
```

## P1 — nice-to-have breadth

Typed methods matching remaining official docs categories ([apidocs.nobitex.ir](https://apidocs.nobitex.ir)). Breadth over strategy logic. This is **not** a trading bot.

### Remaining market data (public)

- `Client.MarketStats` — `GET /market/stats` (optional `srcCurrency` / `dstCurrency`; 20/min). Lookup with `resp.Stat("BTCIRT")` (maps to `btc-rls`).
- `Client.MarketTrades` — `GET /v2/trades/:symbol` (max 20; `all` unsupported; 60/min).
- `Client.MarketDepth` — `GET /v2/depth/:symbol` (depth-chart levels; 300/min).
- `Client.MarketOHLC` — `GET /market/udf/history` (TradingView UDF; max 500 candles; `h.Candles()`). `s=error` maps to a typed API error; `s=no_data` is success with no bars.

### User info (auth, API-key **READ**)

- `Client.UserProfile` — `GET /users/profile` (`profile.websocketAuthParam` for private WS names).
- `Client.UserLimitations` — `POST /users/limitations`.
- `Client.ListWallets` — `POST /users/wallets/list` (`type=spot|margin|credit|debit`).
- `Client.ListWalletsV2` — `POST /v2/wallets` (selected currencies).
- `Client.WalletBalance` — `POST /users/wallets/balance`.
- `Client.ListDeposits` — `GET /users/wallets/deposits/list`.

### Spot helpers (not list/cancel)

- `Client.AddSpotOrder` — `POST /market/orders/add` (**TRADE**; limit / market / stop_limit / stop_market / oco). Helpers: `NewSpotLimitOrder`, `NewSpotMarketOrder`, `NewSpotStopLimitOrder`, `NewSpotStopMarketOrder`, `NewSpotOCOOrder`.
- `Client.SpotOrderStatus` — `POST /market/orders/status` (**READ**; id or `clientOrderId`).
- `Client.ListUserTrades` — `GET /market/trades/list` (**READ**; last 180 days). `srcCurrency`/`dstCurrency` must both be set or both empty.

### Margin helpers

- `Client.MarginMarkets` — `GET /margin/markets/list` (public without a token; a configured authenticator is sent so `maxLeverage` / delegation caps can personalize; `details=true` sends the documented JSON body).
- `Client.MarginDelegationLimit` — `GET /margin/v2/delegation-limit?market=` (**READ**; remaining buy/sell capacity per leverage). `resp.LimitFor(side, leverage)`.
- `Client.TransferWallet` / `TransferSpotToMargin` / `TransferMarginToSpot` — `POST /wallets/transfer` (spot ↔ margin).

### Withdrawals (read-only)

- `Client.ListWithdraws` — `GET /users/wallets/withdraws/list` (**READ**).
- `Client.GetWithdraw` — `GET /withdraws/:withdrawId` (**READ**). Submit / confirm / cancel are out of scope.

### WebSocket overview stub

Not a subscriber. Docs: https://apidocs.nobitex.ir/websocket/%D9%88%D8%A8-%D8%B3%D9%88%DA%A9%D8%AA

- `client.WebSocketOverview()` — production `wss://ws.nobitex.ir/connection/websocket`, testnet URL, token path, and public/private channel patterns.
- Channel helpers: `types.PublicOrderBookChannel`, `PublicTradesChannel`, `PublicCandleChannel`, `PublicMarketStatsChannel`, `PrivateOrdersChannel`, `PrivateTradesChannel`.
- `Client.WebSocketToken` — `GET /auth/ws/token/` (**READ**; JWT, 1200s). Combine with `profile.websocketAuthParam` as `private:{name}#{param}`.

```go
stats, _ := c.MarketStats(ctx, types.MarketStatsQuery{SrcCurrency: "btc", DstCurrency: "rls"})
if st, ok := stats.Stat("BTCIRT"); ok {
	log.Printf("latest=%s change=%v", types.MoneyValue(st.Latest), st.DayChange)
}
ov := client.WebSocketOverview()
log.Printf("ws=%s orderbook=%s", ov.ProductionURL, types.PublicOrderBookChannel("BTCIRT"))
```

## Package layout

```
github.com/MehrdadMiri/nobitex-sdk
├── client/     HTTP core, Do / DoJSON, P0 endpoints, P1 market/user/spot/margin-helper/withdraw/WS-token methods
├── auth/       Token header + API-key Ed25519 signer; env loader
├── errors/     Typed transport + API error model
├── types/      Envelope / money / P0 + P1 request/response shapes / WS channel helpers
└── examples/   Compilable usage: public orderbook + market data, env-token margin, P1 READ account
```

Stdlib only (no third-party dependencies).

## How to extend (follow-up endpoint tickets)

1. Add request/response structs (new file or small domain package; money as `types.Money` / `string`, not `float64`).
2. Add a method on `*client.Client` that calls `DoJSON` with the documented path and `WithJSONBody` / `WithQuery`.
3. Use `WithoutAuth()` only for documented public routes (order book, system options, remaining market data). Keep auth on for user/margin/position/withdraw calls. `MarginMarkets` sends auth when configured so the payload can personalize.
4. Rely on existing error mapping — do not treat HTTP 200 as success without checking `status`.
5. Cover the method with `httptest` + a recorded fixture; do **not** commit credentials.

## P0 endpoints

1. `GET /v3/orderbook/:symbol` — including `symbol=all` (**done**)
2. `POST /margin/orders/add` — limit / market / stop_limit / stop_market / oco (**done**)
3. `GET /positions/list` (**done**)
4. `POST /positions/:positionId/close` (**done**)
5. `GET`/`POST /market/orders/list` (margin filter) (**done**)
6. `POST /market/orders/update-status` (cancel by id or `clientOrderId`, TRADE) (**done**)
7. `GET /v2/options` (`amountPrecisions`, `pricePrecisions`) (**done**)

## P1 endpoints (nice-to-have)

Covered in this module (typed methods + fixture tests): remaining market data, user info, spot helpers, margin markets/leverage/transfer, withdrawals read-only, WebSocket overview stub. See the P1 section above. Docs: https://apidocs.nobitex.ir · repo: https://github.com/MehrdadMiri/nobitex-sdk

## Tests

```bash
go test ./...
go test -short ./...          # skip optional live public calls
go test -race -short ./...
```

Unit tests are table-driven where it helps (P0 parsers, query encoding, auth-required methods, `errors.Check`). They use `httptest` and recorded JSON fixtures. `go test` also `go build`s [`examples/orderbook`](examples/orderbook), [`examples/margin`](examples/margin), [`examples/market`](examples/market), and [`examples/account`](examples/account) — those programs are not executed against the live API in tests.

`TestOrderBookLivePublic`, `TestSystemOptionsLivePublic`, and `TestMarketDataLivePublic` optionally hit live **public** APIs (skipped with `-short`, and skipped if the network is down). Authenticated P0/P1 methods (margin-order, positions, user-orders list/cancel, spot, user, withdraw, WS token) are fixture-only — **no authenticated live calls** in CI and no real funds. `examples/margin` lists by default; place/cancel require `NOBITEX_EXAMPLE_TRADE=1`. No secrets in git.

## Non-goals

- Trading bot / strategy engine / full WebSocket subscriber
- Storing secrets in git
- Withdraw submit / confirm / cancel (P1 is read-only)

## License / ownership

Tradex internal SDK under Mehrdad Miri / Tradex PM workstream.
