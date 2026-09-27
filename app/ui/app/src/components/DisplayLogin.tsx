import type { ErrorEvent } from "@/gotypes";
import { Display, type DisplayAction } from "@/components/ui/display";
import { useUser } from "@/hooks/useUser";
import { useDeviceSignin } from "@/hooks/useDeviceSignin";
import { useEffect } from "react";

interface DisplayLoginProps {
  error: ErrorEvent | null;
  className?: string;
  onDismiss?: () => void;
  message?: string;
}

export const DisplayLogin = ({
  error,
  className,
  onDismiss,
  message,
}: DisplayLoginProps) => {
  const { isAuthenticated } = useUser();
  const deviceSignin = useDeviceSignin();

  useEffect(() => {
    if (isAuthenticated && deviceSignin.isPending) {
      if (onDismiss) {
        onDismiss();
      }
    }
  }, [deviceSignin.isPending, isAuthenticated, onDismiss]);

  if (!error || error.code !== "cloud_unauthorized" || isAuthenticated)
    return null;

  const handleSignIn = async () => {
    const initial = await deviceSignin.begin();
    if (initial?.verification_uri_complete) {
      window.open(initial.verification_uri_complete, "_blank");
    }
  };

  const action: DisplayAction = {
    label: "Sign In",
    onClick: handleSignIn,
  };

  const state = deviceSignin.status?.state;

  return (
    <Display
      message={message || "Cloud models require a Susan account"}
      action={action}
      className={className}
      onDismiss={onDismiss}
    >
      {deviceSignin.isPending && (
        <p role="status" aria-live="polite" className="mt-2 text-sm">
          Confirm verification code{" "}
          <span className="font-mono font-medium">
            {deviceSignin.status?.user_code}
          </span>{" "}
          in your browser.
        </p>
      )}
      {state === "denied" && (
        <p role="alert" className="mt-2 text-sm text-red-600">
          Authorization was denied. Please try again.
        </p>
      )}
      {state === "expired" && (
        <p role="alert" className="mt-2 text-sm text-red-600">
          The verification code has expired. Please try again.
        </p>
      )}
    </Display>
  );
};
