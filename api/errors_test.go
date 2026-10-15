package api

import (
	"errors"
	"net/http"
	"testing"
)

func TestUsageLimitHit(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want UsageUpgradeClass
	}{
		{
			name: "402 status error → payment required",
			err: StatusError{
				StatusCode:   http.StatusPaymentRequired,
				Status:       "402 Payment Required",
				ErrorMessage: "Insufficient credit balance",
			},
			want: UsageUpgradePaymentRequired,
		},
		{
			name: "402 in plain error string → payment required",
			err:  errors.New("payment required 402"),
			want: UsageUpgradePaymentRequired,
		},
		{
			name: "403 susancloud non-reserved → susancloud forbidden",
			err: StatusError{
				StatusCode:   http.StatusForbidden,
				Status:       "403 Forbidden",
				ErrorMessage: "Pro plan required for susancloud models",
			},
			want: UsageUpgradeSusancloudForbidden,
		},
		{
			name: "403 susancloud reserved (is reserved) → no hint",
			err: StatusError{
				StatusCode:   http.StatusForbidden,
				ErrorMessage: `namespace "susancloud" is reserved`,
			},
			want: "",
		},
		{
			name: "403 non-susancloud → no hint",
			err: StatusError{
				StatusCode:   http.StatusForbidden,
				ErrorMessage: "forbidden",
			},
			want: "",
		},
		{
			name: "429 rate limit → no hint",
			err: StatusError{
				StatusCode:   http.StatusTooManyRequests,
				ErrorMessage: "Rate limit exceeded",
			},
			want: "",
		},
		{
			name: "nil error → no hint",
			err:  nil,
			want: "",
		},
		{
			name: "plain unrelated error → no hint",
			err:  errors.New("something went wrong"),
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := UsageLimitHit(tc.err)
			if got != tc.want {
				t.Fatalf("UsageLimitHit() = %q, want %q", got, tc.want)
			}
		})
	}
}
