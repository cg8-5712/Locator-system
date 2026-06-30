import type { Pagination } from "./api";
import type { TrackPoint } from "./device";

export interface ShareSummary {
  id: number;
  device_sn: string;
  device_name: string;
  share_code: string;
  share_mode: "live_only" | "today_track";
  requires_password: boolean;
  note: string;
  expires_at: string;
  max_visits?: number | null;
  visit_count: number;
  remaining_visits?: number | null;
  last_access_at?: string | null;
  revoked_at?: string | null;
  created_at: string;
  status: "active" | "expiring" | "expired" | "quota_used" | "revoked";
}

export interface ShareCreateResult {
  share: ShareSummary;
  password?: string | null;
}

export interface ShareListResult {
  shares: ShareSummary[];
  pagination: Pagination;
}

export interface PublicShareSummary {
  share_code: string;
  device_sn: string;
  device_name: string;
  share_mode: "live_only" | "today_track";
  requires_password: boolean;
  expires_at: string;
  max_visits?: number | null;
  visit_count: number;
  remaining_visits?: number | null;
  last_access_at?: string | null;
  status: "active" | "expiring" | "expired" | "quota_used" | "revoked";
}

export interface ShareVerifyResult {
  access_token: string;
  expires_at: string;
  share: PublicShareSummary;
}

export interface PublicLocationResult {
  device_sn: string;
  device_name: string;
  battery: number;
  gps_state: string;
  status: number;
  last_online?: string | null;
  last_fix_at?: string | null;
  lat?: number | null;
  lng?: number | null;
  time?: string | null;
  still_seconds: number;
  accuracy_m?: number | null;
  address?: string;
  activity?: string;
}

export interface PublicTrackResult {
  device_sn: string;
  tracks: TrackPoint[];
  start_time: string;
  end_time: string;
}
