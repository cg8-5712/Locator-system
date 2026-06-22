export interface Coordinate {
  lat: number;
  lng: number;
}

export function isValidCoordinate(value?: Coordinate | null): value is Coordinate {
  return Boolean(
    value &&
      Number.isFinite(value.lat) &&
      Number.isFinite(value.lng) &&
      Math.abs(value.lat) <= 90 &&
      Math.abs(value.lng) <= 180
  );
}
