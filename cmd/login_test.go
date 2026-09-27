package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

type fakeDaemon struct {
	mu sync.Mutex

	signedIn bool
	// finalState becomes the returned state after pending polls.
	finalState  string
	polls       int
	authorizeAt int
	devicePost  int
	lastForce   bool
	signoutBody []byte
}

func newFakeDaemon(t *testing.T, f *fakeDaemon) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.signedIn {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"6d0c7421-0b16-4d97-be73-1bc643aab1cf","email":"tester@example.com","name":"tester01","plan":"free"}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"you are not signed in","signin_url":"http://localhost:3000/device?user_code=K7PX-3MQD"}`))
	})

	mux.HandleFunc("/api/signin/device", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			var req struct {
				Force bool `json:"force"`
			}
			_ = json.Unmarshal(body, &req)

			f.mu.Lock()
			f.devicePost++
			f.lastForce = req.Force
			f.polls = 0
			f.mu.Unlock()

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"state":"pending","user_code":"K7PX-3MQD","verification_uri":"http://localhost:3000/device","verification_uri_complete":"http://localhost:3000/device?user_code=K7PX-3MQD","expires_in":900}`))
		case http.MethodGet:
			f.mu.Lock()
			f.polls++
			state := "pending"
			if f.authorizeAt > 0 && f.polls >= f.authorizeAt {
				state = f.finalState
				if state == "authorized" {
					f.signedIn = true
				}
			}
			f.mu.Unlock()

			errMsg := ""
			if state == "failed" {
				errMsg = "boom"
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"state": state, "error": errMsg})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/signout", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.signoutBody = body
		signedIn := f.signedIn
		f.signedIn = false
		f.mu.Unlock()

		if !signedIn {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"you are not currently signed in"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"signed out"}`))
	})

	return httptest.NewServer(mux)
}

func captureStdout(fn func() error) (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	execErr := fn()
	w.Close()

	pipeOut, _ := io.ReadAll(r)
	return string(pipeOut), execErr
}

func runLoginCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()

	c := &cobra.Command{Use: "login", RunE: LoginHandler}
	c.Flags().Bool("force", false, "")
	c.SetArgs(args)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c.SetContext(ctx)

	return captureStdout(c.Execute)
}

func runLogoutCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()

	c := &cobra.Command{Use: "logout", RunE: LogoutHandler}
	c.Flags().Bool("revoke", false, "")
	c.SetArgs(args)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c.SetContext(ctx)

	return captureStdout(c.Execute)
}

func TestLoginAlreadySignedIn(t *testing.T) {
	f := &fakeDaemon{signedIn: true}
	srv := newFakeDaemon(t, f)
	defer srv.Close()
	t.Setenv("SUSAN_HOST", srv.URL)

	out, err := runLoginCmd(t)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(out, "Already signed in") {
		t.Fatalf("expected already signed in message, got %q", out)
	}
	if f.devicePost != 0 {
		t.Fatalf("expected no device flow POST, got %d", f.devicePost)
	}
}

func TestLoginSuccess(t *testing.T) {
	f := &fakeDaemon{finalState: "authorized", authorizeAt: 2}
	srv := newFakeDaemon(t, f)
	defer srv.Close()
	t.Setenv("SUSAN_HOST", srv.URL)

	out, err := runLoginCmd(t)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(out, "Signed in successfully") {
		t.Fatalf("expected success message, got %q", out)
	}
}

func TestLoginDenied(t *testing.T) {
	f := &fakeDaemon{finalState: "denied", authorizeAt: 1}
	srv := newFakeDaemon(t, f)
	defer srv.Close()
	t.Setenv("SUSAN_HOST", srv.URL)

	_, err := runLoginCmd(t)
	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected denied error, got %v", err)
	}
}

func TestLoginExpired(t *testing.T) {
	f := &fakeDaemon{finalState: "expired", authorizeAt: 1}
	srv := newFakeDaemon(t, f)
	defer srv.Close()
	t.Setenv("SUSAN_HOST", srv.URL)

	_, err := runLoginCmd(t)
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got %v", err)
	}
}

func TestLoginFailed(t *testing.T) {
	f := &fakeDaemon{finalState: "failed", authorizeAt: 1}
	srv := newFakeDaemon(t, f)
	defer srv.Close()
	t.Setenv("SUSAN_HOST", srv.URL)

	_, err := runLoginCmd(t)
	if err == nil || !strings.Contains(err.Error(), "sign in failed") {
		t.Fatalf("expected failed error, got %v", err)
	}
}

func TestLoginForce(t *testing.T) {
	f := &fakeDaemon{signedIn: true, finalState: "denied", authorizeAt: 1}
	srv := newFakeDaemon(t, f)
	defer srv.Close()
	t.Setenv("SUSAN_HOST", srv.URL)

	_, err := runLoginCmd(t, "--force")
	if err == nil {
		t.Fatal("expected denied error")
	}
	if f.devicePost != 1 {
		t.Fatalf("expected exactly one device POST, got %d", f.devicePost)
	}
	if !f.lastForce {
		t.Fatal("expected force=true in the device POST")
	}
}

func TestLogoutSuccess(t *testing.T) {
	f := &fakeDaemon{signedIn: true}
	srv := newFakeDaemon(t, f)
	defer srv.Close()
	t.Setenv("SUSAN_HOST", srv.URL)

	out, err := runLogoutCmd(t)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(out, "Signed out") {
		t.Fatalf("expected signed out message, got %q", out)
	}
}

func TestLogoutNotSignedIn(t *testing.T) {
	f := &fakeDaemon{signedIn: false}
	srv := newFakeDaemon(t, f)
	defer srv.Close()
	t.Setenv("SUSAN_HOST", srv.URL)

	_, err := runLogoutCmd(t)
	if err == nil || !strings.Contains(err.Error(), "not currently signed in") {
		t.Fatalf("expected not signed in error, got %v", err)
	}
}
