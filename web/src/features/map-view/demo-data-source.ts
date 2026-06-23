import { useEffect, useMemo, useState } from "react";
import type { DeviceSummary } from "../../types/device";
import type { RealtimeEnvelope } from "../../types/realtime";
import type { MapDataSource } from "./data-source";
import type { LiveLocationState } from "./map-types";
import { mockDevices } from "./mock-devices";

function cloneDevices() {
  return mockDevices.map((device) => ({
    ...device,
    status_payload: device.status_payload ? { ...device.status_payload } : undefined,
    config_payload: device.config_payload ? { ...device.config_payload } : undefined,
  }));
}

function useMockRealtime() {
  const [liveLocations, setLiveLocations] = useState<LiveLocationState>(() => {
    const entries = mockDevices.map((device) => {
      const payload = device.status_payload ?? {};
      return [
        device.device_sn,
        {
          lat: Number(payload.lat ?? 39.9),
          lng: Number(payload.lng ?? 116.3),
          time: device.last_online,
          stillSeconds: 0,
        },
      ] as const;
    });

    return Object.fromEntries(entries);
  });
  const [message, setMessage] = useState<RealtimeEnvelope | null>(null);

  useEffect(() => {
    const interval = window.setInterval(() => {
      const first = mockDevices[0];
      if (!first) {
        return;
      }

      const current = liveLocations[first.device_sn];
      const baseLat = current?.lat ?? 39.9074;
      const baseLng = current?.lng ?? 116.3975;
      const nextLat = Number((baseLat + (Math.random() - 0.5) * 0.0012).toFixed(6));
      const nextLng = Number((baseLng + (Math.random() - 0.5) * 0.0012).toFixed(6));
      const now = new Date().toISOString();

      setLiveLocations((prev) => ({
        ...prev,
        [first.device_sn]: {
          lat: nextLat,
          lng: nextLng,
          time: now,
          stillSeconds: 0,
        },
      }));

      setMessage({
        type: "location",
        data: {
          device_sn: first.device_sn,
          topic_prefix: first.topic_prefix,
          lat: nextLat,
          lng: nextLng,
          time: now,
          still_seconds: 0,
          gps_state: first.gps_state,
          status: first.status,
        },
      });
    }, 4000);

    return () => {
      window.clearInterval(interval);
    };
  }, [liveLocations]);

  return {
    connected: true,
    lastMessage: message,
    liveLocations,
  };
}

function useDemoDevices() {
  return {
    devices: useMemo(() => cloneDevices(), []),
    isLoading: false,
    isError: false,
    errorMessage: null,
  };
}

function useDemoDeviceDetail(deviceSN: string | null) {
  const device = useMemo(
    () => cloneDevices().find((item) => item.device_sn === deviceSN) ?? null,
    [deviceSN]
  );

  return {
    device,
    isLoading: false,
  };
}

export const demoDataSource: MapDataSource = {
  mode: "demo",
  useDevices: useDemoDevices,
  useDeviceDetail: useDemoDeviceDetail,
  useRealtimeFeed: useMockRealtime,
};
