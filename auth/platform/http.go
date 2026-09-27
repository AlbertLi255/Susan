package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ollama/ollama/auth"
	"github.com/ollama/ollama/envconfig"
)

// refreshThreshold is the minimum remaining access token lifetime; requests
// with less time left trigger a refresh before being sent.
const refreshThreshold = 60 * time.Second

// DefaultClient builds a platform client from the daemon environment:
// SUSAN_CLOUD_HOST, the device public key (~/.susan/id_ed25519.pub) and the
// machine hostname.
func DefaultClient() (*Client, error) {
	pub, err := auth.GetPublicKey()
	if err != nil {
		return nil, fmt.Errorf("device keypair is not available: %w", err)
	}

	deviceName, err := os.Hostname()
	if err != nil {
		deviceName = "unknown-device"
	}

	return NewClient(envconfig.CloudHost().String(), pub, deviceName), nil
}

// Session holds the current credentials and serializes token refreshes within
// the daemon.
type Session struct {
	client *Client

	mu   sync.Mutex
	cred *Credentials
}

// NewSession creates a session, loading auth.json if present. A cloud host
// mismatch is treated as signed out.
func NewSession(client *Client) (*Session, error) {
	cred, err := LoadCredentials()
	if errors.Is(err, ErrCloudHostMismatch) {
		cred = nil
		err = nil
	}
	if err != nil {
		return nil, err
	}

	return &Session{
		client: client,
		cred:   cred,
	}, nil
}

// SignedIn reports whether usable credentials are held.
func (s *Session) SignedIn() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cred != nil
}

// Credentials returns a copy of the current credentials, or nil.
func (s *Session) Credentials() *Credentials {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cred == nil {
		return nil
	}
	c := *s.cred
	return &c
}

// Clear removes credentials from memory and disk (e.g. after an invalid_grant).
func (s *Session) Clear() error {
	s.mu.Lock()
	s.cred = nil
	s.mu.Unlock()
	return DeleteCredentials()
}

// refreshLocked exchanges the refresh token for a new pair, updates memory and
// disk, and marks the session signed out on invalid_grant. The caller must
// hold s.mu.
func (s *Session) refreshLocked(ctx context.Context) error {
	if s.cred == nil || s.cred.RefreshToken == "" {
		return ErrNotSignedIn
	}

	tok, oerr, err := s.client.Refresh(ctx, s.cred.RefreshToken)
	switch {
	case oerr != nil && oerr.Code == "invalid_grant":
		s.cred = nil
		if rmErr := DeleteCredentials(); rmErr != nil {
			return rmErr
		}
		return fmt.Errorf("%w: %s", ErrNotSignedIn, oerr.ErrorDescription)
	case err != nil:
		return fmt.Errorf("token refresh failed: %w", err)
	case oerr != nil:
		return fmt.Errorf("token refresh failed: %s", oerr.Error())
	}

	newCred := credentialsFromToken(s.client, tok, s.cred.DeviceName)
	if err := SaveCredentials(newCred); err != nil {
		return err
	}
	s.cred = newCred
	return nil
}

func credentialsFromToken(c *Client, tok *Token, deviceName string) *Credentials {
	tokenType := tok.TokenType
	if tokenType == "" {
		tokenType = "bearer"
	}

	expiresIn := time.Duration(tok.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = time.Hour
	}

	return &Credentials{
		CloudHost:    c.BaseURL,
		TokenType:    tokenType,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresAt:    time.Now().Add(expiresIn),
		DeviceName:   deviceName,
	}
}

// Do sends req with a Bearer access token. The access token is refreshed first
// if it expires in less than refreshThreshold, and once more after a 401
// response with a single retry of the request. An invalid refresh token marks
// the session signed out.
func (s *Session) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	if s.cred == nil {
		s.mu.Unlock()
		return nil, ErrNotSignedIn
	}

	if time.Until(s.cred.ExpiresAt) < refreshThreshold {
		if err := s.refreshLocked(ctx); err != nil {
			s.mu.Unlock()
			return nil, err
		}
	}
	cred := *s.cred
	s.mu.Unlock()

	resp, err := s.sendWithToken(req, &cred)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	resp.Body.Close()

	s.mu.Lock()
	if err := s.refreshLocked(ctx); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	cred = *s.cred
	s.mu.Unlock()

	return s.sendWithToken(req, &cred)
}

func (s *Session) sendWithToken(req *http.Request, cred *Credentials) (*http.Response, error) {
	r2 := req.Clone(req.Context())
	r2.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	return s.client.httpClient().Do(r2)
}

// GetProfile fetches the current user from GET /api/user/profile.
func (s *Session) GetProfile(ctx context.Context) (*Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.client.BaseURL+"/api/user/profile", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("profile request returned HTTP %d", resp.StatusCode)
	}

	var profile Profile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}
	return &profile, nil
}
