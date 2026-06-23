export interface MapTilePreset {
  id: string;
  label: string;
  description: string;
  url: string;
  attribution: string;
  subdomains?: string | string[];
  maxZoom?: number;
}

const customTileUrl = import.meta.env.VITE_MAP_TILE_CUSTOM_URL?.trim();
const customTileAttribution =
  import.meta.env.VITE_MAP_TILE_CUSTOM_ATTRIBUTION?.trim() ||
  '&copy; Custom Tile Provider';
const customTileLabel =
  import.meta.env.VITE_MAP_TILE_CUSTOM_LABEL?.trim() || "自定义 XYZ";
const configuredDefaultPresetId =
  import.meta.env.VITE_MAP_TILE_DEFAULT_PRESET?.trim() || "osm";

const basePresets: MapTilePreset[] = [
  {
    id: "osm",
    label: "OSM 标准",
    description: "标准 OpenStreetMap 底图，兼容性最好。",
    url: "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png",
    attribution:
      '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
    maxZoom: 19,
  },
  {
    id: "light",
    label: "浅色政区",
    description: "更适合室内运营台和白天值守场景。",
    url: "https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png",
    attribution:
      '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
    subdomains: "abcd",
    maxZoom: 20,
  },
  {
    id: "dark",
    label: "暗色夜班",
    description: "适合夜间值守和大屏监控。",
    url: "https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png",
    attribution:
      '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
    subdomains: "abcd",
    maxZoom: 20,
  },
  {
    id: "terrain",
    label: "地形辅助",
    description: "适合查看山区、园区和边界环境。",
    url: "https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png",
    attribution:
      'Map data: &copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors, <a href="https://viewfinderpanoramas.org">SRTM</a> | Map style: &copy; <a href="https://opentopomap.org">OpenTopoMap</a>',
    maxZoom: 17,
  },
];

export function getAvailableMapTilePresets(): MapTilePreset[] {
  if (!customTileUrl) {
    return basePresets;
  }

  return [
    ...basePresets,
    {
      id: "custom",
      label: customTileLabel,
      description: "用于接入自定义 XYZ 底图或国内代理源。",
      url: customTileUrl,
      attribution: customTileAttribution,
      maxZoom: 20,
    },
  ];
}

export function getDefaultMapTilePresetId() {
  const presets = getAvailableMapTilePresets();
  return presets.some((preset) => preset.id === configuredDefaultPresetId)
    ? configuredDefaultPresetId
    : presets[0]?.id ?? "osm";
}

export function getMapTilePresetById(id?: string | null) {
  const presets = getAvailableMapTilePresets();
  return (
    presets.find((preset) => preset.id === id) ??
    presets.find((preset) => preset.id === getDefaultMapTilePresetId()) ??
    presets[0]
  );
}
