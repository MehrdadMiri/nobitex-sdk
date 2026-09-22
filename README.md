# nobitex-sdk

Typed Go client for the [Nobitex API](https://apidocs.nobitex.ir) (`https://apiv2.nobitex.ir`).

Repository: https://github.com/MehrdadMiri/nobitex-sdk

This module is the **HTTP foundation** for Tradex services: shared client construction, Token and API-key auth, `TraderBot/<name>-<version>` User-Agent, and a typed error model. **P0 trading endpoints are not in this PR** — order book, margin orders, positions, cancel, and system options land in follow-up tickets.

## Status

| Area | This PR |
|------|---------|
| Go module + packages (`client`, `auth`, `errors`, `types`) | Done |
| Base URL override (default `https://apiv2.nobitex.ir`) | Done |
| Token header + Ed25519 API-key signing | Done |
| User-Agent `TraderBot/<name>-<version>` on every request | Done |
| Typed transport + API errors (`status=failed`, HTTP 4xx/5xx, `backOff`) | Done |
| Env/config secrets (nothing hardcoded) | Done |
| Order book / margin / positions / cancel / options methods | **Out of scope** — follow-up PRs |

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
	"net/http"

	"github.com/MehrdadMiri/nobitex-sdk/client"
)

func main() {
	c, err := client.NewFromEnv(
		client.WithApp("MyBot", "1.0.0"), // User-Agent: TraderBot/MyBot-1.0.0
	)
	if err != nil {
		log.Fatal(err)
	}

	// Endpoint wrappers come in later tickets. Until then, call the HTTP core:
	var dest map[string]any
	err = c.DoJSON(context.Background(), http.MethodGet, "/v2/options", &dest, client.WithoutAuth())
	if err != nil {
		log.Fatal(err)
	}
	_ = dest
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

## Package layout

```
github.com/MehrdadMiri/nobitex-sdk
├── client/   HTTP core, options, Do / DoJSON
├── auth/     Token header + API-key Ed25519 signer; env loader
├── errors/   Typed transport + API error model
└── types/    Shared envelope / status / monetary string
```

Stdlib only (no third-party dependencies).

## How to extend (follow-up endpoint tickets)

1. Add request/response structs (new file or small domain package; money as `types.Money` / `string`, not `float64`).
2. Add a method on `*client.Client` that calls `DoJSON` with the documented path and `WithJSONBody` / `WithQuery`.
3. Use `WithoutAuth()` only for documented public routes (e.g. order book). Keep auth on for user/margin/position calls.
4. Rely on existing error mapping — do not treat HTTP 200 as success without checking `status`.
5. Cover the method with `httptest`; do **not** commit credentials or hit the live API from unit tests.

Example shape for a later ticket (not implemented here):

```go
func (c *Client) OrderBook(ctx context.Context, symbol string) (*OrderBook, error) {
	var out OrderBook
	path := "/v3/orderbook/" + url.PathEscape(symbol)
	err := c.DoJSON(ctx, http.MethodGet, path, &out, WithoutAuth())
	return &out, err
}
```

## P0 endpoints (next PRs)

1. `GET /v3/orderbook/:symbol` — including `symbol=all`
2. `POST /margin/orders/add`
3. `GET /positions/list`
4. `POST /positions/:positionId/close`
5. `GET`/`POST /market/orders/list` (margin filter)
6. `POST /market/orders/update-status` (cancel)
7. `GET /v2/options` (`amountPrecisions`, `pricePrecisions`)

## Tests

```bash
go test ./...
```

Tests use `httptest` and in-memory keys only — no live authenticated calls.

## Non-goals

- Trading bot / strategy engine
- Storing secrets in git
- Full P0 endpoint coverage in this scaffold

## License / ownership

Tradex internal SDK under Mehrdad Miri / Tradex PM workstream.
