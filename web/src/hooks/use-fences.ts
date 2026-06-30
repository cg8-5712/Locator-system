import { useQuery } from "@tanstack/react-query";
import { fetchFences } from "../services/http/fences";

export function useFences(deviceSN: string | null) {
  return useQuery({
    queryKey: ["fences", deviceSN],
    queryFn: () => fetchFences(deviceSN!),
    enabled: Boolean(deviceSN),
    staleTime: 15_000,
  });
}
