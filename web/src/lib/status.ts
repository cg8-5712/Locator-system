import type { DeviceSummary } from "../types/device";

export interface DeviceStateView {
  label: string;
  tone: "safe" | "brand" | "warn" | "danger" | "offline";
}

export function getDeviceStateView(
  device: Pick<DeviceSummary, "status" | "gps_state" | "battery">
): DeviceStateView {
  if (device.status === 0 || device.gps_state === "offline") {
    return { label: "离线", tone: "offline" };
  }

  if (device.battery > 0 && device.battery <= 15) {
    return { label: "低电量", tone: "warn" };
  }

  switch (device.gps_state) {
    case "located":
      return { label: "已定位", tone: "safe" };
    case "searching":
      return { label: "搜星中", tone: "brand" };
    case "unable":
      return { label: "无定位", tone: "warn" };
    case "not_started":
      return { label: "未启动", tone: "offline" };
    default:
      return { label: "在线", tone: "brand" };
  }
}

export function getMarkerAccent(
  device: Pick<DeviceSummary, "status" | "gps_state" | "battery">
): string {
  const tone = getDeviceStateView(device).tone;
  switch (tone) {
    case "safe":
      return "#2f9e68";
    case "warn":
      return "#d48a1f";
    case "danger":
      return "#d94747";
    case "offline":
      return "#7c8b94";
    default:
      return "#1f88c9";
  }
}

export function getGPSStateLabel(state?: string) {
  switch (state) {
    case "not_started":
      return "未启动";
    case "offline":
      return "离线";
    case "searching":
      return "搜星中";
    case "located":
      return "已定位";
    case "unable":
      return "无定位";
    default:
      return "未知";
  }
}

export function getAlarmTypeView(type: string) {
  switch (type) {
    case "sos":
      return { label: "SOS 求救", tone: "danger" as const };
    case "low_battery":
      return { label: "低电量", tone: "warn" as const };
    case "out_of_fence":
      return { label: "围栏告警", tone: "brand" as const };
    case "offline":
      return { label: "离线恢复", tone: "offline" as const };
    default:
      return { label: type, tone: "brand" as const };
  }
}

export function getActivityLabel(activity: unknown) {
  if (typeof activity !== "string") {
    return null;
  }

  switch (activity) {
    case "walking":
      return "步行中";
    case "running":
      return "快速移动";
    case "still":
      return "静止";
    case "sos":
      return "紧急求救";
    default:
      return activity;
  }
}
