package node

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HONG-LOU/entcoin/internal/core"
	"github.com/HONG-LOU/entcoin/internal/ledger"
)

func TestWalletEndpointReturnsBoundedEmptySnapshot(t *testing.T) {
	service := newTestNode(t)
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/wallet/"+wallet.Address, nil)
	request.SetPathValue("address", wallet.Address)
	service.handleWallet(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET wallet returned HTTP %d: %s", recorder.Code, recorder.Body.String())
	}
	var response walletResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Protocol != ledger.ProtocolName || response.ConfirmedBalance != 0 || response.SpendableBalance != 0 {
		t.Fatalf("unexpected wallet snapshot: %+v", response)
	}
	if response.UTXOsTruncated || len(response.UTXOs) != 0 || len(response.History) != 0 {
		t.Fatalf("empty wallet returned data: %+v", response)
	}
}

func TestWalletEndpointRejectsInvalidAddress(t *testing.T) {
	service := newTestNode(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/wallet/not-an-address", nil)
	request.SetPathValue("address", "not-an-address")
	service.handleWallet(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid wallet address returned HTTP %d", recorder.Code)
	}
}

func TestBrowserAccessAllowsOfficialAndLocalOrigins(t *testing.T) {
	service := newTestNode(t)
	handler := service.browserAccess(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	for _, origin := range []string{"https://entcoin.xyz", "https://wallet.entcoin.xyz", "http://localhost:4174", "http://127.0.0.1:3000"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/v2/status", nil)
		request.Header.Set("Origin", origin)
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK || recorder.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Fatalf("origin %q returned HTTP %d with CORS %q", origin, recorder.Code, recorder.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}

func TestBrowserAccessRejectsUntrustedOriginBeforePost(t *testing.T) {
	service := newTestNode(t)
	called := false
	handler := service.browserAccess(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		called = true
		writer.WriteHeader(http.StatusAccepted)
	}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v2/transactions", nil)
	request.Header.Set("Origin", "https://attacker.example")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden || called {
		t.Fatalf("untrusted origin returned HTTP %d, downstream called = %t", recorder.Code, called)
	}
}
