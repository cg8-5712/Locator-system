import { isValidCoordinate } from "../../lib/geo";
import type { DeviceSummary } from "../../types/device";
import type { LiveDevicePoint, LiveLocationState } from "./map-types";

function toNumber(value: unknown) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

export function deriveLivePoints(
  devices: DeviceSummary[],
  liveLocations: LiveLocationState
): LiveDevicePoint[] {
  const points: LiveDevicePoint[] = [];

  for (const device of devices) {
    const liveLocation = liveLocations[device.device_sn];
    const payload = device.status_payload ?? {};
    const accuracyMeters = toNumber(payload.accuracy_m ?? payload.accuracy ?? 5) ?? 5;

    if (liveLocation) {
      points.push({
        deviceSN: device.device_sn,
        name: device.name,
        lat: liveLocation.lat,
        lng: liveLocation.lng,
        battery: device.battery,
        status: device.status,
        gpsState: device.gps_state,
        lastUpdate: liveLocation.time ?? device.last_online,
        accuracyMeters,
      });
      continue;
    }

    const lat = toNumber(device.last_latitude ?? payload.lat ?? payload.latitude);
    const lng = toNumber(device.last_longitude ?? payload.lng ?? payload.lon ?? payload.longitude);
    const point = {
      deviceSN: device.device_sn,
      name: device.name,
      lat: lat ?? NaN,
      lng: lng ?? NaN,
      battery: device.battery,
      status: device.status,
      gpsState: device.gps_state,
      lastUpdate: device.last_location_at ?? device.last_online,
      accuracyMeters,
    };

    if (isValidCoordinate(point)) {
      points.push(point);
    }
  }

  return points;
}
