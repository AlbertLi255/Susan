package platform

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State is the state of the current device flow.
type State string

const (
	// StateIdle: no flow has been started.
	StateIdle State = "idle"
	// StatePending: a device code exists and the daemon is polling.
	StatePending State = "pending"
	// StateAuthorized: tokens were obtained and auth.json was written.
	StateAuthorized State = "authorized"
	// StateDenied: the user denied authorization.
	StateDenied State = "denied"
	// StateExpired: the device code expired before authorization.
	StateExpired State = "expired"
	// StateFailed: an unrecoverable error occurred (e.g. platform unreachable).
	StateFailed State = "failed"
)

// Status is the safe view of a device flow. It never contains the device code
// or tokens.
type Status struct {
	State                   State  `json:"state"`
	UserCode                string `json:"user_code,omitempty"`
	VerificationURI         string `json:"verification_uri,omitempty"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in,omitempty"`
	Error                   string `json:"error,omitempty"`
}

// Manager ensures at most one device flow runs at a time and tracks its state.
// CLI login, Desktop sign-in and the signin_url helper all share the daemon's
// default manager.
type Manager struct {
	mu sync.Mutex

	// newClient builds the platform client for a new flow. It is a field so
	// tests can point the manager at an httptest server.
	newClient func() (*Client, error)

	status Status
	dc     *DeviceCode
	cancel context.CancelFunc
	done   chan struct{}
}

// NewManager creates a manager using newClient to construct platform clients.
func NewManager(newClient func() (*Client, error)) *Manager {
	return &Manager{
		newClient: newClient,
		status:    Status{State: StateIdle},
	}
}

var defaultManager = NewManager(DefaultClient)

// DefaultManager returns the daemon-wide singleton flow manager.
func DefaultManager() *Manager {
	return defaultManager
}

// Status returns the current flow status.
func (m *Manager) CurrentStatus() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

// Start begins a device flow. A live, unexpired pending flow is reused unless
// force is true, in which case it is abandoned and a new device code is
// requested.
func (m *Manager) Start(ctx context.Context, clientName string, force bool) (Status, error) {
	m.mu.Lock()

	if m.status.State == StatePending && m.dc != nil && !m.dc.Expired(time.Now()) && !force {
		status := m.status
		m.mu.Unlock()
		return status, nil
	}

	// Abandon any previous or expired flow and wait for its poller to stop.
	if m.cancel != nil {
		m.cancel()
	}
	if m.done != nil {
		done := m.done
		m.mu.Unlock()
		select {
		case <-done:
		case <-ctx.Done():
			return Status{}, ctx.Err()
		}
		m.mu.Lock()
	}

	client, err := m.newClient()
	if err != nil {
		m.failLocked(err)
		status := m.status
		m.mu.Unlock()
		return status, err
	}

	dc, err := client.StartDeviceFlow(ctx, clientName)
	if err != nil {
		m.failLocked(err)
		status := m.status
		m.mu.Unlock()
		return status, err
	}

	bgCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	m.dc = dc
	m.cancel = cancel
	m.done = done
	m.status = Status{
		State:                   StatePending,
		UserCode:                dc.UserCode,
		VerificationURI:         dc.VerificationURI,
		VerificationURIComplete: dc.VerificationURIComplete,
		ExpiresIn:               dc.ExpiresIn,
	}
	status := m.status

	go m.poll(bgCtx, client, dc, done)

	m.mu.Unlock()
	return status, nil
}

func (m *Manager) failLocked(err error) {
	m.status = Status{
		State: StateFailed,
		Error: err.Error(),
	}
}

// Reset cancels any in-progress flow and returns the manager to idle. It is
// used by sign out so a pending poller cannot rewrite credentials after the
// user has signed out.
func (m *Manager) Reset() {
	m.mu.Lock()

	if m.cancel != nil {
		m.cancel()
	}
	if m.done != nil {
		done := m.done
		m.mu.Unlock()
		<-done
		m.mu.Lock()
	}

	m.dc = nil
	m.cancel = nil
	m.done = nil
	m.status = Status{State: StateIdle}

	m.mu.Unlock()
}

func (m *Manager) poll(ctx context.Context, client *Client, dc *DeviceCode, done chan struct{}) {
	defer close(done)

	tok, err := client.PollToken(ctx, dc)

	m.mu.Lock()
	defer m.mu.Unlock()

	switch {
	case err == nil:
		cred := credentialsFromToken(client, tok, client.DeviceName)
		if werr := SaveCredentials(cred); werr != nil {
			m.failLocked(werr)
			return
		}
		m.status = Status{State: StateAuthorized}
	case errors.Is(err, ErrAccessDenied):
		m.status = Status{
			State:    StateDenied,
			UserCode: dc.UserCode,
			Error:    "authorization was denied",
		}
	case errors.Is(err, ErrDeviceExpired):
		m.status = Status{
			State:    StateExpired,
			UserCode: dc.UserCode,
			Error:    "the verification code has expired; please sign in again",
		}
	default:
		m.status = Status{
			State:    StateFailed,
			UserCode: dc.UserCode,
			Error:    fmt.Sprintf("sign in failed: %v", err),
		}
	}
}
