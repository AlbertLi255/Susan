package api

import (
	"errors"
	"net/http"
	"strings"
)

// UsageUpgradeClass identifies which platform-side error condition should surface
// an upgrade / top-up prompt. An empty string means "not a usage-limit case".
//
// The mapping lives here so Desktop (app/ui) and CLI (cmd) can share one source
// of truth; new platform error codes only need one entry added to classifyUsageLimit.
type UsageUpgradeClass string

const (
	// UsageUpgradePaymentRequired means the platform returned 402 Payment Required.
	UsageUpgradePaymentRequired UsageUpgradeClass = "payment_required"
	// UsageUpgradeSusancloudForbidden means the platform returned 403 Forbidden for
	// a susancloud/* namespace that is reserved (push) — only the non-reserved
	// "pro plan required for susancloud models" case qualifies.
	UsageUpgradeSusancloudForbidden UsageUpgradeClass = "susancloud_forbidden"
)

// UsageLimitHit returns the UsageUpgradeClass if err is a cloud-side error that
// should prompt the user to upgrade or top up, or "" otherwise.
//
// It accepts both typed *StatusError and string-form errors so wrapped errors
// (e.g. fmt.Errorf("%w", StatusError{...})) still classify correctly regardless
// of whether errors.As unwinds them.
func UsageLimitHit(err error) UsageUpgradeClass {
	if err == nil {
		return ""
	}
	return classifyUsageLimit(err.Error(), extractStatusCode(err))
}

// extractStatusCode pulls the HTTP status code out of err if it carries a
// *StatusError. Returns 0 if err is not a StatusError.
func extractStatusCode(err error) int {
	var s StatusError
	if errors.As(err, &s) {
		return s.StatusCode
	}
	return 0
}

// classifyUsageLimit is the single table that maps (status code, error string)
// to a UsageUpgradeClass. Extend this list when the platform introduces new
// 4xx error codes that should surface an upgrade hint.
func classifyUsageLimit(errStr string, statusCode int) UsageUpgradeClass {
	// 402 Payment Required → always upgrade (余额不足 / 套餐需升级)
	if statusCode == http.StatusPaymentRequired || strings.Contains(errStr, "402") {
		return UsageUpgradePaymentRequired
	}

	// 403 Forbidden + susancloud namespace + NOT "is reserved" → upgrade
	// (Pro plan required for susancloud models / susancloud 模型需 Pro 档)
	// The "is reserved" guard keeps legitimate namespace reservation 403s out
	// of the upgrade path (e.g. push to susancloud/something).
	if (statusCode == http.StatusForbidden || strings.Contains(errStr, "403")) &&
		strings.Contains(errStr, "susancloud") &&
		!strings.Contains(errStr, "is reserved") {
		return UsageUpgradeSusancloudForbidden
	}

	return ""
}
