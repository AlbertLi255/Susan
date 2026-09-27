package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/auth/platform"
	"golang.org/x/crypto/ssh"
)

// generateDaemonTestKeypair writes a keypair into home/.susan so
// DefaultClient can read the public key.
func generateDaemonTestKeypair(t *testing.T, home string) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(home, ".susan")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	pemBytes, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), pem.EncodeToMemory(pemBytes), 0o600); err != nil {
		t.Fatal(err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519.pub"), ssh.MarshalAuthorizedKey(sshPub), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertNoSecrets(t *testing.T, body string) {
	t.Helper()

	for _, secret := range []string{
		"device_code",
		"access_token",
		"refresh_token",
		"dc_susan_secretvalue",
	} {
		if strings.Contains(body, secret) {
			t.Errorf("response contains %q; body: %s", secret, body)
		}
	}
}

type mockTokenStep struct {
	status int
	body   any
}

// mockPlatformConfig scripts the platform mock: token endpoint steps and the
// profile endpoint response.
type mockPlatformConfig struct {
	tokenSteps    []mockTokenStep
	profileStatus int
	profileBody   any
}

// mockPlatformCounters reports how often platform endpoints were hit.
type mockPlatformCounters struct {
	profileHits int
	lastBearer  string
}

// newMockPlatformForSignin runs a minimal Susan platform mock serving
// /auth/device, /auth/token (scripted steps) and /api/user/profile.
func newMockPlatformForSignin(t *testing.T, cfg mockPlatformConfig) (*httptest.Server, *mockPlatformCounters) {
	t.Helper()

	var counters mockPlatformCounters
	var server *httptest.Server
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/device", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"device_code":               "dc_susan_secretvalue",
			"user_code":                 "K7PX-3MQD",
			"verification_uri":          server.URL + "/device",
			"verification_uri_complete": server.URL + "/device?user_code=K7PX-3MQD",
			"expires_in":                900,
			"interval":                  5,
		})
	})

	var tokenIdx int
	mux.HandleFunc("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		step := mockTokenStep{
			status: http.StatusBadRequest,
			body:   map[string]any{"error": "authorization_pending", "interval": 5},
		}
		if tokenIdx < len(cfg.tokenSteps) {
			step = cfg.tokenSteps[tokenIdx]
		}
		tokenIdx++

		w.WriteHeader(step.status)
		json.NewEncoder(w).Encode(step.body)
	})

	mux.HandleFunc("/api/user/profile", func(w http.ResponseWriter, r *http.Request) {
		counters.profileHits++
		counters.lastBearer = r.Header.Get("Authorization")

		if cfg.profileStatus == 0 {
			cfg.profileStatus = http.StatusOK
		}
		if cfg.profileStatus == http.StatusUnauthorized {
			w.Header().Set("WWW-Authenticate", "Bearer")
		}
		if cfg.profileBody == nil {
			cfg.profileBody = map[string]any{
				"id":         "3f6c1d2e-1a2b-4c3d-9e8f-000000000001",
				"email":      "tester@example.com",
				"username":   "tester",
				"created_at": "2026-09-20T08:00:00Z",
			}
		}

		w.WriteHeader(cfg.profileStatus)
		json.NewEncoder(w).Encode(cfg.profileBody)
	})

	server = httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server, &counters
}

func TestSigninDeviceAPI(t *testing.T) {
	srv, _ := newMockPlatformForSignin(t, mockPlatformConfig{
		tokenSteps: []mockTokenStep{
			{status: http.StatusOK, body: map[string]any{
				"access_token":  "ACCESS-1",
				"token_type":    "bearer",
				"refresh_token": "REFRESH-1",
				"expires_in":    3600,
			}},
		},
	})

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("SUSAN_CLOUD_HOST", srv.URL)
	generateDaemonTestKeypair(t, home)

	var s Server
	router, err := s.GenerateRoutes()
	if err != nil {
		t.Fatal(err)
	}

	postReq := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/signin/device", strings.NewReader(`{"client_name":"Susan CLI"}`))
	postReq.RemoteAddr = "127.0.0.1:1234"
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	router.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want 200; body: %s", postRec.Code, postRec.Body.String())
	}
	postBody := postRec.Body.String()
	assertNoSecrets(t, postBody)

	var started platform.Status
	if err := json.Unmarshal(postRec.Body.Bytes(), &started); err != nil {
		t.Fatal(err)
	}
	if started.UserCode == "" || started.VerificationURIComplete == "" {
		t.Errorf("POST response missing user code / verification link: %s", postBody)
	}

	// Poll the local status endpoint until the background flow authorizes.
	var lastBody string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		getReq := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/signin/device", nil)
		getReq.RemoteAddr = "127.0.0.1:1234"
		getRec := httptest.NewRecorder()
		router.ServeHTTP(getRec, getReq)

		if getRec.Code != http.StatusOK {
			t.Fatalf("GET status = %d, want 200; body: %s", getRec.Code, getRec.Body.String())
		}
		lastBody = getRec.Body.String()
		assertNoSecrets(t, lastBody)

		var status platform.Status
		if err := json.Unmarshal(getRec.Body.Bytes(), &status); err != nil {
			t.Fatal(err)
		}
		if status.State == platform.StateAuthorized {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("flow did not reach authorized; last status: %s", lastBody)
}

func TestSigninDeviceRequiresCloudHost(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUSAN_CLOUD_HOST", "")

	var s Server
	router, err := s.GenerateRoutes()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/signin/device", strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "SUSAN_CLOUD_HOST") {
		t.Errorf("body = %q, want it to mention SUSAN_CLOUD_HOST", rec.Body.String())
	}
}
