import { useMemo } from "react";
import { useDeviceDetail, useDeviceList } from "../../hooks/use-devices";
import { useRealtime } from "../../hooks/use-realtime";
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
  const liveLocations = useMapStore((state) => state.liveLocations);

  return useMemo(
    () => ({
      connected: wsConnected,
      lastMessage: null,
      liveLocations,
    }),
    [liveLocations, wsConnected]
  );
}

export const liveDataSource: MapDataSource = {
  mode: "live",
  useDevices: useLiveDevices,
  useDeviceDetail: useLiveDeviceDetail,
  useRealtimeFeed: useLiveRealtime,
};
