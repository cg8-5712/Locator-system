import { create } from "zustand";
import type { RealtimeEnvelope } from "../types/realtime";

interface MapState {
  selectedDeviceSN: string | null;
  searchText: string;
  followSelected: boolean;
  wsConnected: boolean;
  lastRealtimeMessage: RealtimeEnvelope | null;
  liveLocations: Record<
    string,
    {
      lat: number;
      lng: number;
      time?: string;
      stillSeconds?: number;
    }
  >;
  setSelectedDeviceSN: (deviceSN: string | null) => void;
  setSearchText: (value: string) => void;
  setFollowSelected: (value: boolean) => void;
  setWsConnected: (value: boolean) => void;
  setLastRealtimeMessage: (message: RealtimeEnvelope | null) => void;
  upsertLiveLocation: (
    deviceSN: string,
    location: {
      lat: number;
      lng: number;
      time?: string;
      stillSeconds?: number;
    }
  ) => void;
}

export const useMapStore = create<MapState>((set) => ({
  selectedDeviceSN: null,
  searchText: "",
  followSelected: true,
  wsConnected: false,
  lastRealtimeMessage: null,
  liveLocations: {},
  setSelectedDeviceSN: (selectedDeviceSN) => set({ selectedDeviceSN }),
  setSearchText: (searchText) => set({ searchText }),
  setFollowSelected: (followSelected) => set({ followSelected }),
  setWsConnected: (wsConnected) => set({ wsConnected }),
  setLastRealtimeMessage: (lastRealtimeMessage) => set({ lastRealtimeMessage }),
  upsertLiveLocation: (deviceSN, location) =>
    set((state) => ({
      liveLocations: {
        ...state.liveLocations,
        [deviceSN]: location,
      },
    })),
}));
