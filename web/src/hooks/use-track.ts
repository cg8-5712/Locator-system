import { useQuery } from "@tanstack/react-query";
import { fetchDeviceTrack } from "../services/http/devices";

export function useTrack(deviceSN: string | null, rangeHours: number) {
  return useQuery({
    queryKey: ["track", deviceSN, rangeHours],
    queryFn: () => {
      const end = new Date();
      const start = new Date(end.getTime() - rangeHours * 60 * 60 * 1000);

      return fetchDeviceTrack(deviceSN!, {
        startTime: start.toISOString(),
        endTime: end.toISOString(),
        page: 1,
        pageSize: 500,
      });
    },
    enabled: Boolean(deviceSN),
    staleTime: 10_000,
  });
}
