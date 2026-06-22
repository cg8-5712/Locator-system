import type { DeviceListResult, DeviceSummary } from "../../types/device";
import { apiRequest } from "./client";

export async function fetchDevices(params?: {
  device_sn?: string;
  status?: number;
  page?: number;
  page_size?: number;
}) {
  const query = new URLSearchParams();
  if (params?.device_sn) {
    query.set("device_sn", params.device_sn);
  }
  if (typeof params?.status === "number") {
    query.set("status", String(params.status));
  }
  query.set("page", String(params?.page ?? 1));
  query.set("page_size", String(params?.page_size ?? 100));

  return apiRequest<DeviceListResult>(`/api/devices?${query.toString()}`);
}

export async function fetchDevice(deviceSN: string) {
  return apiRequest<DeviceSummary>(`/api/devices/${deviceSN}`);
}
