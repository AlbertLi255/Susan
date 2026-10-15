package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/cmd/launch"
)

const cloudUsageLimitMessage = "You've reached your usage limit, please upgrade to continue"

// withCloudUpgradeHint appends the Desktop usage-limit hint to cloud inference
// errors that need an upgrade or top-up. The platform message is kept and the
// original error stays reachable through errors.As.
func withCloudUpgradeHint(ctx context.Context, err error) error {
	url := cloudUpgradeURL(ctx, err, currentCloudPlan)
	if url == "" {
		return err
	}
	return fmt.Errorf("%w\n\n%s:\n    %s", err, cloudUsageLimitMessage, url)
}

func cloudUpgradeURL(ctx context.Context, err error, plan func(context.Context) string) string {
	class := api.UsageLimitHit(err)
	switch class {
	case api.UsageUpgradePaymentRequired:
		if launch.PlanSatisfies(plan(ctx), "pro") {
			return launch.DefaultBillingURL
		}
		return launch.DefaultUpgradeURL
	case api.UsageUpgradeSusancloudForbidden:
		return launch.DefaultUpgradeURL
	}
	return ""
}

func currentCloudPlan(ctx context.Context) string {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	user, err := client.Whoami(ctx)
	if err != nil || user == nil {
		return ""
	}
	return user.Plan
}
