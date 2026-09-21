# nobitex-sdk

Nobitex exchange connection SDK for **Tradex**.

Public scaffold for a typed Go client against the [Nobitex API](https://apidocs.nobitex.ir) (`https://apiv2.nobitex.ir`).

> **Status:** empty scaffold only. Implementation is owned by Tradex DE via agent-kanban tickets. Do not treat this README as a finished SDK.

## Language

**Go** — aligned with existing Tradex/Nobitex codebases (`autotradex`, `TradeX`, `nobitex-engine`, `nobitex-datagatherer`, `NovaTrade`).

## Purpose

Provide a reusable, well-typed Nobitex HTTP (and later WebSocket) client so Tradex trading services stop embedding ad-hoc API calls. First workstream prioritizes margin (تعهدی) trading and market fundamentals.

## Base URL & conventions

| Item | Value |
|------|--------|
| Docs | https://apidocs.nobitex.ir |
| Base URL | `https://apiv2.nobitex.ir` |
| User-Agent | `TraderBot/<name-and-version>` recommended on all calls |
| Auth | Token header and/or API-key signing (TRADE permission where noted) |

## P0 endpoints (must ship first)

1. `GET /v3/orderbook/:symbol` — Order book v3; support `symbol=all` and specific markets. Public. Rate: 300/min.
2. `POST /margin/orders/add` — Place margin order (limit / market / stop_limit / stop_market / oco). Auth Token; TRADE if API key.
3. `GET /positions/list` — List positions; filters e.g. `srcCurrency`, `status`. Auth.
4. `POST /positions/:positionId/close` — Close position (opposite-side); execution types as docs. Auth.
5. `GET|POST /market/orders/list` — User orders list with **margin** trade-type filter. Auth.
6. `POST /market/orders/update-status` — Cancel order (`status=canceled`) by order id or `clientOrderId`. Auth TRADE.
7. `GET /v2/options` — System settings; parse `amountPrecisions` and `pricePrecisions`.

## P1 (nice-to-have)

Broader typed coverage: remaining market data, user info, spot trade, margin helpers (markets list, leverage, margin transfer), read-only withdrawals, WebSocket overview stub.

## Non-goals (this repo scaffold)

- Full trading bot / strategy engine
- Storing secrets or tokens in git
- Implementing endpoints before DE tickets land

## License / ownership

Tradex internal SDK under Mehrdad Miri / Tradex PM workstream.