package client_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type p1Capture struct {
	method, path, rawQuery, ua, auth, key, sig, ts, ctype string
	body                                                  []byte
}

func newP1Server(t *testing.T, status int, fixture string, cap *p1Capture) *httptest.Server {
	t.Helper()
	payload := testdata(t, fixture)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cap != nil {
			cap.method = r.Method
			cap.path = r.URL.Path
			cap.rawQuery = r.URL.RawQuery
			cap.ua = r.Header.Get("User-Agent")
			cap.auth = r.Header.Get("Authorization")
			cap.key = r.Header.Get("Nobitex-Key")
			cap.sig = r.Header.Get("Nobitex-Signature")
			cap.ts = r.Header.Get("Nobitex-Timestamp")
			cap.ctype = r.Header.Get("Content-Type")
			cap.body, _ = io.ReadAll(r.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(payload)
	}))
}
