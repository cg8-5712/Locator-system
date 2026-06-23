import { useQuery } from "@tanstack/react-query";
import { fetchAlarms } from "../services/http/alarms";

export function useAlarmList(options?: {
  deviceSN?: string | null;
  type?: string;
  pageSize?: number;
}) {
  return useQuery({
    queryKey: ["alarms", options?.deviceSN ?? "", options?.type ?? "", options?.pageSize ?? 50],
    queryFn: () =>
      fetchAlarms({
        deviceSN: options?.deviceSN ?? undefined,
        type: options?.type,
        page: 1,
        pageSize: options?.pageSize ?? 50,
      }),
    staleTime: 15_000,
  });
}
