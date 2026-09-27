import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  type DeviceSigninStatus,
  getDeviceSigninStatus,
  startDeviceSignin,
} from "@/api";

// useDeviceSignin starts the daemon's device flow and polls the local UI
// service every 2 seconds until the flow reaches a terminal state.
export function useDeviceSignin() {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<DeviceSigninStatus | null>(null);
  const [error, setError] = useState<string | null>(null);
  const pollRef = useRef<number | null>(null);

  const stopPolling = useCallback(() => {
    if (pollRef.current !== null) {
      window.clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }, []);

  const begin = useCallback(
    async (force = false): Promise<DeviceSigninStatus | null> => {
      setError(null);
      try {
        const initial = await startDeviceSignin(force);
        setStatus(initial);
        if (initial.state !== "pending") {
          return initial;
        }

        stopPolling();
        pollRef.current = window.setInterval(async () => {
          try {
            const next = await getDeviceSigninStatus();
            setStatus(next);
            if (next.state !== "pending") {
              stopPolling();
              if (next.state === "authorized") {
                void queryClient.invalidateQueries({ queryKey: ["user"] });
              }
            }
          } catch (e) {
            setError(e instanceof Error ? e.message : String(e));
            stopPolling();
          }
        }, 2000);
        return initial;
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
        return null;
      }
    },
    [queryClient, stopPolling],
  );

  const reset = useCallback(() => {
    stopPolling();
    setStatus(null);
    setError(null);
  }, [stopPolling]);

  useEffect(() => stopPolling, [stopPolling]);

  return {
    begin,
    reset,
    status,
    error,
    isPending: status?.state === "pending",
  };
}
