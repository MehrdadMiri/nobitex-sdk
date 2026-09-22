package types_test

import (
	"encoding/json"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

// Documented single-market sample (https://apidocs.nobitex.ir / GET /v3/orderbook/:symbol).
const documentedOrderBook = `{
  "status": "ok",
  "lastUpdate": 1644991756704,
  "lastTradePrice": "35650565900",
  "asks": [
    ["1476091000", "1.016"],
    ["1479700000", "0.2561"]
  ],
  "bids": [
    ["1470001120", "0.126571"],
    ["1470000000", "0.818994"]
  ]
}`

const documentedOrderBookAll = `{
  "status": "ok",
  "BTCIRT": {
    "lastUpdate": 1644991756704,
    "lastTradePrice": "35650565900",
    "asks": [
      ["1476091000", "1.016"],
      ["1479700000", "0.2561"]
    ],
    "bids": [
      ["1470001120", "0.126571"],
      ["1470000000", "0.818994"]
    ]
  },
  "USDTIRT": {
    "lastUpdate": 1644991767392,
    "lastTradePrice": "277960",
    "asks": [
      ["277990", "6688.3"],
      ["278000", "28185.03"]
    ],
    "bids": [
      ["277960", "119.31"],
      ["271240", "1079.75"]
    ]
  }
}`

func TestOrderBookParsesDocumentedAsksBids(t *testing.T) {
	t.Parallel()
	var ob types.OrderBook
	if err := json.Unmarshal([]byte(documentedOrderBook), &ob); err != nil {
		t.Fatal(err)
	}
	if !ob.Status.IsOK() {
		t.Fatalf("status = %q", ob.Status)
	}
	if ob.LastTradePrice != "35650565900" {
		t.Fatalf("lastTradePrice = %q", ob.LastTradePrice)
	}
	if ob.LastUpdate == nil || *ob.LastUpdate != 1644991756704 {
		t.Fatalf("lastUpdate = %v", ob.LastUpdate)
	}
	if len(ob.Asks) != 2 || len(ob.Bids) != 2 {
		t.Fatalf("asks=%d bids=%d", len(ob.Asks), len(ob.Bids))
	}
	if ob.Asks[0].Price != "1476091000" || ob.Asks[0].Amount != "1.016" {
		t.Fatalf("asks[0] = %+v", ob.Asks[0])
	}
	if ob.Bids[0].Price != "1470001120" || ob.Bids[0].Amount != "0.126571" {
		t.Fatalf("bids[0] = %+v", ob.Bids[0])
	}

	raw, err := json.Marshal(ob.Asks[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `["1476091000","1.016"]` {
		t.Fatalf("marshal price level = %s", raw)
	}
}

func TestOrderBookAllParsesConsolidatedBooks(t *testing.T) {
	t.Parallel()
	var all types.OrderBookAll
	if err := json.Unmarshal([]byte(documentedOrderBookAll), &all); err != nil {
		t.Fatal(err)
	}
	if !all.Status.IsOK() {
		t.Fatalf("status = %q", all.Status)
	}
	if len(all.Books) != 2 {
		t.Fatalf("books = %d", len(all.Books))
	}
	btc, ok := all.Book("BTCIRT")
	if !ok {
		t.Fatal("missing BTCIRT")
	}
	if btc.Symbol != "BTCIRT" {
		t.Fatalf("symbol = %q", btc.Symbol)
	}
	if btc.LastTradePrice != "35650565900" {
		t.Fatalf("BTC lastTradePrice = %q", btc.LastTradePrice)
	}
	if len(btc.Asks) != 2 || btc.Asks[1].Amount != "0.2561" {
		t.Fatalf("BTC asks = %+v", btc.Asks)
	}
	usdt, ok := all.Book("usdtirt")
	if !ok {
		t.Fatal("Book() should match USDTIRT case-insensitively")
	}
	if usdt.Bids[0].Price != "277960" {
		t.Fatalf("USDT bids[0] = %+v", usdt.Bids[0])
	}
	if _, ok := all.Book("NOPE"); ok {
		t.Fatal("unexpected NOPE book")
	}
}

func TestOrderBookLastUpdateNull(t *testing.T) {
	t.Parallel()
	var ob types.OrderBook
	body := `{"status":"ok","lastUpdate":null,"lastTradePrice":"1","asks":[],"bids":[]}`
	if err := json.Unmarshal([]byte(body), &ob); err != nil {
		t.Fatal(err)
	}
	if ob.LastUpdate != nil {
		t.Fatalf("lastUpdate = %v, want nil", ob.LastUpdate)
	}
}

func TestPriceLevelRejectsWrongArity(t *testing.T) {
	t.Parallel()
	var p types.PriceLevel
	if err := json.Unmarshal([]byte(`["only-price"]`), &p); err == nil {
		t.Fatal("expected error")
	}
	if err := json.Unmarshal([]byte(`["1","2","3"]`), &p); err == nil {
		t.Fatal("expected error")
	}
}
