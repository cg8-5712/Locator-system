import { useQuery } from "@tanstack/react-query";
import { fetchMQTTMessages, fetchMQTTStatus } from "../services/http/mqtt";

export function useMQTTStatus() {
  return useQuery({
    queryKey: ["mqtt-status"],
    queryFn: fetchMQTTStatus,
    staleTime: 10_000,
    refetchInterval: 10_000,
  });
}

export function useMQTTMessages(limit = 20) {
  return useQuery({
    queryKey: ["mqtt-messages", limit],
    queryFn: () => fetchMQTTMessages(limit),
    staleTime: 5_000,
    refetchInterval: 5_000,
  });
}
