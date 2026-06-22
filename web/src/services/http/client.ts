import type { ApiEnvelope } from "../../types/api";
import { authStore } from "../../stores/auth-store";

const configuredBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? "").trim();
const API_BASE_URL = configuredBaseUrl || "";

export class HttpError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export async function apiRequest<T>(
  input: string,
  init?: RequestInit
): Promise<T> {
  const token = authStore.getState().token;
  const headers = new Headers(init?.headers);
  headers.set("Accept", "application/json");

  const hasBody = init?.body !== undefined && init?.body !== null;
  if (hasBody && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE_URL}${input}`, {
    ...init,
    headers,
  });

  const text = await response.text();
  const payload = text ? (JSON.parse(text) as ApiEnvelope<T>) : null;

  if (!response.ok || !payload?.success) {
    const message =
      payload?.error || `request failed with status ${response.status}`;
    throw new HttpError(response.status, message);
  }

  return payload.data;
}
