import { useEffect, useMemo, useState } from "react";
import type { MapDataSource } from "./data-source";
import type { LiveLocationState } from "./map-types";
import type { TrackPoint } from "../../types/device";
import type { RealtimeEnvelope } from "../../types/realtime";
import { mockAlarms } from "./mock-alarms";
import { mockDevices } from "./mock-devices";
import { mockTracksByDeviceSN } from "./mock-tracks";

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
          stillSeconds: Number(payload.still_seconds ?? 0),
        },
      ] as const;
    });

    return Object.fromEntries(entries);
  });
  const [message, setMessage] = useState<RealtimeEnvelope | null>(null);

  useEffect(() => {
    let emission = 0;

    const interval = window.setInterval(() => {
      const walker = mockDevices[0];
      const sosDevice = mockDevices[2];
      if (!walker || !sosDevice) {
        return;
      }

      const now = new Date().toISOString();

      if (emission % 3 === 2) {
        setMessage({
          type: "alarm",
          data: {
            device_sn: sosDevice.device_sn,
            type: "sos",
            content: "演示告警：人员触发 SOS 求救，请立即处理。",
            created_at: now,
          },
        });
        emission += 1;
        return;
      }

      setLiveLocations((prev) => {
        const current = prev[walker.device_sn];
        const baseLat = current?.lat ?? 39.9074;
        const baseLng = current?.lng ?? 116.3975;
        const nextLat = Number((baseLat + (Math.random() - 0.45) * 0.0012).toFixed(6));
        const nextLng = Number((baseLng + (Math.random() - 0.45) * 0.0012).toFixed(6));

        setMessage({
          type: "location",
          data: {
            device_sn: walker.device_sn,
            topic_prefix: walker.topic_prefix,
            lat: nextLat,
            lng: nextLng,
            time: now,
            still_seconds: 0,
            gps_state: walker.gps_state,
            status: walker.status,
          },
        });

        return {
          ...prev,
          [walker.device_sn]: {
            lat: nextLat,
            lng: nextLng,
            time: now,
            stillSeconds: 0,
          },
        };
      });

      emission += 1;
    }, 4000);

    return () => {
      window.clearInterval(interval);
    };
  }, []);

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

function useDemoAlarms(options?: { deviceSN?: string | null; limit?: number }) {
  const alarms = useMemo(() => {
    const filtered = options?.deviceSN
      ? mockAlarms.filter((alarm) => alarm.device_sn === options.deviceSN)
      : mockAlarms;

    return filtered.slice(0, options?.limit ?? filtered.length);
  }, [options?.deviceSN, options?.limit]);

  return {
    alarms,
    isLoading: false,
    isError: false,
    errorMessage: null,
  };
}

function useDemoTrack(deviceSN: string | null, options: { rangeHours: number }) {
  const tracks = useMemo<TrackPoint[]>(() => {
    if (!deviceSN) {
      return [];
    }

    const allTracks = mockTracksByDeviceSN[deviceSN] ?? [];
    const cutoff = Date.now() - options.rangeHours * 60 * 60 * 1000;
    return allTracks.filter((track) => new Date(track.time).getTime() >= cutoff);
  }, [deviceSN, options.rangeHours]);

  return {
    tracks,
    isLoading: false,
    isError: false,
    errorMessage: null,
  };
}

export const demoDataSource: MapDataSource = {
  mode: "demo",
  useDevices: useDemoDevices,
  useDeviceDetail: useDemoDeviceDetail,
  useRealtimeFeed: useMockRealtime,
  useAlarms: useDemoAlarms,
  useTrack: useDemoTrack,
};
