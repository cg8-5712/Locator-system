/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_MAP_TILE_DEFAULT_PRESET?: string;
  readonly VITE_MAP_TILE_CUSTOM_URL?: string;
  readonly VITE_MAP_TILE_CUSTOM_ATTRIBUTION?: string;
  readonly VITE_MAP_TILE_CUSTOM_LABEL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
