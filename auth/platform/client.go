// Package platform implements the daemon-side client for the Susan platform:
// the OAuth 2.0 Device Authorization flow, token storage (auth.json),
// automatic token refresh, and authenticated requests.
//
// This package is only used inside the local daemon. The device code and
// tokens must never be exposed through the daemon's local API; CLI and Desktop
// clients talk only to 127.0.0.1.
package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DeviceCode is the response to POST /auth/device. DeviceCode is a secret: it
// stays in daemon memory only and is never logged or returned to local API
// clients.
type DeviceCode struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`

	// IssuedAt is set locally to drive expiry calculations.
	IssuedAt time.Time
}

// Expired reports whether the device code has passed its expires_in lifetime.
func (d *DeviceCode) Expired(now time.Time) bool {
	return !now.Before(d.IssuedAt.Add(time.Duration(d.ExpiresIn) * time.Second))
}

// Token is the response to POST /auth/token for both the device_code and
// refresh_token grants.
type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// Profile is the Susan platform user, returned by GET /api/user/profile.
type Profile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

// Terminal errors returned by the device flow. Callers map them to flow states.
var (
	ErrAccessDenied  = errors.New("authorization denied")
	ErrDeviceExpired = errors.New("device code expired")
	ErrInvalidGrant  = errors.New("invalid grant")
	ErrInvalidFlow   = errors.New("invalid device flow request")
	ErrNotSignedIn   = errors.New("not signed in")
)

// tokenError is the top-level error body returned by POST /auth/token on
// HTTP 400.
type tokenError struct {
	Code             string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Interval         int    `json:"interval"`
}

func (e *tokenError) Error() string {
	if e.ErrorDescription != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.ErrorDescription)
	}
	return e.Code
}

// Client talks to one Susan platform instance.
type Client struct {
	BaseURL    string
	PublicKey  string
	DeviceName string
	HTTPClient *http.Client

	// wait pauses the polling loop. It is a field so tests can record the
	// requested wait durations without actually sleeping.
	wait waitFunc
}

type waitFunc func(ctx context.Context, d time.Duration) error

// NewClient creates a platform client. baseURL is the Susan platform root
// (e.g. http://localhost:8000), publicKey the full "ssh-ed25519 ..." line and
// deviceName a name shown on the authorization page and in device lists.
func NewClient(baseURL, publicKey, deviceName string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		PublicKey:  publicKey,
		DeviceName: deviceName,
		HTTPClient: http.DefaultClient,
		wait:       defaultWait,
	}
}

func defaultWait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// doJSON POSTs body to path and decodes a 2xx JSON response into out. For
// non-2xx responses it returns the HTTP status along with an error containing
// the response body; when the body parses as {"error": ...} (the OAuth token
// endpoint), a *tokenError is returned.
func (c *Client) doJSON(ctx context.Context, path string, body, out any) (int, error) {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		r = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, r)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return 0, fmt.Errorf("platform request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}

	if resp.StatusCode/100 != 2 {
		var oerr tokenError
		if json.Unmarshal(data, &oerr) == nil && oerr.Code != "" {
			return resp.StatusCode, &oerr
		}
		return resp.StatusCode, fmt.Errorf("platform returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decoding platform response: %w", err)
		}
	}

	return resp.StatusCode, nil
}

// StartDeviceFlow requests a new device code from POST /auth/device.
func (c *Client) StartDeviceFlow(ctx context.Context, clientName string) (*DeviceCode, error) {
	reqBody := map[string]string{
		"public_key": c.PublicKey,
	}
	if c.DeviceName != "" {
		reqBody["device_name"] = c.DeviceName
	}
	if clientName != "" {
		reqBody["client_name"] = clientName
	}

	var dc DeviceCode
	status, err := c.doJSON(ctx, "/auth/device", reqBody, &dc)
	if err != nil {
		if status == http.StatusUnprocessableEntity {
			return nil, fmt.Errorf("platform rejected the public key (422): %w", err)
		}
		return nil, err
	}

	if dc.Interval <= 0 {
		dc.Interval = 5
	}
	if dc.ExpiresIn <= 0 {
		dc.ExpiresIn = 900
	}
	dc.IssuedAt = time.Now()

	return &dc, nil
}

// ExchangeDeviceCode performs a single POST /auth/token call for a device code.
// On HTTP 400 the OAuth error is returned as *tokenError.
func (c *Client) ExchangeDeviceCode(ctx context.Context, deviceCode string) (*Token, *tokenError, error) {
	return c.tokenRequest(ctx, map[string]string{
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		"device_code": deviceCode,
	})
}

// Refresh performs a single POST /auth/token refresh_token call.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (*Token, *tokenError, error) {
	return c.tokenRequest(ctx, map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	})
}

func (c *Client) tokenRequest(ctx context.Context, body map[string]string) (*Token, *tokenError, error) {
	var tok Token
	status, err := c.doJSON(ctx, "/auth/token", body, &tok)
	if err != nil {
		var oerr *tokenError
		if errors.As(err, &oerr) {
			return nil, oerr, nil
		}
		if status == http.StatusBadRequest {
			return nil, nil, err
		}
		return nil, nil, err
	}
	return &tok, nil, nil
}

// PollToken blocks until the device flow reaches a terminal state. It handles
// authorization_pending and slow_down internally, sleeping the full interval
// (plus a 1 second safety margin) after each response, and retries through
// network failures and 5xx responses until the device code expires.
func (c *Client) PollToken(ctx context.Context, dc *DeviceCode) (*Token, error) {
	interval := time.Duration(dc.Interval) * time.Second

	for {
		if dc.Expired(time.Now()) {
			return nil, ErrDeviceExpired
		}

		tok, oerr, err := c.ExchangeDeviceCode(ctx, dc.DeviceCode)
		switch {
		case oerr != nil:
			switch oerr.Code {
			case "authorization_pending":
				if oerr.Interval > 0 {
					interval = time.Duration(oerr.Interval) * time.Second
				}
			case "slow_down":
				if oerr.Interval > 0 {
					interval = time.Duration(oerr.Interval) * time.Second
				} else {
					interval += 5 * time.Second
				}
			case "access_denied":
				return nil, fmt.Errorf("%w: %s", ErrAccessDenied, oerr.ErrorDescription)
			case "expired_token":
				return nil, fmt.Errorf("%w: %s", ErrDeviceExpired, oerr.ErrorDescription)
			case "invalid_grant":
				return nil, fmt.Errorf("%w: %s", ErrInvalidGrant, oerr.ErrorDescription)
			case "invalid_request", "unsupported_grant_type":
				return nil, fmt.Errorf("%w: %s", ErrInvalidFlow, oerr.Error())
			default:
				return nil, fmt.Errorf("%w: %s", ErrInvalidFlow, oerr.Error())
			}
		case err == nil:
			return tok, nil
		}
		// err != nil (network failure or 5xx): keep the current interval and
		// retry until expiry.

		if err := c.wait(ctx, interval+time.Second); err != nil {
			return nil, err
		}
	}
}
