import type { AlarmSummary } from "../../types/alarm";
import type { DeviceSummary } from "../../types/device";
import type { GeofenceOverlay } from "./map-types";

export interface DeviceAlertState {
  deviceSN: string;
  type: string;
  createdAt: string;
  content: string;
}

const ALARM_PRIORITY: Record<string, number> = {
  sos: 400,
  out_of_fence: 300,
  low_battery: 200,
  offline: 100,
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function readNumber(value: unknown) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function readString(value: unknown) {
  return typeof value === "string" && value.trim() ? value.trim() : null;
}

function getAlarmTimestamp(value: string) {
  const timestamp = new Date(value).getTime();
  return Number.isFinite(timestamp) ? timestamp : 0;
}

function pickGeofenceOverlay(
  value: Record<string, unknown>,
  deviceSN: string
): GeofenceOverlay | null {
  const center = isRecord(value.center) ? value.center : null;
  const centerLat = readNumber(
    value.center_lat ?? value.centerLat ?? center?.lat ?? value.lat ?? value.latitude
  );
  const centerLng = readNumber(
    value.center_lng ??
      value.centerLng ??
      center?.lng ??
      center?.lon ??
      value.lng ??
      value.lon ??
      value.longitude
  );
  const radiusMeters = readNumber(value.radius_m ?? value.radiusMeters ?? value.radius);

  if (
    centerLat === null ||
    centerLng === null ||
    radiusMeters === null ||
    radiusMeters <= 0
  ) {
    return null;
  }

  return {
    deviceSN,
    name: readString(value.name ?? value.label),
    centerLat,
    centerLng,
    radiusMeters,
  };
}

function readGeofenceOverlay(device: DeviceSummary): GeofenceOverlay | null {
  const payloads = [device.status_payload, device.config_payload];

  for (const payload of payloads) {
    if (!payload) {
      continue;
    }

    const nestedFence = isRecord(payload.geofence)
      ? pickGeofenceOverlay(payload.geofence, device.device_sn)
      : null;
    if (nestedFence) {
      return nestedFence;
    }

    const nestedLegacyFence = isRecord(payload.fence)
      ? pickGeofenceOverlay(payload.fence, device.device_sn)
      : null;
    if (nestedLegacyFence) {
      return nestedLegacyFence;
    }

    const flatFence = pickGeofenceOverlay(
      {
        name: payload.fence_name ?? payload.geofence_name,
        center_lat:
          payload.fence_center_lat ??
          payload.geofence_center_lat ??
          payload.fence_lat ??
          payload.geofence_lat,
        center_lng:
          payload.fence_center_lng ??
          payload.geofence_center_lng ??
          payload.fence_lng ??
          payload.fence_lon ??
          payload.geofence_lng ??
          payload.geofence_lon,
        radius_m:
          payload.fence_radius_m ??
          payload.geofence_radius_m ??
          payload.fence_radius ??
          payload.geofence_radius,
      },
      device.device_sn
    );

    if (flatFence) {
      return flatFence;
    }
  }

  return null;
}

export function getAlarmPriority(type?: string | null) {
  return type ? ALARM_PRIORITY[type] ?? 0 : 0;
}

export function buildMapDecorations(devices: DeviceSummary[], alarms: AlarmSummary[]) {
  const alertStateByDeviceSN = new Map<string, DeviceAlertState>();
  const fenceAlarmDeviceSNs = new Set<string>();

  for (const alarm of alarms) {
    const nextAlert: DeviceAlertState = {
      deviceSN: alarm.device_sn,
      type: alarm.type,
      createdAt: alarm.created_at,
      content: alarm.content,
    };
    const currentAlert = alertStateByDeviceSN.get(alarm.device_sn);

    if (
      !currentAlert ||
      getAlarmPriority(nextAlert.type) > getAlarmPriority(currentAlert.type) ||
      (getAlarmPriority(nextAlert.type) === getAlarmPriority(currentAlert.type) &&
        getAlarmTimestamp(nextAlert.createdAt) > getAlarmTimestamp(currentAlert.createdAt))
    ) {
      alertStateByDeviceSN.set(alarm.device_sn, nextAlert);
    }

    if (alarm.type === "out_of_fence") {
      fenceAlarmDeviceSNs.add(alarm.device_sn);
    }
  }

  const geofenceOverlays = devices.flatMap((device) => {
    if (!fenceAlarmDeviceSNs.has(device.device_sn)) {
      return [];
    }

    const overlay = readGeofenceOverlay(device);
    return overlay ? [overlay] : [];
  });

  const emergencyDeviceSNs = new Set<string>();
  const prioritizedFenceAlertDeviceSNs = new Set<string>();

  for (const alertState of alertStateByDeviceSN.values()) {
    if (alertState.type === "sos") {
      emergencyDeviceSNs.add(alertState.deviceSN);
    }

    if (alertState.type === "out_of_fence") {
      prioritizedFenceAlertDeviceSNs.add(alertState.deviceSN);
    }
  }

  return {
    alertStateByDeviceSN,
    emergencyDeviceSNs,
    fenceAlertDeviceSNs: prioritizedFenceAlertDeviceSNs,
    geofenceOverlays,
  };
}
