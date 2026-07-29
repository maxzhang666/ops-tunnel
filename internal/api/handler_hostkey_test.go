package api

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func newTestHostKeyServer(store tunnelssh.HostKeyStore) *Server {
	return &Server{hostKeys: store}
}

func testKey(t *testing.T) ([]byte, string) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	pk, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	return pk.Marshal(), gossh.FingerprintSHA256(pk)
}

func newHostKeyStore(t *testing.T) *tunnelssh.JSONHostKeyStore {
	t.Helper()
	store, err := tunnelssh.NewJSONHostKeyStore(filepath.Join(t.TempDir(), "known_hosts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	return store
}

func TestListKnownHosts(t *testing.T) {
	store := newHostKeyStore(t)
	key, fp := testKey(t)
	store.Add("10.0.0.1:22", key)

	s := newTestHostKeyServer(store)

	rr := httptest.NewRecorder()
	s.listKnownHosts(rr, httptest.NewRequest(http.MethodGet, "/api/v1/host-keys", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body)
	}
	var got knownHostsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Hosts) != 1 || got.Hosts[0].Fingerprint != fp {
		t.Errorf("hosts = %+v", got.Hosts)
	}
}

func TestTrustHostKey(t *testing.T) {
	store := newHostKeyStore(t)
	oldKey, _ := testKey(t)
	newKey, newFP := testKey(t)
	store.Add("10.0.0.1:22", oldKey)
	store.SetPending("10.0.0.1:22", newKey)

	s := newTestHostKeyServer(store)

	body := `{"hostPort":"10.0.0.1:22","fingerprint":"` + newFP + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/host-keys", strings.NewReader(body))
	rr := httptest.NewRecorder()
	s.trustHostKey(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body)
	}
	got, ok := store.Lookup("10.0.0.1:22")
	if !ok || string(got) != string(newKey) {
		t.Error("new key was not trusted")
	}
	if _, ok := store.Pending("10.0.0.1:22"); ok {
		t.Error("pending key should be cleared")
	}
}

func TestTrustHostKey_NoPending(t *testing.T) {
	s := newTestHostKeyServer(newHostKeyStore(t))

	body := `{"hostPort":"10.0.0.1:22","fingerprint":"SHA256:whatever"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/host-keys", strings.NewReader(body))
	rr := httptest.NewRecorder()
	s.trustHostKey(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "no_pending_host_key") {
		t.Errorf("body = %s", rr.Body)
	}
}

func TestTrustHostKey_FingerprintMismatch(t *testing.T) {
	store := newHostKeyStore(t)
	pending, _ := testKey(t)
	store.SetPending("10.0.0.1:22", pending)

	s := newTestHostKeyServer(store)

	body := `{"hostPort":"10.0.0.1:22","fingerprint":"SHA256:stale"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/host-keys", strings.NewReader(body))
	rr := httptest.NewRecorder()
	s.trustHostKey(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "fingerprint_mismatch") {
		t.Errorf("body = %s", rr.Body)
	}
	if _, ok := store.Lookup("10.0.0.1:22"); ok {
		t.Error("nothing should have been trusted")
	}
}

func TestTrustHostKey_MissingFields(t *testing.T) {
	s := newTestHostKeyServer(newHostKeyStore(t))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/host-keys", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	s.trustHostKey(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestRevokeHostKey(t *testing.T) {
	store := newHostKeyStore(t)
	key, _ := testKey(t)
	store.Add("10.0.0.1:22", key)
	store.SetPending("10.0.0.1:22", key)

	s := newTestHostKeyServer(store)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/host-keys?hostPort=10.0.0.1%3A22", nil)
	rr := httptest.NewRecorder()
	s.revokeHostKey(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body)
	}
	if _, ok := store.Lookup("10.0.0.1:22"); ok {
		t.Error("key should be gone")
	}
	if _, ok := store.Pending("10.0.0.1:22"); ok {
		t.Error("pending should be cleared")
	}

	// Idempotent.
	rr2 := httptest.NewRecorder()
	s.revokeHostKey(rr2, httptest.NewRequest(http.MethodDelete, "/api/v1/host-keys?hostPort=10.0.0.1%3A22", nil))
	if rr2.Code != http.StatusOK {
		t.Errorf("second delete status = %d, want 200", rr2.Code)
	}
}

func TestRevokeHostKey_MissingHostPort(t *testing.T) {
	s := newTestHostKeyServer(newHostKeyStore(t))

	rr := httptest.NewRecorder()
	s.revokeHostKey(rr, httptest.NewRequest(http.MethodDelete, "/api/v1/host-keys", nil))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}
