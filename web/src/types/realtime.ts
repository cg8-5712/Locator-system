export interface LocationEvent {
  device_sn: string;
  topic_prefix: string;
  lat: number;
  lng: number;
  time: string;
  still_seconds: number;
  gps_state: string;
  status: number;
}

export interface DeviceStatusEvent {
  device_sn: string;
  topic_prefix: string;
  status: number;
  gps_state: string;
  battery: number;
  imei?: string;
  iccid?: string;
  status_payload?: Record<string, unknown>;
  config_payload?: Record<string, unknown>;
  status_updated_at?: string;
  config_updated_at?: string;
  last_online?: string;
  last_fix_at?: string;
}

export interface AlarmEvent {
  device_sn: string;
  type: string;
  content: string;
  created_at: string;
}

export type RealtimeEnvelope =
  | { type: "location"; data: LocationEvent }
  | { type: "device_status"; data: DeviceStatusEvent }
  | { type: "alarm"; data: AlarmEvent };
