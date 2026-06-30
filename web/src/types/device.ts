export type DeviceStatus = number;

export type GPSState =
  | "not_started"
  | "offline"
  | "searching"
  | "located"
  | "unable"
  | "";

export interface DeviceSummary {
  device_sn: string;
  imei: string;
  iccid: string;
  name: string;
  topic_prefix: string;
  gps_state: GPSState;
  status: DeviceStatus;
  battery: number;
  status_payload?: Record<string, unknown>;
  config_payload?: Record<string, unknown>;
  status_updated_at?: string;
  config_updated_at?: string;
  last_fix_at?: string;
  last_online?: string;
  created_at: string;
}

export interface DeviceListResult {
  devices: DeviceSummary[];
  pagination: import("./api").Pagination;
}

export interface TrackPoint {
  lat: number;
  lng: number;
  time: string;
  still_seconds: number;
}

export interface DeviceTrackResult {
  device_sn: string;
  tracks: TrackPoint[];
  pagination: import("./api").Pagination;
}

export interface DeviceCommandResult {
  topic: string;
  published: boolean;
  payload: Record<string, unknown>;
}
