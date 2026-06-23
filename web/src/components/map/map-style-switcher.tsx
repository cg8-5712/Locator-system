import { getAvailableMapTilePresets } from "../../lib/map-tiles";
import { useMapPreferencesStore } from "../../stores/map-preferences-store";

export function MapStyleSwitcher({
  compact = false,
}: {
  compact?: boolean;
}) {
  const tilePresetId = useMapPreferencesStore((state) => state.tilePresetId);
  const setTilePresetId = useMapPreferencesStore((state) => state.setTilePresetId);
  const presets = getAvailableMapTilePresets();

  return (
    <div
      className={`rounded-[24px] border border-black/8 bg-white/78 backdrop-blur-sm ${
        compact ? "px-3 py-3" : "px-4 py-4"
      }`}
    >
      <div className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[#7a8a94]">
        底图样式
      </div>
      <div className={`mt-3 grid gap-2 ${compact ? "sm:grid-cols-2" : "sm:grid-cols-4"}`}>
        {presets.map((preset) => {
          const active = preset.id === tilePresetId;

          return (
            <button
              key={preset.id}
              type="button"
              onClick={() => setTilePresetId(preset.id)}
              className={`rounded-2xl border px-3 py-2.5 text-left transition ${
                active
                  ? "border-[#1f88c9]/28 bg-[#1f88c9]/10 shadow-[0_12px_24px_rgba(31,136,201,0.12)]"
                  : "border-black/8 bg-white/84 hover:border-[#1f88c9]/20 hover:bg-white"
              }`}
            >
              <div className="text-sm font-semibold text-[#10212b]">{preset.label}</div>
              <div className="mt-1 text-xs leading-5 text-[#546570]">{preset.description}</div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
