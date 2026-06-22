import { getDeviceStateView } from "../../lib/status";
import type { DeviceSummary } from "../../types/device";

const toneClassName = {
  safe: "bg-[#2f9e68]/12 text-[#20724c]",
  brand: "bg-[#1f88c9]/12 text-[#176794]",
  warn: "bg-[#d48a1f]/12 text-[#9d6412]",
  danger: "bg-[#d94747]/12 text-[#9d2323]",
  offline: "bg-[#7c8b94]/12 text-[#51616a]",
} as const;

export function StatusBadge({
  device,
}: {
  device: Pick<DeviceSummary, "status" | "gps_state" | "battery">;
}) {
  const view = getDeviceStateView(device);

  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${toneClassName[view.tone]}`}
    >
      {view.label}
    </span>
  );
}
