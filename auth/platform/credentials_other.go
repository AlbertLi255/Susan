//go:build !windows

package platform

import "os"

// hardenCredentialsFile restricts the credentials file to owner-only access.
func hardenCredentialsFile(path string) error {
	return os.Chmod(path, 0o600)
}
