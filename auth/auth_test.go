package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// setupTestHome points the user home directory at a fresh temp directory so
// keypair operations never touch the developer's real ~/.susan. Windows uses
// USERPROFILE; POSIX uses HOME, so set both.
func setupTestHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	return home
}

// generateTestKeypair writes a keypair into home/.susan, mirroring the format
// produced by initializeKeypair in cmd/cmd.go.
func generateTestKeypair(t *testing.T, home string) {
	t.Helper()

	cryptoPub, cryptoPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	keyDir := filepath.Join(home, ".susan")
	if err := os.MkdirAll(keyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	privateKeyBytes, err := ssh.MarshalPrivateKey(cryptoPriv, "")
	if err != nil {
		t.Fatalf("MarshalPrivateKey: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(keyDir, "id_ed25519"),
		pem.EncodeToMemory(privateKeyBytes),
		0o600,
	); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	sshPub, err := ssh.NewPublicKey(cryptoPub)
	if err != nil {
		t.Fatalf("NewPublicKey: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(keyDir, "id_ed25519.pub"),
		ssh.MarshalAuthorizedKey(sshPub),
		0o644,
	); err != nil {
		t.Fatalf("write public key: %v", err)
	}
}

func TestGetPublicKey(t *testing.T) {
	home := setupTestHome(t)
	generateTestKeypair(t, home)

	pub, err := GetPublicKey()
	if err != nil {
		t.Fatalf("GetPublicKey: %v", err)
	}

	if !strings.HasPrefix(pub, "ssh-ed25519 ") {
		t.Errorf("GetPublicKey() = %q, want it to start with %q", pub, "ssh-ed25519 ")
	}

	pubFileBytes, err := os.ReadFile(filepath.Join(home, ".susan", "id_ed25519.pub"))
	if err != nil {
		t.Fatalf("read public key file: %v", err)
	}

	if got, want := pub, strings.TrimSpace(string(pubFileBytes)); got != want {
		t.Errorf("GetPublicKey() = %q, want %q (id_ed25519.pub)", got, want)
	}
}

func TestGetPublicKeyMissingKey(t *testing.T) {
	setupTestHome(t)

	if _, err := GetPublicKey(); err == nil {
		t.Error("GetPublicKey() error = nil, want an error when no keypair exists")
	}
}

func TestSign(t *testing.T) {
	home := setupTestHome(t)
	generateTestKeypair(t, home)

	payload := []byte("susan-auth-challenge:test-nonce")

	signature, err := Sign(context.Background(), payload)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	parts := strings.Split(signature, ":")
	if len(parts) != 2 {
		t.Fatalf("Sign() = %q, want exactly two colon-separated parts", signature)
	}

	pubKeyBase64, sigBase64 := parts[0], parts[1]

	// The public key segment must match GetPublicKey().
	expectedPub, err := GetPublicKey()
	if err != nil {
		t.Fatalf("GetPublicKey: %v", err)
	}
	if got, want := pubKeyBase64, strings.Fields(expectedPub)[1]; got != want {
		t.Errorf("signing public key = %q, want %q", got, want)
	}

	parsedPub, _, _, _, err := ssh.ParseAuthorizedKey([]byte("ssh-ed25519 " + pubKeyBase64))
	if err != nil {
		t.Fatalf("ParseAuthorizedKey: %v", err)
	}

	sigBlob, err := base64.StdEncoding.DecodeString(sigBase64)
	if err != nil {
		t.Fatalf("signature is not base64: %v", err)
	}

	sshSig := &ssh.Signature{
		Format: parsedPub.Type(),
		Blob:   sigBlob,
	}

	if err := parsedPub.Verify(payload, sshSig); err != nil {
		t.Errorf("valid signature failed verification: %v", err)
	}

	tampered := make([]byte, len(payload))
	copy(tampered, payload)
	tampered[0]++
	if err := parsedPub.Verify(tampered, sshSig); err == nil {
		t.Error("signature verified tampered payload, want verification failure")
	}

	otherBlob := make([]byte, len(sigBlob))
	copy(otherBlob, sigBlob)
	otherBlob[0]++
	if err := parsedPub.Verify(payload, &ssh.Signature{Format: parsedPub.Type(), Blob: otherBlob}); err == nil {
		t.Error("tampered signature verified, want verification failure")
	}
}

func TestKeypairFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}

	home := setupTestHome(t)
	generateTestKeypair(t, home)

	privInfo, err := os.Stat(filepath.Join(home, ".susan", "id_ed25519"))
	if err != nil {
		t.Fatalf("stat private key: %v", err)
	}

	if perm := privInfo.Mode().Perm(); perm != 0o600 {
		t.Errorf("private key permissions = %04o, want 0600", perm)
	}
}
