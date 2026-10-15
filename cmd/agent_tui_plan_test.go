package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollama/ollama/api"
	agentchat "github.com/ollama/ollama/cmd/tui/chat"
)

func newPlanTestClient(t *testing.T, meBody string) *api.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/me":
			fmt.Fprint(w, meBody)
		case "/api/status":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error":"not found"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("SUSAN_HOST", srv.URL)
	t.Setenv("SUSAN_NO_CLOUD", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	client, err := api.ClientFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestCloudAvailabilityBadgesEmptyPlan(t *testing.T) {
	options := []agentchat.ModelOption{{Name: "kimi-k2.6:cloud", Cloud: true, RequiredPlan: "pro"}}

	tests := []struct {
		name   string
		meBody string
		want   string
	}{
		{"empty plan shows no upgrade badge", `{"name":"test-user"}`, ""},
		{"free plan shows upgrade badge", `{"name":"test-user","plan":"free"}`, "Upgrade required"},
		{"pro plan shows no badge", `{"name":"test-user","plan":"pro"}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newPlanTestClient(t, tt.meBody)
			badges, _ := cloudAvailabilityBadges(context.Background(), client, options)
			if got := badges["kimi-k2.6:cloud"]; got != tt.want {
				t.Fatalf("badge = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnsureCloudModelAccessEmptyPlan(t *testing.T) {
	client := newPlanTestClient(t, `{"name":"test-user"}`)
	if err := ensureCloudModelAccess(context.Background(), client, "kimi-k2.6:cloud", "pro"); err != nil {
		t.Fatalf("empty plan should be allowed, got %v", err)
	}

	client = newPlanTestClient(t, `{"name":"test-user","plan":"free"}`)
	if err := ensureCloudModelAccess(context.Background(), client, "kimi-k2.6:cloud", "pro"); err == nil {
		t.Fatal("free plan should require upgrade")
	}
}
