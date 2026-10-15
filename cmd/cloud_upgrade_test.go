package cmd

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/cmd/launch"
)

func TestCloudUpgradeURL(t *testing.T) {
	plan := func(p string) func(context.Context) string {
		return func(context.Context) string { return p }
	}

	cases := []struct {
		name string
		err  error
		plan string
		want string
	}{
		{
			name: "402 free plan upgrades",
			err:  api.StatusError{StatusCode: http.StatusPaymentRequired, Status: "402 Payment Required", ErrorMessage: "Insufficient credit balance / 余额不足，请充值"},
			plan: "free",
			want: launch.DefaultUpgradeURL,
		},
		{
			name: "402 unknown plan upgrades",
			err:  api.StatusError{StatusCode: http.StatusPaymentRequired, ErrorMessage: "Insufficient credit balance"},
			want: launch.DefaultUpgradeURL,
		},
		{
			name: "402 pro plan tops up",
			err:  api.StatusError{StatusCode: http.StatusPaymentRequired, ErrorMessage: "Insufficient credit balance"},
			plan: "pro",
			want: launch.DefaultBillingURL,
		},
		{
			name: "403 susancloud upgrades",
			err:  api.StatusError{StatusCode: http.StatusForbidden, ErrorMessage: "Pro plan required for susancloud models / susancloud 模型需 Pro 档"},
			plan: "free",
			want: launch.DefaultUpgradeURL,
		},
		{
			name: "403 reserved name has no hint",
			err:  api.StatusError{StatusCode: http.StatusForbidden, ErrorMessage: "model name susancloud/x is reserved"},
		},
		{
			name: "403 other has no hint",
			err:  api.StatusError{StatusCode: http.StatusForbidden, ErrorMessage: "forbidden"},
		},
		{
			name: "429 has no hint",
			err:  api.StatusError{StatusCode: http.StatusTooManyRequests, ErrorMessage: "Rate limit exceeded / 请求过于频繁"},
		},
		{
			// plain error with no magic strings → no hint. Note: an error whose
			// text literally contains "402" or "403+susancloud" will hit the
			// same table-driven classification that Desktop uses; CLI and
			// Desktop are intentionally aligned here.
			name: "plain error has no hint",
			err:  errors.New("something went wrong"),
		},
		{
			name: "nil error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cloudUpgradeURL(t.Context(), tc.err, plan(tc.plan))
			if got != tc.want {
				t.Fatalf("cloudUpgradeURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPrintSigninInstructions(t *testing.T) {
	out := captureStdout(t, func() {
		printSigninInstructions("http://localhost:3000/device?user_code=WQW5-ZYP9")
	})
	for _, want := range []string{
		"Open this link to sign in, then run the command again:",
		"http://localhost:3000/device?user_code=WQW5-ZYP9",
		"Verification code: WQW5-ZYP9",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q does not contain %q", out, want)
		}
	}
	if strings.Contains(out, "browser") {
		t.Fatalf("CLI does not open a browser here, got %q", out)
	}

	out = captureStdout(t, func() { printSigninInstructions("https://ollama.com/connect?key=abc") })
	if strings.Contains(out, "Verification code") {
		t.Fatalf("no user_code in link, got %q", out)
	}
}

func TestWithCloudUpgradeHintKeepsPlatformMessage(t *testing.T) {
	t.Setenv("SUSAN_HOST", "http://127.0.0.1:1")

	original := api.StatusError{StatusCode: http.StatusPaymentRequired, Status: "402 Payment Required", ErrorMessage: "Insufficient credit balance / 余额不足，请充值"}
	err := withCloudUpgradeHint(t.Context(), original)

	msg := err.Error()
	for _, want := range []string{original.Error(), cloudUsageLimitMessage, launch.DefaultUpgradeURL} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not contain %q", msg, want)
		}
	}
	var statusErr api.StatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("wrapped error lost StatusError: %v", err)
	}

	rate := api.StatusError{StatusCode: http.StatusTooManyRequests, ErrorMessage: "Rate limit exceeded"}
	if got := withCloudUpgradeHint(t.Context(), rate); got != error(rate) {
		t.Fatalf("429 should be returned unchanged, got %v", got)
	}
}
