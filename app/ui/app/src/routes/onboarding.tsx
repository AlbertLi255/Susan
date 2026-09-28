import Onboarding from "@/components/Onboarding";
import { getSettings } from "@/api";
import { useSettings } from "@/hooks/useSettings";
import { useUser } from "@/hooks/useUser";
import { useDeviceSignin } from "@/hooks/useDeviceSignin";
import {
  CURRENT_ONBOARDING_VERSION,
  homeChatId,
} from "@/lib/onboarding";
import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { useCallback, useEffect, useRef, useState } from "react";

export const Route = createFileRoute("/onboarding")({
  beforeLoad: async ({ context }) => {
    // Let developers review onboarding without resetting their local app data.
    if (
      import.meta.env.DEV &&
      new URLSearchParams(window.location.search).get("preview") === "1"
    ) {
      return;
    }

    const settingsData = await context.queryClient.ensureQueryData({
      queryKey: ["settings"],
      queryFn: getSettings,
    });

    if (settingsData.settings.OnboardingVersion >= CURRENT_ONBOARDING_VERSION) {
      const chatId = homeChatId();
      throw redirect({
        to: "/c/$chatId",
        params: { chatId },
        mask: { to: "/" },
      });
    }
  },
  component: OnboardingRoute,
});

function OnboardingRoute() {
  const navigate = useNavigate();
  const { settingsData, setSettings } = useSettings();
  const { isAuthenticated } = useUser();
  const deviceSignin = useDeviceSignin();
  const [isAwaitingAuth, setIsAwaitingAuth] = useState(false);
  const [signInError, setSignInError] = useState<string | null>(null);
  const [signInCode, setSignInCode] = useState<string | null>(null);
  const [completionError, setCompletionError] = useState<string | null>(null);
  const authAttemptRef = useRef(0);

  const completeOnboarding = useCallback(async (): Promise<boolean> => {
    setCompletionError(null);

    try {
      if (!settingsData) {
        throw new Error("Settings are not loaded");
      }

      await setSettings({
        OnboardingVersion: CURRENT_ONBOARDING_VERSION,
      });
      return true;
    } catch (error) {
      console.error("Failed to save onboarding state:", error);
      setCompletionError("Unable to save setup. Please try again.");
      return false;
    }
  }, [setSettings, settingsData]);

  const finishSetup = useCallback(() => {
    void completeOnboarding();
  }, [completeOnboarding]);

  const openApps = useCallback(async (): Promise<boolean> => {
    if (!(await completeOnboarding())) return false;
    await navigate({ to: "/connect" });
    return true;
  }, [completeOnboarding, navigate]);

  const retryCompletion = useCallback(() => {
    void completeOnboarding();
  }, [completeOnboarding]);

  const authenticate = useCallback(async () => {
    setSignInError(null);
    setSignInCode(null);

    if (isAuthenticated) {
      return;
    }

    const authAttempt = ++authAttemptRef.current;
    setIsAwaitingAuth(true);

    const initial = await deviceSignin.begin();
    if (authAttempt !== authAttemptRef.current) return;

    if (!initial) {
      setIsAwaitingAuth(false);
      setSignInError("Unable to start sign in. Please try again.");
      return;
    }

    setSignInCode(initial.user_code ?? null);
    if (initial.verification_uri_complete) {
      const url = initial.verification_uri_complete;
      if (window.openURL) void window.openURL(url);
      else window.open(url, "_blank");
    }
  }, [deviceSignin, isAuthenticated]);

  const signIn = useCallback(() => authenticate(), [authenticate]);
  const signUp = useCallback(() => authenticate(), [authenticate]);

  const useLocal = useCallback(() => {
    authAttemptRef.current += 1;
    deviceSignin.reset();
    setIsAwaitingAuth(false);
    setSignInError(null);
    setSignInCode(null);
    finishSetup();
  }, [deviceSignin, finishSetup]);

  // Map the device flow's terminal states to UI messages. "authorized" is
  // handled by the user query invalidation inside the hook.
  useEffect(() => {
    const state = deviceSignin.status?.state;
    if (!state || state === "pending") return;

    if (state === "denied") {
      setIsAwaitingAuth(false);
      setSignInError("Authorization was denied. Please try again.");
    } else if (state === "expired") {
      setIsAwaitingAuth(false);
      setSignInError("The verification code has expired. Please try again.");
    } else if (state === "failed") {
      setIsAwaitingAuth(false);
      setSignInError("Sign in failed. Please try again.");
    }
  }, [deviceSignin.status]);

  useEffect(() => {
    if (isAuthenticated && isAwaitingAuth) {
      setIsAwaitingAuth(false);
    }
  }, [isAuthenticated, isAwaitingAuth]);

  return (
    <Onboarding
      completionError={completionError}
      isAuthenticated={isAuthenticated}
      isSigningIn={isAwaitingAuth}
      signInError={signInError}
      signInCode={signInCode}
      onOpenApps={openApps}
      onSignIn={signIn}
      onSignUp={signUp}
      onRetryCompletion={retryCompletion}
      onUseLocal={useLocal}
    />
  );
}
