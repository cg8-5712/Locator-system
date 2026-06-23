export type AppMode = "live" | "demo";

export function getModeBasePath(mode: AppMode) {
  return mode === "demo" ? "/demo" : "/app";
}

export function buildModePath(mode: AppMode, path: string) {
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return `${getModeBasePath(mode)}${normalized}`;
}

export function getModeLabel(mode: AppMode) {
  return mode === "demo"
    ? "\u6b7b\u6570\u636e\u9a8c\u8bc1"
    : "\u540e\u7aef\u8054\u8c03";
}
