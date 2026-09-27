package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Device is one entry returned by GET /api/user/devices.
type Device struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	DeviceName string `json:"device_name"`
	PublicKey  string `json:"public_key"`
	LastActive string `json:"last_active"`
	CreatedAt  string `json:"created_at"`
}

// ListDevices returns the signed in user's devices.
func (s *Session) ListDevices(ctx context.Context) ([]Device, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.client.BaseURL+"/api/user/devices", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("devices request returned HTTP %d", resp.StatusCode)
	}

	var devices []Device
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, err
	}
	return devices, nil
}

// RevokeDevice deletes the given device from the user's account.
func (s *Session) RevokeDevice(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.client.BaseURL+"/api/user/devices/"+id, nil)
	if err != nil {
		return err
	}

	resp, err := s.Do(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("revoke device request returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// RevokeCurrentDevice finds the device whose public key matches and removes it
// from the user's account.
func (s *Session) RevokeCurrentDevice(ctx context.Context, publicKey string) error {
	devices, err := s.ListDevices(ctx)
	if err != nil {
		return err
	}

	for _, d := range devices {
		if d.PublicKey == publicKey {
			return s.RevokeDevice(ctx, d.ID)
		}
	}

	return fmt.Errorf("current device was not found in the account's device list")
}
