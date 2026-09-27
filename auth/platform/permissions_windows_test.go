//go:build windows

package platform

import (
	"os/user"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestCredentialsFileDACL(t *testing.T) {
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

	sd, err := windows.GetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		t.Fatal(err)
	}

	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatalf("DACL is not present on credentials file: %v", err)
	}

	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	userSid, err := windows.StringToSid(u.Uid)
	if err != nil {
		t.Fatal(err)
	}
	systemSid, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		t.Fatal(err)
	}

	allowed := map[string]bool{}

	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			t.Fatalf("GetAce(%d): %v", i, err)
		}

		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		allowed[sid.String()] = true
	}

	if len(allowed) != 2 {
		t.Fatalf("DACL has %d trustees (%v), want exactly 2 (current user + SYSTEM)", len(allowed), allowed)
	}

	if !allowed[userSid.String()] {
		t.Errorf("current user %s missing from DACL: %v", userSid, allowed)
	}
	if !allowed[systemSid.String()] {
		t.Errorf("SYSTEM %s missing from DACL: %v", systemSid, allowed)
	}
}
