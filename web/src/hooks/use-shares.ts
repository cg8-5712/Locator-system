import { useQuery } from "@tanstack/react-query";
import { fetchDeviceShares, fetchShares } from "../services/http/shares";

export function useShares(deviceSN?: string | null) {
  const enabled = deviceSN !== null;

  return useQuery({
    queryKey: ["shares", deviceSN ?? "all"],
    queryFn: () =>
      deviceSN
        ? fetchDeviceShares(deviceSN, { page: 1, pageSize: 100 })
        : fetchShares({ page: 1, pageSize: 100 }),
    enabled,
    staleTime: 15_000,
  });
}
