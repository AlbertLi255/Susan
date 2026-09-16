import { useQuery } from "@tanstack/react-query";
import { fetchHealth } from "@/api";

export function healthRefetchInterval(query: {
  state: { data?: boolean };
}): number {
  // Failed requests leave data undefined; an unhealthy server returns false.
  // Poll until healthy so the UI can leave Loading without a window restart.
  // Keep a slower poll after success so a later server crash still self-heals.
  return query.state.data === true ? 5000 : 1000;
}

export function useHealth() {
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: healthRefetchInterval,
    refetchIntervalInBackground: true,
    retry: true,
    staleTime: 0, // Always consider stale so we keep polling
  });

  return {
    isHealthy: healthQuery.data ?? false,
    isChecking: healthQuery.isLoading,
  };
}
