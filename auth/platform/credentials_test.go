package platform

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"
)

func saveTestCredentials(t *testing.T, cloudHost, access, refresh string) {
	t.Helper()

	cred := &Credentials{
		CloudHost:    cloudHost,
		TokenType:    "bearer",
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    time.Now().Add(-time.Hour), // forces a refresh on next Do
		DeviceName:   "TEST-DEVICE",
	}
	if err := SaveCredentials(cred); err != nil {
		t.Fatal(err)
	}
}

func newTestSession(t *testing.T, srvURL string) *Session {
	t.Helper()

	c := NewClient(srvURL, testPublicKey(t), "TEST-DEVICE")
	c.HTTPClient = &http.Client{}

	s, err := NewSession(c)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSessionRefreshOverwritesTokens(t *testing.T) {
	srv, mock := newMockPlatform(t, []tokenStep{
		tokenResponse("ACCESS-2", "REFRESH-2"),
	})
	setupTestEnv(t, srv.URL)
	saveTestCredentials(t, srv.URL, "ACCESS-1", "REFRESH-1")

	s := newTestSession(t, srv.URL)

	profile, err := s.GetProfile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if profile.Username != "tester01" {
		t.Errorf("profile username = %q, want tester01", profile.Username)
	}

	cred, err := LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if cred.AccessToken != "ACCESS-2" || cred.RefreshToken != "REFRESH-2" {
		t.Errorf("tokens after refresh = %q/%q, want ACCESS-2/REFRESH-2", cred.AccessToken, cred.RefreshToken)
	}

	mock.mu.Lock()
	authHeader := mock.lastAuthHeader
	mock.mu.Unlock()
	if authHeader != "Bearer ACCESS-2" {
		t.Errorf("profile was requested with %q, want Bearer ACCESS-2", authHeader)
	}
}

func TestSessionRefreshInvalidGrantDeletesCredentials(t *testing.T) {
	srv, _ := newMockPlatform(t, []tokenStep{
		oauthError("invalid_grant", 0),
	})
	setupTestEnv(t, srv.URL)
	saveTestCredentials(t, srv.URL, "ACCESS-1", "REFRESH-1")

	s := newTestSession(t, srv.URL)

	if _, err := s.GetProfile(context.Background()); !errors.Is(err, ErrNotSignedIn) {
		t.Errorf("GetProfile error = %v, want ErrNotSignedIn", err)
	}

	cred, err := LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if cred != nil {
		t.Errorf("credentials still present after invalid_grant: %+v", cred)
	}

	path, err := CredentialsPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("auth.json still exists after invalid_grant")
	}
}

func TestLoadCredentialsCloudHostMismatch(t *testing.T) {
	srv, mock := newMockPlatform(t, nil)

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	t.Setenv("SUSAN_CLOUD_HOST", "http://other.example")
	saveTestCredentials(t, "http://other.example", "ACCESS-1", "REFRESH-1")

	t.Setenv("SUSAN_CLOUD_HOST", srv.URL)

	if _, err := LoadCredentials(); !errors.Is(err, ErrCloudHostMismatch) {
		t.Errorf("LoadCredentials error = %v, want ErrCloudHostMismatch", err)
	}

	s := newTestSession(t, srv.URL)
	if s.SignedIn() {
		t.Error("session reports signed in despite cloud host mismatch")
	}
	if _, err := s.GetProfile(context.Background()); !errors.Is(err, ErrNotSignedIn) {
		t.Errorf("GetProfile error = %v, want ErrNotSignedIn", err)
	}

	mock.mu.Lock()
	protectedHits := mock.protectedHits
	mock.mu.Unlock()
	if protectedHits != 0 {
		t.Errorf("platform protected endpoint was hit %d times, want 0", protectedHits)
	}
}

func TestMissingCredentials(t *testing.T) {
	setupTestEnv(t, "http://unused.example")

	cred, err := LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if cred != nil {
		t.Errorf("LoadCredentials = %+v, want nil", cred)
	}

	if err := DeleteCredentials(); err != nil {
		t.Errorf("DeleteCredentials without file: %v", err)
	}
}
