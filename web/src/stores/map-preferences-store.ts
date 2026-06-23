import { create } from "zustand";
import { persist } from "zustand/middleware";
import { getDefaultMapTilePresetId } from "../lib/map-tiles";

interface MapPreferenceState {
  tilePresetId: string;
  setTilePresetId: (tilePresetId: string) => void;
}

export const useMapPreferencesStore = create<MapPreferenceState>()(
  persist(
    (set) => ({
      tilePresetId: getDefaultMapTilePresetId(),
      setTilePresetId: (tilePresetId) => set({ tilePresetId }),
    }),
    {
      name: "locator-map-preferences",
    }
  )
);
