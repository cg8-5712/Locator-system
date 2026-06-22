import { BatteryBadge } from "../status/battery-badge";
import { StatusBadge } from "../status/status-badge";
import { formatRelativeTime } from "../../lib/time";
import type { DeviceSummary } from "../../types/device";

export function DeviceSidebar({
  devices,
  selectedDeviceSN,
  searchText,
  onSearchTextChange,
  onSelect,
}: {
  devices: DeviceSummary[];
  selectedDeviceSN: string | null;
  searchText: string;
  onSearchTextChange: (value: string) => void;
  onSelect: (deviceSN: string) => void;
}) {
  return (
    <aside className="glass-panel flex h-full min-h-0 flex-col rounded-[28px] p-4">
      <div className="mb-4">
        <p className="text-xs font-semibold uppercase tracking-[0.22em] text-[#1f88c9]">
          队伍视图
        </p>
        <h2 className="mt-2 text-2xl font-semibold text-[#10212b]">人员列表</h2>
      </div>

      <label className="mb-4 block">
        <span className="sr-only">搜索人员</span>
        <input
          value={searchText}
          onChange={(event) => onSearchTextChange(event.target.value)}
          placeholder="搜索人员姓名或设备号"
          className="w-full rounded-2xl border border-black/8 bg-white/80 px-4 py-3 text-sm outline-none transition focus:border-[#1f88c9] focus:ring-4 focus:ring-[#1f88c9]/10"
        />
      </label>

      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
        {devices.map((device) => {
          const active = selectedDeviceSN === device.device_sn;
          return (
            <button
              key={device.device_sn}
              type="button"
              onClick={() => onSelect(device.device_sn)}
              className={`w-full rounded-[24px] border px-4 py-4 text-left transition ${
                active
                  ? "border-[#1f88c9]/30 bg-[#1f88c9]/8 shadow-[0_12px_24px_rgba(31,136,201,0.12)]"
                  : "border-black/6 bg-white/66 hover:border-[#1f88c9]/20 hover:bg-white/84"
              }`}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="truncate text-sm font-semibold text-[#10212b]">
                    {device.name || device.device_sn}
                  </div>
                  <div className="mt-1 truncate text-xs text-[#546570]">
                    {device.device_sn}
                  </div>
                </div>
                <StatusBadge device={device} />
              </div>

              <div className="mt-4 flex flex-wrap gap-2">
                <BatteryBadge value={device.battery} />
              </div>

              <div className="mt-4 text-xs text-[#6a7a84]">
                最近在线 {formatRelativeTime(device.last_online)}
              </div>
            </button>
          );
        })}
      </div>
    </aside>
  );
}
