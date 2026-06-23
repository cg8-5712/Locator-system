import { useNavigate } from "react-router-dom";
import { buildModePath } from "../../features/map-view/mode";
import type { AppMode } from "../../features/map-view/mode";
import { getActivityLabel, getGPSStateLabel } from "../../lib/status";
import { formatDateTime, formatRelativeTime } from "../../lib/time";
import type { DeviceSummary } from "../../types/device";
import { BatteryBadge } from "../status/battery-badge";
import { StatusBadge } from "../status/status-badge";

const text = {
  placeholder:
    "选中某个人员后，这里会显示实时状态、设备信息、最后可信定位时间和当前生效配置摘要。",
  current: "当前选中",
  lastOnline: "最近在线",
  trustedFix: "可信定位",
  statusUpdatedAt: "状态更新时间",
  configUpdatedAt: "配置更新时间",
  sceneStatus: "现场状态",
  currentAddress: "当前位置",
  accuracy: "定位精度",
  network: "网络注册",
  firmware: "模块固件",
  build: "构建版本",
  deviceInfo: "设备信息",
  mqttOnline: "MQTT 在线",
  startupDone: "启动完成",
  healthCheck: "健康检查",
  configSummary: "生效配置摘要",
  pubMs: "上报周期",
  moveThreshold: "移动阈值",
  stillConfirmMs: "静止确认",
  nofixKeepaliveMs: "无定位保活",
  remoteConfig: "远程配置",
  history: "历史轨迹",
  geofence: "查看围栏",
  share: "分享实时位置",
};

function renderValue(value: unknown): string {
  if (typeof value === "number" || typeof value === "string") {
    return String(value);
  }

  return "--";
}

function readString(value: unknown) {
  return typeof value === "string" && value.trim() ? value.trim() : null;
}

function readNumber(value: unknown) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

export function DeviceDetailPanel({
  device,
  mode,
  onOpenShare,
}: {
  device: DeviceSummary | null;
  mode: AppMode;
  onOpenShare: () => void;
}) {
  const navigate = useNavigate();

  if (!device) {
    return (
      <section className="glass-panel flex h-full min-h-0 items-center justify-center rounded-[28px] p-6">
        <div className="max-w-sm text-center text-sm leading-7 text-[#546570]">
          {text.placeholder}
        </div>
      </section>
    );
  }

  const statusPayload = device.status_payload ?? {};
  const configPayload = device.config_payload ?? {};
  const role = readString(statusPayload.role);
  const address = readString(statusPayload.address);
  const activity = getActivityLabel(statusPayload.activity);
  const speedKmh = readNumber(statusPayload.speed_kmh);
  const accuracyMeters = readNumber(statusPayload.accuracy_m ?? statusPayload.accuracy);

  return (
    <section className="glass-panel h-full min-h-0 rounded-[28px] p-5">
      <div className="flex h-full min-h-0 flex-col">
        <div className="shrink-0">
          <div className="flex items-start justify-between gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.22em] text-[#1f88c9]">
                {text.current}
              </p>
              <h3 className="mt-2 text-2xl font-semibold text-[#10212b]">
                {device.name || device.device_sn}
              </h3>
              <p className="mt-1 text-sm text-[#546570]">
                {role ? `${role} · ${device.device_sn}` : device.device_sn}
              </p>
            </div>
            <StatusBadge device={device} />
          </div>

          <div className="mt-5 flex flex-wrap gap-2">
            <BatteryBadge value={device.battery} />
            <span className="inline-flex rounded-full bg-[#10212b]/6 px-2.5 py-1 text-xs font-semibold text-[#3e505a]">
              GPS {getGPSStateLabel(device.gps_state)}
            </span>
            {activity ? (
              <span className="inline-flex rounded-full bg-[#1f88c9]/8 px-2.5 py-1 text-xs font-semibold text-[#1b628a]">
                {activity}
                {speedKmh ? ` · ${speedKmh.toFixed(1)} km/h` : ""}
              </span>
            ) : null}
          </div>
        </div>

        <div className="mt-6 min-h-0 flex-1 space-y-6 overflow-y-auto pr-1">
          <div className="grid gap-3 sm:grid-cols-2">
            <MetricCard label={text.lastOnline} value={formatRelativeTime(device.last_online)} />
            <MetricCard label={text.trustedFix} value={formatRelativeTime(device.last_fix_at)} />
            <MetricCard
              label={text.statusUpdatedAt}
              value={formatDateTime(device.status_updated_at)}
            />
            <MetricCard
              label={text.configUpdatedAt}
              value={formatDateTime(device.config_updated_at)}
            />
          </div>

          <section>
            <h4 className="text-sm font-semibold text-[#10212b]">{text.sceneStatus}</h4>
            <dl className="mt-3 grid gap-3 text-sm text-[#546570]">
              <DetailRow label={text.currentAddress} value={address ?? "--"} />
              <DetailRow
                label={text.accuracy}
                value={accuracyMeters !== null ? `${accuracyMeters} m` : "--"}
              />
              <DetailRow label={text.network} value={renderValue(statusPayload.net)} />
              <DetailRow label={text.firmware} value={renderValue(statusPayload.fw)} />
              <DetailRow label={text.build} value={renderValue(statusPayload.build)} />
            </dl>
          </section>

          <section>
            <h4 className="text-sm font-semibold text-[#10212b]">{text.deviceInfo}</h4>
            <dl className="mt-3 grid gap-3 text-sm text-[#546570]">
              <DetailRow label="IMEI" value={device.imei || "--"} />
              <DetailRow label="ICCID" value={device.iccid || "--"} />
              <DetailRow label={text.mqttOnline} value={renderValue(statusPayload.mqtt)} />
              <DetailRow label={text.startupDone} value={renderValue(statusPayload.startup)} />
              <DetailRow label={text.healthCheck} value={renderValue(statusPayload.health)} />
            </dl>
          </section>

          <section>
            <h4 className="text-sm font-semibold text-[#10212b]">{text.configSummary}</h4>
            <dl className="mt-3 grid gap-3 text-sm text-[#546570]">
              <DetailRow label={text.pubMs} value={`${renderValue(configPayload.pub_ms)} ms`} />
              <DetailRow
                label={text.moveThreshold}
                value={`${renderValue(configPayload.move_m)} m`}
              />
              <DetailRow
                label={text.stillConfirmMs}
                value={`${renderValue(configPayload.still_confirm_ms)} ms`}
              />
              <DetailRow
                label={text.nofixKeepaliveMs}
                value={`${renderValue(configPayload.nofix_keepalive_ms)} ms`}
              />
              <DetailRow
                label={text.remoteConfig}
                value={renderValue(configPayload.remote_cfg)}
              />
            </dl>
          </section>
        </div>

        <div className="mt-6 shrink-0 flex flex-wrap gap-3">
          <ActionButton
            onClick={() =>
              navigate(buildModePath(mode, `/devices/${device.device_sn}/history`))
            }
          >
            {text.history}
          </ActionButton>
          <ActionButton disabled>{text.geofence}</ActionButton>
          <ActionButton tone="dark" onClick={onOpenShare}>
            {text.share}
          </ActionButton>
        </div>
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
  onClick,
  disabled = false,
}: {
  children: string;
  tone?: "light" | "dark";
  onClick?: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={
        tone === "dark"
          ? "rounded-2xl bg-[#10212b] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#163445] disabled:cursor-not-allowed disabled:opacity-50"
          : "rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b] transition hover:border-[#1f88c9]/20 hover:bg-white disabled:cursor-not-allowed disabled:opacity-50"
      }
    >
      {children}
    </button>
  );
}
