import { BatteryBadge } from "../status/battery-badge";
import { StatusBadge } from "../status/status-badge";
import { formatDateTime, formatRelativeTime } from "../../lib/time";
import type { DeviceSummary } from "../../types/device";

function renderConfigValue(value: unknown): string {
  if (typeof value === "number" || typeof value === "string") {
    return String(value);
  }

  return "--";
}

export function DeviceDetailPanel({
  device,
}: {
  device: DeviceSummary | null;
}) {
  if (!device) {
    return (
      <section className="glass-panel flex h-full items-center justify-center rounded-[28px] p-6">
        <div className="max-w-sm text-center text-sm leading-7 text-[#546570]">
          选择一名人员后，可查看实时状态、设备信息、最近可信定位时间和当前生效配置摘要。
        </div>
      </section>
    );
  }

  const statusPayload = device.status_payload ?? {};
  const configPayload = device.config_payload ?? {};

  return (
    <section className="glass-panel h-full rounded-[28px] p-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.22em] text-[#1f88c9]">
            当前选中
          </p>
          <h3 className="mt-2 text-2xl font-semibold text-[#10212b]">
            {device.name || device.device_sn}
          </h3>
          <p className="mt-1 text-sm text-[#546570]">{device.device_sn}</p>
        </div>
        <StatusBadge device={device} />
      </div>

      <div className="mt-5 flex flex-wrap gap-2">
        <BatteryBadge value={device.battery} />
        <span className="inline-flex rounded-full bg-[#10212b]/6 px-2.5 py-1 text-xs font-semibold text-[#3e505a]">
          GPS {device.gps_state || "未知"}
        </span>
      </div>

      <div className="mt-6 grid gap-3 sm:grid-cols-2">
        <MetricCard label="最近在线" value={formatRelativeTime(device.last_online)} />
        <MetricCard label="可信定位" value={formatRelativeTime(device.last_fix_at)} />
        <MetricCard label="状态更新时间" value={formatDateTime(device.status_updated_at)} />
        <MetricCard label="配置更新时间" value={formatDateTime(device.config_updated_at)} />
      </div>

      <div className="mt-6 space-y-6">
        <section>
          <h4 className="text-sm font-semibold text-[#10212b]">设备信息</h4>
          <dl className="mt-3 grid gap-3 text-sm text-[#546570]">
            <DetailRow label="IMEI" value={device.imei || "--"} />
            <DetailRow label="ICCID" value={device.iccid || "--"} />
            <DetailRow
              label="模块固件"
              value={renderConfigValue(statusPayload.fw)}
            />
            <DetailRow
              label="网络注册"
              value={renderConfigValue(statusPayload.net)}
            />
            <DetailRow
              label="设备构建"
              value={renderConfigValue(statusPayload.build)}
            />
          </dl>
        </section>

        <section>
          <h4 className="text-sm font-semibold text-[#10212b]">生效配置摘要</h4>
          <dl className="mt-3 grid gap-3 text-sm text-[#546570]">
            <DetailRow
              label="上报周期"
              value={`${renderConfigValue(configPayload.pub_ms)} ms`}
            />
            <DetailRow
              label="移动阈值"
              value={`${renderConfigValue(configPayload.move_m)} m`}
            />
            <DetailRow
              label="静止确认"
              value={`${renderConfigValue(configPayload.still_confirm_ms)} ms`}
            />
            <DetailRow
              label="无定位保活"
              value={`${renderConfigValue(configPayload.nofix_keepalive_ms)} ms`}
            />
            <DetailRow
              label="远程配置"
              value={renderConfigValue(configPayload.remote_cfg)}
            />
          </dl>
        </section>
      </div>

      <div className="mt-6 flex flex-wrap gap-3">
        <ActionButton>历史轨迹</ActionButton>
        <ActionButton>查看围栏</ActionButton>
        <ActionButton tone="dark">分享实时位置</ActionButton>
      </div>
    </section>
  );
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[22px] border border-black/6 bg-white/64 p-4">
      <div className="text-xs uppercase tracking-[0.18em] text-[#7a8a94]">{label}</div>
      <div className="mt-2 text-sm font-semibold text-[#10212b]">{value}</div>
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-2xl bg-white/50 px-3 py-2.5">
      <dt className="text-[#6a7a84]">{label}</dt>
      <dd className="text-right font-medium text-[#10212b]">{value}</dd>
    </div>
  );
}

function ActionButton({
  children,
  tone = "light",
}: {
  children: string;
  tone?: "light" | "dark";
}) {
  return (
    <button
      type="button"
      className={
        tone === "dark"
          ? "rounded-2xl bg-[#10212b] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#163445]"
          : "rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b] transition hover:border-[#1f88c9]/20 hover:bg-white"
      }
    >
      {children}
    </button>
  );
}
