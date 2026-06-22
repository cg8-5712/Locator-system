import type { RealtimeEnvelope } from "../../types/realtime";

const configuredWsBaseUrl = (import.meta.env.VITE_WS_BASE_URL ?? "").trim();

function resolveWebSocketUrl(token: string): string {
  if (configuredWsBaseUrl) {
    const url = new URL(configuredWsBaseUrl);
    url.searchParams.set("token", token);
    return url.toString();
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const url = new URL(`${protocol}//${window.location.host}/ws`);
  url.searchParams.set("token", token);
  return url.toString();
}

export function connectRealtime(
  token: string,
  handlers: {
    onMessage: (message: RealtimeEnvelope) => void;
    onOpen?: () => void;
    onClose?: () => void;
    onError?: () => void;
  }
) {
  const socket = new WebSocket(resolveWebSocketUrl(token));

  socket.addEventListener("open", () => {
    handlers.onOpen?.();
  });

  socket.addEventListener("message", (event) => {
    try {
      const parsed = JSON.parse(String(event.data)) as RealtimeEnvelope;
      handlers.onMessage(parsed);
    } catch {
      // Ignore malformed transient messages from the broker or reverse proxy.
    }
  });

  socket.addEventListener("close", () => {
    handlers.onClose?.();
  });

  socket.addEventListener("error", () => {
    handlers.onError?.();
  });

  return socket;
}
