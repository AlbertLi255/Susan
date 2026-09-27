package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ollama/ollama/envconfig"
)

const credentialsVersion = 1

// Credentials is the on-disk representation of the platform session in
// ~/.susan/auth.json.
type Credentials struct {
	Version      int       `json:"version"`
	CloudHost    string    `json:"cloud_host"`
	TokenType    string    `json:"token_type"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	DeviceName   string    `json:"device_name"`
}

// ErrCloudHostMismatch indicates stored credentials were issued by a different
// platform than the current SUSAN_CLOUD_HOST; they must not be used.
var ErrCloudHostMismatch = errors.New("credentials belong to a different cloud host")

// CredentialsPath returns ~/.susan/auth.json.
func CredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".susan", "auth.json"), nil
}

// LoadCredentials reads and validates auth.json. A missing file returns
// (nil, nil). Credentials whose cloud_host differs from the current CloudHost
// are ignored and ErrCloudHostMismatch is returned. On POSIX, a file with
// permissions wider than 0600 is tightened; if it cannot be tightened the
// credentials are rejected.
func LoadCredentials() (*Credentials, error) {
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm&0o077 != 0 {
			slog.Warn("credentials file is accessible by others, tightening permissions", "path", path, "permissions", fmt.Sprintf("%04o", perm))
			if err := os.Chmod(path, 0o600); err != nil {
				return nil, fmt.Errorf("could not tighten credentials file permissions: %w", err)
			}
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cred Credentials
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}

	if cred.CloudHost != envconfig.CloudHost().String() {
		return nil, fmt.Errorf("%w: credentials for %q but cloud host is %q", ErrCloudHostMismatch, cred.CloudHost, envconfig.CloudHost().String())
	}

	return &cred, nil
}

// SaveCredentials atomically writes cred to auth.json: a temporary file in the
// same directory is created, hardened and then renamed over the target.
func SaveCredentials(cred *Credentials) error {
	if cred == nil {
		return errors.New("cannot save nil credentials")
	}

	cred.Version = credentialsVersion

	path, err := CredentialsPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating credentials directory: %w", err)
	}

	data, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".auth.json.tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	cleanup := true
	defer func() {
		if cleanup {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// Harden before publishing the name. Failures are best effort: log and
	// continue on Windows, the platform-specific implementation decides.
	if err := hardenCredentialsFile(tmpName); err != nil {
		if runtime.GOOS == "windows" {
			slog.Warn("could not apply protected DACL to credentials file", "path", path, "error", err)
		} else {
			return fmt.Errorf("could not secure credentials file: %w", err)
		}
	}

	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false

	return nil
}

// DeleteCredentials removes auth.json. A missing file is not an error.
func DeleteCredentials() error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
