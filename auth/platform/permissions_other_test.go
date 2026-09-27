//go:build !windows

package platform

import (
	"os"
	"testing"
	"time"
)

func TestCredentialsFilePermissions(t *testing.T) {
	setupTestEnv(t, "http://localhost:8000")

	if err := SaveCredentials(&Credentials{
		CloudHost:    "http://localhost:8000",
		AccessToken:  "a",
		RefreshToken: "r",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	path, err := CredentialsPath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("credentials permissions = %04o, want 0600", perm)
	}
}
