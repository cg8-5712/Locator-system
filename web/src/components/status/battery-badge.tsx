export function BatteryBadge({ value }: { value: number }) {
  const normalized = Math.max(0, Math.min(100, value));
  const toneClassName =
    normalized <= 15
      ? "bg-[#d94747]/12 text-[#9d2323]"
      : normalized <= 35
        ? "bg-[#d48a1f]/12 text-[#9d6412]"
        : "bg-[#2f9e68]/12 text-[#20724c]";

  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${toneClassName}`}
    >
      电量 {normalized}%
    </span>
  );
}
