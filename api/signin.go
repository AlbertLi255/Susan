package api

import (
	"context"
	"net/http"
)

// DeviceSigninState is the state of a device flow.
type DeviceSigninState string

const (
	DeviceStateIdle       DeviceSigninState = "idle"
	DeviceStatePending    DeviceSigninState = "pending"
	DeviceStateAuthorized DeviceSigninState = "authorized"
	DeviceStateDenied     DeviceSigninState = "denied"
	DeviceStateExpired    DeviceSigninState = "expired"
	DeviceStateFailed     DeviceSigninState = "failed"
)

// DeviceSigninRequest starts a device flow through the local daemon.
type DeviceSigninRequest struct {
	// ClientName is shown on the platform authorization page.
	ClientName string `json:"client_name,omitempty"`
	// Force abandons any in-progress flow and requests a new device code.
	Force bool `json:"force,omitempty"`
}

// DeviceSigninResponse is the safe view of a device flow returned by the
// daemon. It never contains the device code or tokens.
type DeviceSigninResponse struct {
	State                   string `json:"state,omitempty"`
	UserCode                string `json:"user_code,omitempty"`
	VerificationURI         string `json:"verification_uri,omitempty"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in,omitempty"`
	Error                   string `json:"error,omitempty"`
}

// RevokeAndSignout asks the daemon to remove the current device from the
// Susan platform account and then delete local credentials.
func (c *Client) RevokeAndSignout(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/signout", map[string]bool{"revoke": true}, nil)
}

// StartDeviceSignin starts (or reuses) the daemon's device flow. The daemon
// polls the Susan platform in the background.
func (c *Client) StartDeviceSignin(ctx context.Context, req *DeviceSigninRequest) (*DeviceSigninResponse, error) {
	var resp DeviceSigninResponse
	if err := c.do(ctx, http.MethodPost, "/api/signin/device", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeviceSigninStatus returns the current device flow status from the daemon.
func (c *Client) DeviceSigninStatus(ctx context.Context) (*DeviceSigninResponse, error) {
	var resp DeviceSigninResponse
	if err := c.do(ctx, http.MethodGet, "/api/signin/device", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
