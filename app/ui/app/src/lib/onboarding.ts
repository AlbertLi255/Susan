// Keep in sync with store.CurrentOnboardingVersion in app/store/store.go.
export const CURRENT_ONBOARDING_VERSION = 1;

export type OnboardingStep = "intro" | "welcome" | "apps" | "run";

export type OnboardingAction = "continue" | "authenticated" | "local";

export function nextOnboardingStep(
  step: OnboardingStep,
  action: OnboardingAction,
  isAuthenticated: boolean,
): OnboardingStep {
  if (action === "local") return "run";
  if (step === "intro" && action === "continue") {
    return isAuthenticated ? "apps" : "welcome";
  }
  if (step === "welcome" && action === "authenticated") return "apps";
  return step;
}

export function homeChatId(): "new" {
  return "new";
}
