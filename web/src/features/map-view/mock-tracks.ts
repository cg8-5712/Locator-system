import type { TrackPoint } from "../../types/device";

const now = Date.now();

export const mockTracksByDeviceSN: Record<string, TrackPoint[]> = {
  "locator-esp32s3-001": [
    { lat: 39.9055, lng: 116.3945, time: new Date(now - 50 * 60_000).toISOString(), still_seconds: 0 },
    { lat: 39.9062, lng: 116.3959, time: new Date(now - 42 * 60_000).toISOString(), still_seconds: 0 },
    { lat: 39.9071, lng: 116.3968, time: new Date(now - 30 * 60_000).toISOString(), still_seconds: 0 },
    { lat: 39.9079, lng: 116.3979, time: new Date(now - 12 * 60_000).toISOString(), still_seconds: 180 },
    { lat: 39.9086, lng: 116.3993, time: new Date(now - 5 * 60_000).toISOString(), still_seconds: 0 },
  ],
  "locator-esp32s3-002": [
    { lat: 39.9132, lng: 116.4106, time: new Date(now - 90 * 60_000).toISOString(), still_seconds: 0 },
    { lat: 39.9141, lng: 116.4115, time: new Date(now - 75 * 60_000).toISOString(), still_seconds: 0 },
    { lat: 39.9146, lng: 116.4122, time: new Date(now - 60 * 60_000).toISOString(), still_seconds: 3600 },
  ],
  "locator-esp32s3-003": [
    { lat: 39.8996, lng: 116.4301, time: new Date(now - 45 * 60_000).toISOString(), still_seconds: 0 },
    { lat: 39.9002, lng: 116.4311, time: new Date(now - 28 * 60_000).toISOString(), still_seconds: 0 },
    { lat: 39.9008, lng: 116.4321, time: new Date(now - 12 * 60_000).toISOString(), still_seconds: 0 },
  ],
};
