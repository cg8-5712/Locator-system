import type { FenceListResult, FenceSummary } from "../../types/fence";
import { apiRequest } from "./client";

export async function fetchFences(deviceSN: string) {
  return apiRequest<FenceListResult>(`/api/devices/${deviceSN}/fences`);
}

export async function createFence(
  deviceSN: string,
  input: {
    name: string;
    polygon: { lat: number; lng: number }[];
  }
) {
  return apiRequest<FenceSummary>(`/api/devices/${deviceSN}/fences`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateFence(
  deviceSN: string,
  fenceID: number,
  input: {
    name: string;
    polygon: { lat: number; lng: number }[];
  }
) {
  return apiRequest<FenceSummary>(`/api/devices/${deviceSN}/fences/${fenceID}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteFence(deviceSN: string, fenceID: number) {
  return apiRequest<{ deleted: boolean; fence_id: number }>(
    `/api/devices/${deviceSN}/fences/${fenceID}`,
    {
      method: "DELETE",
    }
  );
}
