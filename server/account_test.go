package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/auth/platform"
)

func currentCloudHost(t *testing.T) string {
	t.Helper()
	return os.Getenv("SUSAN_CLOUD_HOST")
}

func extractJSONString(body, key string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func setupAccountTest(t *testing.T, cfg mockPlatformConfig) (http.Handler, *mockPlatformCounters) {
	t.Helper()

	srv, counters := newMockPlatformForSignin(t, cfg)

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("SUSAN_CLOUD_HOST", srv.URL)
	generateDaemonTestKeypair(t, home)

	platform.DefaultManager().Reset()

	router, err := (&Server{}).GenerateRoutes()
	if err != nil {
		t.Fatal(err)
	}
	return router, counters
}

func postToRouter(t *testing.T, router http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1"+path, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func writeAccountCredentials(t *testing.T, srvURL, accessToken string, expiresAt time.Time) {
	t.Helper()

	err := platform.SaveCredentials(&platform.Credentials{
		CloudHost:    srvURL,
		TokenType:    "bearer",
		AccessToken:  accessToken,
		RefreshToken: "REFRESH-1",
		ExpiresAt:    expiresAt,
		DeviceName:   "TEST-DEVICE",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWhoamiUnauthorizedIncludesSigninURL(t *testing.T) {
	router, counters := setupAccountTest(t, mockPlatformConfig{})

	rec := postToRouter(t, router, "/api/me", "{}")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"error":"unauthorized"`) {
		t.Errorf("body = %q, want unauthorized error", body)
	}
	signinURL := extractJSONString(body, "signin_url")
	if !strings.Contains(signinURL, "/device?user_code=") {
		t.Errorf("signin_url = %q, want a device flow complete link", signinURL)
	}

	// The profile endpoint must never be called when there are no credentials.
	if counters.profileHits != 0 {
		t.Errorf("profile endpoint was hit %d times, want 0", counters.profileHits)
	}

	// A second 401 reuses the in-progress flow and returns the same URL.
	rec2 := postToRouter(t, router, "/api/me", "{}")
	signinURL2 := extractJSONString(rec2.Body.String(), "signin_url")
	if signinURL2 != signinURL {
		t.Errorf("second signin_url = %q, want same as %q", signinURL2, signinURL)
	}
}

func TestWhoamiProfileMapping(t *testing.T) {
	router, _ := setupAccountTest(t, mockPlatformConfig{
		profileBody: map[string]any{
			"id":         "3f6c1d2e-1a2b-4c3d-9e8f-000000000042",
			"email":      "tester@example.com",
			"username":   "tester",
			"created_at": "2026-09-20T08:00:00Z",
		},
	})

	srvURL := currentCloudHost(t)
	writeAccountCredentials(t, srvURL, "ACCESS-1", time.Now().Add(time.Hour))

	rec := postToRouter(t, router, "/api/me", "{}")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{
		`"id":"3f6c1d2e-1a2b-4c3d-9e8f-000000000042"`,
		`"email":"tester@example.com"`,
		`"name":"tester"`,
		`"plan":"free"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q; got: %s", want, body)
		}
	}
}

func TestWhoamiRefreshesExpiredAccessToken(t *testing.T) {
	router, counters := setupAccountTest(t, mockPlatformConfig{
		tokenSteps: []mockTokenStep{
			{status: http.StatusOK, body: map[string]any{
				"access_token":  "ACCESS-2",
				"token_type":    "bearer",
				"refresh_token": "REFRESH-2",
				"expires_in":    3600,
			}},
		},
	})

	srvURL := currentCloudHost(t)
	writeAccountCredentials(t, srvURL, "ACCESS-1", time.Now().Add(-time.Hour))

	rec := postToRouter(t, router, "/api/me", "{}")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	if counters.lastBearer != "Bearer ACCESS-2" {
		t.Errorf("profile used %q, want Bearer ACCESS-2", counters.lastBearer)
	}

	cred, err := platform.LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if cred.AccessToken != "ACCESS-2" || cred.RefreshToken != "REFRESH-2" {
		t.Errorf("credentials after refresh = %q/%q, want ACCESS-2/REFRESH-2", cred.AccessToken, cred.RefreshToken)
	}
}

func TestWhoamiRefreshFailureDeletesCredentials(t *testing.T) {
	router, counters := setupAccountTest(t, mockPlatformConfig{
		tokenSteps: []mockTokenStep{
			{status: http.StatusBadRequest, body: map[string]any{
				"error":             "invalid_grant",
				"error_description": "Invalid or expired refresh token",
			}},
		},
	})

	srvURL := currentCloudHost(t)
	writeAccountCredentials(t, srvURL, "ACCESS-1", time.Now().Add(-time.Hour))

	rec := postToRouter(t, router, "/api/me", "{}")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body: %s", rec.Code, rec.Body.String())
	}

	if counters.profileHits != 0 {
		t.Errorf("profile was hit %d times, want 0", counters.profileHits)
	}

	cred, err := platform.LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if cred != nil {
		t.Errorf("credentials still present after refresh failure: %+v", cred)
	}
}

func TestWhoamiPlatform5xxReturns503(t *testing.T) {
	router, _ := setupAccountTest(t, mockPlatformConfig{
		profileStatus: http.StatusInternalServerError,
		profileBody:   map[string]any{"detail": "boom"},
	})

	srvURL := currentCloudHost(t)
	writeAccountCredentials(t, srvURL, "ACCESS-1", time.Now().Add(time.Hour))

	rec := postToRouter(t, router, "/api/me", "{}")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "account unavailable") {
		t.Errorf("body = %q, want account unavailable", rec.Body.String())
	}

	// Credentials must not be deleted on a platform failure.
	cred, err := platform.LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if cred == nil {
		t.Error("credentials were removed on platform 5xx; they should remain")
	}
}

func TestSignout(t *testing.T) {
	t.Run("signed in", func(t *testing.T) {
		router, _ := setupAccountTest(t, mockPlatformConfig{})

		srvURL := currentCloudHost(t)
		writeAccountCredentials(t, srvURL, "ACCESS-1", time.Now().Add(time.Hour))

		rec := postToRouter(t, router, "/api/signout", "{}")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
		}

		cred, err := platform.LoadCredentials()
		if err != nil {
			t.Fatal(err)
		}
		if cred != nil {
			t.Error("credentials still present after sign out")
		}
	})

	t.Run("not signed in", func(t *testing.T) {
		router, _ := setupAccountTest(t, mockPlatformConfig{})

		rec := postToRouter(t, router, "/api/signout", "{}")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401; body: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "not currently signed in") {
			t.Errorf("body = %q, want not signed in message", rec.Body.String())
		}
	})
}
