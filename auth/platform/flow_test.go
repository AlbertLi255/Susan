package platform

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// tokenStep is one scripted POST /auth/token response.
type tokenStep struct {
	status int
	body   any
}

type mockPlatform struct {
	mu             sync.Mutex
	deviceHits     int
	tokenHits      int
	protectedHits  int
	lastAuthHeader string
	tokenSteps     []tokenStep
	server         *httptest.Server
}

func newMockPlatform(t *testing.T, steps []tokenStep) (*httptest.Server, *mockPlatform) {
	t.Helper()

	m := &mockPlatform{tokenSteps: steps}
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/device", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		pk, _ := body["public_key"].(string)
		if !strings.HasPrefix(pk, "ssh-ed25519 ") {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]any{"detail": []any{map[string]string{"msg": "invalid public key"}}})
			return
		}

		m.mu.Lock()
		m.deviceHits++
		userCode := fmt.Sprintf("K7PX-3M%02d", m.deviceHits)
		m.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"device_code":               "dc_susan_secretvalue",
			"user_code":                 userCode,
			"verification_uri":          m.server.URL + "/device",
			"verification_uri_complete": m.server.URL + "/device?user_code=" + userCode,
			"expires_in":                900,
			"interval":                  5,
		})
	})

	mux.HandleFunc("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		idx := m.tokenHits
		m.tokenHits++

		step := tokenStep{
			status: http.StatusBadRequest,
			body: map[string]any{
				"error":    "authorization_pending",
				"interval": 5,
			},
		}
		if idx < len(m.tokenSteps) {
			step = m.tokenSteps[idx]
		}
		m.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(step.status)
		json.NewEncoder(w).Encode(step.body)
	})

	mux.HandleFunc("/api/user/profile", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		m.mu.Lock()
		m.protectedHits++
		m.lastAuthHeader = authHeader
		m.mu.Unlock()

		if !strings.HasPrefix(authHeader, "Bearer ACCESS-") {
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"detail": "Could not validate credentials"})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":         "user-id-1",
			"email":      "susan_test@example.com",
			"username":   "tester01",
			"created_at": "2026-01-01T00:00:00Z",
		})
	})

	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)

	return m.server, m
}

func testPublicKey(t *testing.T) string {
	t.Helper()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(s)))
}

type waitRecorder struct {
	mu    sync.Mutex
	waits []time.Duration
}

func (w *waitRecorder) record(ctx context.Context, d time.Duration) error {
	w.mu.Lock()
	w.waits = append(w.waits, d)
	w.mu.Unlock()
	return nil
}

func (w *waitRecorder) lengths() []time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]time.Duration, len(w.waits))
	copy(out, w.waits)
	return out
}

// testManager builds a manager whose clients point at srvURL and never really
// sleep while polling.
func testManager(t *testing.T, srvURL string, wr *waitRecorder) *Manager {
	t.Helper()

	pub := testPublicKey(t)
	return NewManager(func() (*Client, error) {
		c := NewClient(srvURL, pub, "TEST-DEVICE")
		c.HTTPClient = &http.Client{}
		c.wait = wr.record
		return c, nil
	})
}

func waitForState(t *testing.T, m *Manager, want State) Status {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status := m.CurrentStatus()
		if status.State == want {
			return status
		}
		if status.State != StatePending && want != StatePending {
			// settle briefly for terminal transitions
		}
		time.Sleep(5 * time.Millisecond)
	}

	status := m.CurrentStatus()
	t.Fatalf("state = %q, want %q (error: %s)", status.State, want, status.Error)
	return Status{}
}

func tokenResponse(access, refresh string) tokenStep {
	return tokenStep{
		status: http.StatusOK,
		body: map[string]any{
			"access_token":  access,
			"token_type":    "bearer",
			"refresh_token": refresh,
			"expires_in":    3600,
		},
	}
}

func oauthError(code string, interval int) tokenStep {
	body := map[string]any{"error": code}
	if interval > 0 {
		body["interval"] = interval
	}
	return tokenStep{status: http.StatusBadRequest, body: body}
}

func setupTestEnv(t *testing.T, srvURL string) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("SUSAN_CLOUD_HOST", srvURL)
}

func TestFlowAuthorizedAfterPending(t *testing.T) {
	srv, _ := newMockPlatform(t, []tokenStep{
		oauthError("authorization_pending", 5),
		oauthError("authorization_pending", 5),
		tokenResponse("ACCESS-1", "REFRESH-1"),
	})
	setupTestEnv(t, srv.URL)

	var wr waitRecorder
	m := testManager(t, srv.URL, &wr)

	status, err := m.Start(context.Background(), "Susan CLI", false)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != StatePending || status.UserCode == "" {
		t.Fatalf("start status = %+v, want pending with user code", status)
	}

	final := waitForState(t, m, StateAuthorized)
	_ = final

	path, err := CredentialsPath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("auth.json was not written: %v", err)
	}
	body := string(data)
	for _, want := range []string{`"cloud_host": "` + srv.URL + `"`, "ACCESS-1", "REFRESH-1", `"version": 1`} {
		if !strings.Contains(body, want) {
			t.Errorf("auth.json missing %q; contents:\n%s", want, body)
		}
	}

	waits := wr.lengths()
	if len(waits) < 2 {
		t.Fatalf("recorded %d waits, want at least 2", len(waits))
	}
}

func TestFlowSlowDownIncreasesInterval(t *testing.T) {
	srv, _ := newMockPlatform(t, []tokenStep{
		oauthError("slow_down", 7),
		tokenResponse("ACCESS-1", "REFRESH-1"),
	})
	setupTestEnv(t, srv.URL)

	var wr waitRecorder
	m := testManager(t, srv.URL, &wr)

	if _, err := m.Start(context.Background(), "Susan CLI", false); err != nil {
		t.Fatal(err)
	}

	waitForState(t, m, StateAuthorized)

	waits := wr.lengths()
	if len(waits) < 1 {
		t.Fatal("no wait recorded after slow_down")
	}
	// The client must sleep the new interval (7s) plus the 1s margin.
	if want := 8 * time.Second; waits[0] < want {
		t.Errorf("wait after slow_down = %v, want at least %v", waits[0], want)
	}
}

func TestFlowTerminalStates(t *testing.T) {
	cases := []struct {
		name      string
		step      tokenStep
		wantState State
	}{
		{"denied", oauthError("access_denied", 0), StateDenied},
		{"expired", oauthError("expired_token", 0), StateExpired},
		{"invalid_grant", oauthError("invalid_grant", 0), StateFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, mock := newMockPlatform(t, []tokenStep{tc.step})
			setupTestEnv(t, srv.URL)

			var wr waitRecorder
			m := testManager(t, srv.URL, &wr)

			if _, err := m.Start(context.Background(), "Susan CLI", false); err != nil {
				t.Fatal(err)
			}

			status := waitForState(t, m, tc.wantState)
			if status.Error == "" {
				t.Errorf("state %q has empty error description", tc.wantState)
			}

			mock.mu.Lock()
			hits := mock.tokenHits
			mock.mu.Unlock()
			if hits != 1 {
				t.Errorf("token requests = %d, want 1 (polling must stop)", hits)
			}
		})
	}
}

func TestFlowReuseAndForce(t *testing.T) {
	srv, mock := newMockPlatform(t, []tokenStep{
		oauthError("authorization_pending", 5),
		oauthError("authorization_pending", 5),
		tokenResponse("ACCESS-1", "REFRESH-1"),
	})
	setupTestEnv(t, srv.URL)

	pub := testPublicKey(t)

	// proceed gates the polling loop: the first poller parks after its
	// pending response, making reuse deterministic. Closing it releases
	// later flows.
	proceed := make(chan struct{})
	wait := func(ctx context.Context, d time.Duration) error {
		select {
		case <-proceed:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	m := NewManager(func() (*Client, error) {
		c := NewClient(srv.URL, pub, "TEST-DEVICE")
		c.HTTPClient = &http.Client{}
		c.wait = wait
		return c, nil
	})

	first, err := m.Start(context.Background(), "Susan CLI", false)
	if err != nil {
		t.Fatal(err)
	}
	if m.CurrentStatus().State != StatePending {
		t.Fatal("first flow is not pending")
	}

	reused, err := m.Start(context.Background(), "Susan CLI", false)
	if err != nil {
		t.Fatal(err)
	}

	if reused.UserCode != first.UserCode {
		t.Errorf("reuse user code = %q, want %q", reused.UserCode, first.UserCode)
	}
	mock.mu.Lock()
	deviceHits := mock.deviceHits
	mock.mu.Unlock()
	if deviceHits != 1 {
		t.Errorf("device requests = %d, want 1 for reuse", deviceHits)
	}

	forced, err := m.Start(context.Background(), "Susan CLI", true)
	if err != nil {
		t.Fatal(err)
	}
	if forced.UserCode == first.UserCode {
		t.Errorf("force returned same user code %q, want a new one", forced.UserCode)
	}
	mock.mu.Lock()
	deviceHits = mock.deviceHits
	mock.mu.Unlock()
	if deviceHits != 2 {
		t.Errorf("device requests = %d, want 2 after force", deviceHits)
	}

	close(proceed)
	waitForState(t, m, StateAuthorized)
}

func TestStartDeviceFlow422(t *testing.T) {
	srv, _ := newMockPlatform(t, nil)
	setupTestEnv(t, srv.URL)

	c := NewClient(srv.URL, "not-a-valid-key", "TEST-DEVICE")
	c.HTTPClient = &http.Client{}

	if _, err := c.StartDeviceFlow(context.Background(), "Susan CLI"); err == nil {
		t.Fatal("StartDeviceFlow with invalid public key: want error, got nil")
	}
}
