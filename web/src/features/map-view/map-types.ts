export interface LiveDevicePoint {
  deviceSN: string;
  name: string;
  lat: number;
  lng: number;
  battery: number;
  status: number;
  gpsState: string;
  lastUpdate?: string;
}
