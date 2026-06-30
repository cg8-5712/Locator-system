export interface FencePoint {
  lat: number;
  lng: number;
}

export interface FenceSummary {
  id: number;
  device_sn: string;
  name: string;
  polygon: FencePoint[];
  last_inside?: boolean;
  last_checked_at?: string;
  created_at: string;
}

export interface FenceListResult {
  fences: FenceSummary[];
}
