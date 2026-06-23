import type { ReactNode } from "react";
import { createContext, useContext } from "react";
import type { MapDataSource } from "./data-source";

const MapDataContext = createContext<MapDataSource | null>(null);

export function MapDataProvider({
  value,
  children,
}: {
  value: MapDataSource;
  children: ReactNode;
}) {
  return <MapDataContext.Provider value={value}>{children}</MapDataContext.Provider>;
}

export function useMapDataSource() {
  const value = useContext(MapDataContext);
  if (!value) {
    throw new Error("MapDataProvider is missing");
  }

  return value;
}
