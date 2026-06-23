import { useMemo } from "react";
import { useAlarmList } from "../../hooks/use-alarms";
import { useDeviceDetail, useDeviceList } from "../../hooks/use-devices";
import { useRealtime } from "../../hooks/use-realtime";
import { useTrack } from "../../hooks/use-track";
import { useMapStore } from "../../stores/map-store";
import type { MapDataSource } from "./data-source";

function useLiveDevices() {
  const query = useDeviceList();

  return {
    devices: query.data?.devices ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    errorMessage: query.error instanceof Error ? query.error.message : null,
  };
}

function useLiveDeviceDetail(deviceSN: string | null) {
  const query = useDeviceDetail(deviceSN);

  return {
    device: query.data ?? null,
    isLoading: query.isLoading,
  };
}

function useLiveRealtime() {
  useRealtime();
  const wsConnected = useMapStore((state) => state.wsConnected);
  const lastRealtimeMessage = useMapStore((state) => state.lastRealtimeMessage);
  const liveLocations = useMapStore((state) => state.liveLocations);

  return useMemo(
    () => ({
      connected: wsConnected,
      lastMessage: lastRealtimeMessage,
      liveLocations,
    }),
    [lastRealtimeMessage, liveLocations, wsConnected]
  );
}

function useLiveAlarms(options?: { deviceSN?: string | null; limit?: number }) {
  const query = useAlarmList({
    deviceSN: options?.deviceSN ?? undefined,
    pageSize: options?.limit ?? 50,
  });

  return {
    alarms: query.data?.alarms ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    errorMessage: query.error instanceof Error ? query.error.message : null,
  };
}

function useLiveTrack(deviceSN: string | null, options: { rangeHours: number }) {
  const query = useTrack(deviceSN, options.rangeHours);

  return {
    tracks: query.data?.tracks ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    errorMessage: query.error instanceof Error ? query.error.message : null,
  };
}

export const liveDataSource: MapDataSource = {
  mode: "live",
  useDevices: useLiveDevices,
  useDeviceDetail: useLiveDeviceDetail,
  useRealtimeFeed: useLiveRealtime,
  useAlarms: useLiveAlarms,
  useTrack: useLiveTrack,
};
