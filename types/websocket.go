package types

import (
	"fmt"
	"strings"
	"time"
)

const (
	// DefaultWebSocketURL is the production Centrifugo endpoint.
	DefaultWebSocketURL = "wss://ws.nobitex.ir/connection/websocket"
	// TestnetWebSocketURL is the documented testnet Centrifugo endpoint.
	TestnetWebSocketURL = "wss://testnetws.nobitex.ir/connection/websocket"
	// WebSocketTokenTTL is the documented connection-token lifetime.
	WebSocketTokenTTL = 1200 * time.Second
	// WebSocketTokenPath is GET /auth/ws/token/ (trailing slash is significant).
	WebSocketTokenPath = "/auth/ws/token/"

	wsPublicPrefix  = "public:"
	wsPrivatePrefix = "private:"
)

// Documented channel name fragments (not a live subscriber).
const (
	WSChannelOrderBook   = "orderbook"
	WSChannelCandle      = "candle"
	WSChannelTrades      = "trades"
	WSChannelMarketStats = "market-stats"
	WSChannelOrders      = "orders"
	WSChannelUserTrades  = "trades"
	WSMarketStatsAll     = "all"
)

// WebSocketOverview is a stub: connection URLs + channel patterns. This SDK
// does not implement a trading bot or Centrifugo subscriber.
type WebSocketOverview struct {
	ProductionURL      string        `json:"productionUrl"`
	TestnetURL         string        `json:"testnetUrl"`
	TokenPath          string        `json:"tokenPath"`
	TokenTTL           time.Duration `json:"tokenTtl"`
	PublicChannels     []string      `json:"publicChannels"`
	PrivateChannels    []string      `json:"privateChannels"`
	PrivateNamePattern string        `json:"privateNamePattern"`
}

// DefaultWebSocketOverview returns the documented WS overview stub.
func DefaultWebSocketOverview() WebSocketOverview {
	return WebSocketOverview{
		ProductionURL: DefaultWebSocketURL,
		TestnetURL:    TestnetWebSocketURL,
		TokenPath:     WebSocketTokenPath,
		TokenTTL:      WebSocketTokenTTL,
		PublicChannels: []string{
			"public:orderbook-{MARKET_SYMBOL}",
			"public:candle-{MARKET_SYMBOL}-{RESOLUTION}",
			"public:trades-{MARKET_SYMBOL}",
			"public:market-stats-{MARKET_SYMBOL}",
			"public:market-stats-all",
		},
		PrivateChannels: []string{
			"private:orders#{websocketAuthParam}",
			"private:trades#{websocketAuthParam}",
		},
		PrivateNamePattern: "private:{channelName}#{websocketAuthParam}",
	}
}

// PublicOrderBookChannel is public:orderbook-BTCIRT.
func PublicOrderBookChannel(symbol string) string {
	return wsPublicPrefix + WSChannelOrderBook + "-" + strings.ToUpper(strings.TrimSpace(symbol))
}

// PublicTradesChannel is public:trades-BTCIRT.
func PublicTradesChannel(symbol string) string {
	return wsPublicPrefix + WSChannelTrades + "-" + strings.ToUpper(strings.TrimSpace(symbol))
}

// PublicCandleChannel is public:candle-BTCIRT-15.
func PublicCandleChannel(symbol, resolution string) string {
	return wsPublicPrefix + WSChannelCandle + "-" + strings.ToUpper(strings.TrimSpace(symbol)) + "-" + strings.TrimSpace(resolution)
}

// PublicMarketStatsChannel is public:market-stats-BTCIRT, or public:market-stats-all.
func PublicMarketStatsChannel(symbol string) string {
	s := strings.TrimSpace(symbol)
	if s == "" || strings.EqualFold(s, WSMarketStatsAll) {
		return wsPublicPrefix + WSChannelMarketStats + "-" + WSMarketStatsAll
	}
	return wsPublicPrefix + WSChannelMarketStats + "-" + strings.ToUpper(s)
}

// PrivateChannel is private:{name}#{websocketAuthParam}.
func PrivateChannel(name, websocketAuthParam string) string {
	return wsPrivatePrefix + strings.TrimSpace(name) + "#" + strings.TrimSpace(websocketAuthParam)
}

// PrivateOrdersChannel is private:orders#{websocketAuthParam}.
func PrivateOrdersChannel(websocketAuthParam string) string {
	return PrivateChannel(WSChannelOrders, websocketAuthParam)
}

// PrivateTradesChannel is private:trades#{websocketAuthParam}.
func PrivateTradesChannel(websocketAuthParam string) string {
	return PrivateChannel(WSChannelUserTrades, websocketAuthParam)
}

// WebSocketTokenResponse is GET /auth/ws/token/. The JWT is for the Centrifugo
// connection only (not websocketAuthParam).
type WebSocketTokenResponse struct {
	Envelope
	Token string `json:"token"`
}

// ValidateToken reports whether a connection token is present.
func (r *WebSocketTokenResponse) ValidateToken() error {
	if r == nil || strings.TrimSpace(r.Token) == "" {
		return fmt.Errorf("types: websocket connection token is empty")
	}
	return nil
}
