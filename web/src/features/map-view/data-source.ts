import type { DeviceSummary } from "../../types/device";
import type { RealtimeEnvelope } from "../../types/realtime";
import type { AppMode } from "./mode";
import type { LiveLocationState } from "./map-types";

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
}
