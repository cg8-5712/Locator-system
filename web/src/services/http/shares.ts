import type {
  PublicLocationResult,
  PublicShareSummary,
  PublicTrackResult,
  ShareCreateResult,
  ShareListResult,
  ShareVerifyResult,
} from "../../types/share";
import { apiRequest } from "./client";

export async function fetchShares(params?: {
  deviceSN?: string;
  page?: number;
  pageSize?: number;
}) {
  const query = new URLSearchParams();
  if (params?.deviceSN) {
    query.set("device_sn", params.deviceSN);
  }
  query.set("page", String(params?.page ?? 1));
  query.set("page_size", String(params?.pageSize ?? 100));

  return apiRequest<ShareListResult>(`/api/shares?${query.toString()}`);
}

export async function fetchDeviceShares(deviceSN: string, params?: { page?: number; pageSize?: number }) {
  const query = new URLSearchParams();
  query.set("page", String(params?.page ?? 1));
  query.set("page_size", String(params?.pageSize ?? 100));

  return apiRequest<ShareListResult>(
    `/api/devices/${deviceSN}/shares?${query.toString()}`
  );
}

export async function createShare(
  deviceSN: string,
  input: {
    share_mode: "live_only" | "today_track";
    password?: string | null;
    expires_at: string;
    max_visits?: number | null;
    note?: string;
  }
) {
  return apiRequest<ShareCreateResult>(`/api/devices/${deviceSN}/shares`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function revokeShare(shareID: number) {
  return apiRequest<{ deleted: boolean; share_id: number }>(`/api/shares/${shareID}`, {
    method: "DELETE",
  });
}

export async function fetchPublicShare(shareCode: string) {
  return apiRequest<PublicShareSummary>(`/api/public/shares/${shareCode}`);
}

export async function verifyPublicShare(
  shareCode: string,
  input: {
    viewer_id: string;
    password?: string;
  }
) {
  return apiRequest<ShareVerifyResult>(`/api/public/shares/${shareCode}/verify`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

function withAccessToken(path: string, accessToken: string) {
  const query = new URLSearchParams();
  query.set("access_token", accessToken);
  return `${path}?${query.toString()}`;
}

export async function fetchPublicLocation(shareCode: string, accessToken: string) {
  return apiRequest<PublicLocationResult>(
    withAccessToken(`/api/public/shares/${shareCode}/location`, accessToken)
  );
}

export async function fetchPublicTrack(shareCode: string, accessToken: string) {
  return apiRequest<PublicTrackResult>(
    withAccessToken(`/api/public/shares/${shareCode}/track`, accessToken)
  );
}

export function buildPublicShareWsUrl(shareCode: string, accessToken: string) {
  const configuredWsBaseUrl = (import.meta.env.VITE_WS_BASE_URL ?? "").trim();
  const baseUrl = configuredWsBaseUrl
    ? new URL(configuredWsBaseUrl)
    : new URL(
        `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/api/public/shares/${shareCode}/ws`
      );

  if (!configuredWsBaseUrl) {
    baseUrl.searchParams.set("access_token", accessToken);
    return baseUrl.toString();
  }

  const path = `/api/public/shares/${shareCode}/ws`;
  baseUrl.pathname = path;
  baseUrl.searchParams.set("access_token", accessToken);
  return baseUrl.toString();
}
