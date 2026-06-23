export interface LiveDevicePoint {
  deviceSN: string;
  name: string;
  lat: number;
  lng: number;
  battery: number;
  status: number;
  gpsState: string;
  lastUpdate?: string;
  accuracyMeters?: number;
}

export interface GeofenceOverlay {
  deviceSN: string;
  name: string | null;
  centerLat: number;
  centerLng: number;
  radiusMeters: number;
}

export type LiveLocationState = Record<
  string,
  {
    lat: number;
    lng: number;
    time?: string;
    stillSeconds?: number;
  }
>;
