package types_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/MehrdadMiri/nobitex-sdk/types"
)

// Documented sample from https://apidocs.nobitex.ir/openapi/options.yaml
const documentedSystemOptions = `{
  "status": "ok",
  "features": {
    "fcmEnabled": true,
    "chat": "enabled",
    "walletsToNet": true,
    "autoKYC": true,
    "enabledFeatures": ["feature1", "feature2"],
    "betaFeatures": ["beta1"]
  },
  "coins": [
    {
      "coin": "rls",
      "name": "Rial",
      "defaultNetwork": "FIAT_MONEY",
      "displayPrecision": "10",
      "stdName": "﷼",
      "networkList": {
        "FIAT_MONEY": {
          "network": "FIAT_MONEY",
          "name": "FIAT",
          "isDefault": true,
          "beta": false,
          "depositEnable": true,
          "minConfirm": 0,
          "withdrawEnable": true
        }
      }
    }
  ],
  "nobitex": {
    "allCurrencies": ["btc", "eth", "usdt"],
    "activeCurrencies": ["btc", "eth"],
    "topCurrencies": ["btc"],
    "minOrders": {"rls": "500000", "usdt": "5"},
    "amountPrecisions": {
      "BTCIRT": "0.000001",
      "BTCUSDT": "0.000001",
      "ETHIRT": "0.00001"
    },
    "pricePrecisions": {
      "BTCIRT": "10",
      "BTCUSDT": "0.01",
      "DOGEUSDT": "0.0000001"
    },
    "giftCard": {"physicalFee": "50000"}
  }
}`

func TestSystemOptionsParsesPrecisions(t *testing.T) {
	t.Parallel()
	var opts types.SystemOptions
	if err := json.Unmarshal([]byte(documentedSystemOptions), &opts); err != nil {
		t.Fatal(err)
	}
	if !opts.Status.IsOK() {
		t.Fatalf("status = %q", opts.Status)
	}
	if got := opts.AmountPrecisions()["BTCIRT"]; got != "0.000001" {
		t.Fatalf("amount BTCIRT = %q", got)
	}
	if got := opts.PricePrecisions()["BTCUSDT"]; got != "0.01" {
		t.Fatalf("price BTCUSDT = %q", got)
	}
	if got := opts.PricePrecisions()["BTCIRT"]; got != "10" {
		t.Fatalf("price BTCIRT = %q", got)
	}
	if got := opts.Nobitex.MinOrders["rls"]; got != "500000" {
		t.Fatalf("minOrders rls = %q", got)
	}
	if len(opts.Coins) != 1 || opts.Coins[0].Coin != "rls" {
		t.Fatalf("coins = %+v", opts.Coins)
	}

	amt, ok := opts.AmountPrecision("btcirt")
	if !ok || amt != "0.000001" {
		t.Fatalf("AmountPrecision(btcirt) = %q ok=%v", amt, ok)
	}
	px, ok := opts.PricePrecision(" BTCIRT ")
	if !ok || px != "10" {
		t.Fatalf("PricePrecision(BTCIRT) = %q ok=%v", px, ok)
	}

	mp, err := opts.MarketPrecision("btcusdt")
	if err != nil {
		t.Fatal(err)
	}
	if mp.Market != "BTCUSDT" || mp.Amount != "0.000001" || mp.Price != "0.01" {
		t.Fatalf("MarketPrecision = %+v", mp)
	}
}

func TestValidateOrderDecimals(t *testing.T) {
	t.Parallel()
	var opts types.SystemOptions
	if err := json.Unmarshal([]byte(documentedSystemOptions), &opts); err != nil {
		t.Fatal(err)
	}

	if err := opts.ValidateOrderDecimals("BTCIRT", "0.001", "35650565900"); err != nil {
		t.Fatalf("valid limit order: %v", err)
	}
	if err := opts.ValidateOrderDecimals("BTCIRT", "0.001", ""); err != nil {
		t.Fatalf("valid market order (empty price): %v", err)
	}
	if err := opts.ValidateOrderDecimals("BTCUSDT", "0.01", "68000.01"); err != nil {
		t.Fatalf("valid BTCUSDT: %v", err)
	}

	err := opts.ValidateOrderDecimals("BTCIRT", "0.00000015", "10")
	if !errors.Is(err, types.ErrNotMultiple) {
		t.Fatalf("want ErrNotMultiple for amount, got %v", err)
	}
	err = opts.ValidateOrderDecimals("BTCIRT", "0.001", "123")
	if !errors.Is(err, types.ErrNotMultiple) {
		t.Fatalf("want ErrNotMultiple for price, got %v", err)
	}
	err = opts.ValidateOrderDecimals("NOPE", "1", "1")
	if !errors.Is(err, types.ErrUnknownMarket) {
		t.Fatalf("want ErrUnknownMarket, got %v", err)
	}
	err = opts.ValidateAmount("BTCIRT", "0")
	if !errors.Is(err, types.ErrNonPositive) {
		t.Fatalf("want ErrNonPositive, got %v", err)
	}
	err = opts.ValidateAmount("  ", "1")
	if !errors.Is(err, types.ErrEmptyMarket) {
		t.Fatalf("want ErrEmptyMarket, got %v", err)
	}
}

func TestPrecisionMapLookupAndTruncate(t *testing.T) {
	t.Parallel()
	p := types.PrecisionMap{"BTCIRT": "0.000001", "100K_FLOKIIRT": "0.001"}
	if v, ok := p.Lookup("btcirt"); !ok || v != "0.000001" {
		t.Fatalf("lookup btcirt = %q %v", v, ok)
	}
	if v, ok := p.Lookup("100K_FLOKIIRT"); !ok || v != "0.001" {
		t.Fatalf("lookup 100K_FLOKIIRT = %q %v", v, ok)
	}
	got, err := p.Truncate("BTCIRT", "0.0000015")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.000001" {
		t.Fatalf("truncate = %q", got)
	}
	if _, ok := p.Lookup("missing"); ok {
		t.Fatal("unexpected hit")
	}
}
