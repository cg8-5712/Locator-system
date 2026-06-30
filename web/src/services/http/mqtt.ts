import type { MQTTMessageListResult, MQTTStatus } from "../../types/mqtt";
import { apiRequest } from "./client";

export async function fetchMQTTStatus() {
  return apiRequest<MQTTStatus>("/api/mqtt/status");
}

export async function fetchMQTTMessages(limit = 20) {
  return apiRequest<MQTTMessageListResult>(`/api/mqtt/messages?limit=${limit}`);
}
