import type { DeviceSummary } from "../../types/device";
import type { AlarmSummary } from "../../types/alarm";
import type { RealtimeEnvelope } from "../../types/realtime";
import type { AppMode } from "./mode";
import type { LiveLocationState } from "./map-types";
import type { TrackPoint } from "../../types/device";

export interface MapDataSource {
  mode: AppMode;
  useDevices: () => {
    devices: DeviceSummary[];
    isLoading: boolean;
    isError: boolean;
    errorMessage: string | null;
  };
  useDeviceDetail: (deviceSN: string | null) => {
    device: DeviceSummary | null;
    isLoading: boolean;
  };
  useRealtimeFeed: () => {
    connected: boolean;
    lastMessage: RealtimeEnvelope | null;
    liveLocations: LiveLocationState;
  };
  useAlarms: (options?: { deviceSN?: string | null; limit?: number }) => {
    alarms: AlarmSummary[];
    isLoading: boolean;
    isError: boolean;
    errorMessage: string | null;
  };
  useTrack: (
    deviceSN: string | null,
    options: { rangeHours: number }
  ) => {
    tracks: TrackPoint[];
    isLoading: boolean;
    isError: boolean;
    errorMessage: string | null;
  };
}
