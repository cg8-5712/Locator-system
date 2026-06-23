import { BatteryBadge } from "../status/battery-badge";
import { StatusBadge } from "../status/status-badge";
import { formatDurationSeconds, formatRelativeTime } from "../../lib/time";
import { getActivityLabel } from "../../lib/status";
import type { DeviceSummary } from "../../types/device";

function readString(value: unknown) {
  return typeof value === "string" && value.trim() ? value.trim() : null;
}

function readNumber(value: unknown) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

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
          Team View
        </p>
        <h2 className="mt-2 text-2xl font-semibold text-[#10212b]">人员列表</h2>
      </div>

      <label className="mb-4 block">
        <span className="sr-only">搜索人员</span>
        <input
          value={searchText}
          onChange={(event) => onSearchTextChange(event.target.value)}
          placeholder="搜索姓名、设备号、IMEI 或 ICCID"
          className="w-full rounded-2xl border border-black/8 bg-white/80 px-4 py-3 text-sm outline-none transition focus:border-[#1f88c9] focus:ring-4 focus:ring-[#1f88c9]/10"
        />
      </label>

      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
        {devices.length === 0 ? (
          <div className="rounded-[24px] border border-dashed border-black/10 bg-white/56 px-4 py-6 text-sm leading-7 text-[#546570]">
            当前筛选条件下没有匹配的人员。
          </div>
        ) : null}

        {devices.map((device) => {
          const payload = device.status_payload ?? {};
          const role = readString(payload.role);
          const activity = getActivityLabel(payload.activity);
          const stillSeconds = readNumber(payload.still_seconds);
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
                    {role ? `${role} · ${device.device_sn}` : device.device_sn}
                  </div>
                </div>
                <StatusBadge device={device} />
              </div>

              <div className="mt-4 flex flex-wrap gap-2">
                <BatteryBadge value={device.battery} />
                {activity ? (
                  <span className="inline-flex rounded-full bg-[#10212b]/6 px-2.5 py-1 text-xs font-semibold text-[#3e505a]">
                    {activity}
                    {activity === "静止" && stillSeconds
                      ? ` ${formatDurationSeconds(stillSeconds)}`
                      : ""}
                  </span>
                ) : null}
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
