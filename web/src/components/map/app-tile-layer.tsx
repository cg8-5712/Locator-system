import { TileLayer } from "react-leaflet";
import { getMapTilePresetById } from "../../lib/map-tiles";
import { useMapPreferencesStore } from "../../stores/map-preferences-store";

export function AppTileLayer() {
  const tilePresetId = useMapPreferencesStore((state) => state.tilePresetId);
  const preset = getMapTilePresetById(tilePresetId);

  return (
    <TileLayer
      attribution={preset.attribution}
      url={preset.url}
      subdomains={preset.subdomains}
      maxZoom={preset.maxZoom}
    />
  );
}
