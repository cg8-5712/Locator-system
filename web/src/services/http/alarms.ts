import type { AlarmListResult } from "../../types/alarm";
import { apiRequest } from "./client";

export async function fetchAlarms(params?: {
  deviceSN?: string;
  type?: string;
  page?: number;
  pageSize?: number;
  startTime?: string;
  endTime?: string;
}) {
  const query = new URLSearchParams();
  if (params?.deviceSN) {
    query.set("device_sn", params.deviceSN);
  }
  if (params?.type) {
    query.set("type", params.type);
  }
  if (params?.startTime) {
    query.set("start_time", params.startTime);
  }
  if (params?.endTime) {
    query.set("end_time", params.endTime);
  }
  query.set("page", String(params?.page ?? 1));
  query.set("page_size", String(params?.pageSize ?? 50));

  return apiRequest<AlarmListResult>(`/api/alarms?${query.toString()}`);
}
