import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { DeviceMap } from "../../components/map/device-map";
import { AppHeader } from "../../components/shell/app-header";
import { buildMapDecorations } from "../../features/map-view/map-decorations";
import { useMapDataSource } from "../../features/map-view/map-data-context";
import { buildModePath } from "../../features/map-view/mode";
import { deriveLivePoints } from "../../features/map-view/points";
import { getAlarmTypeView, getGPSStateLabel } from "../../lib/status";
import { formatDateTime, formatRelativeTime } from "../../lib/time";
import { useMapStore } from "../../stores/map-store";

const alarmTypes = ["all", "sos", "low_battery", "out_of_fence", "offline"] as const;

const text = {
  total: "告警总数",
  online: "在线",
  title: "告警中心",
  desc: "集中查看 SOS、围栏告警、低电量和离线恢复记录，并联动地图定位到目标人员。",
  backMap: "返回地图",
  all: "全部",
  loadError: "告警数据加载失败",
  empty: "当前没有匹配的告警记录。",
  focus: "当前焦点",
  openHistory: "查看历史轨迹",
  type: "告警类型",
  occurredAt: "发生时间",
  lastOnline: "最近在线",
  deviceStatus: "设备状态",
  placeholder: "选择一条告警后查看地图联动和详情摘要。",
};

export function AlarmsPage() {
  const navigate = useNavigate();
  const dataSource = useMapDataSource();
  const alarmsResult = dataSource.useAlarms({ limit: 50 });
  const devicesResult = dataSource.useDevices();
  const realtimeResult = dataSource.useRealtimeFeed();
  const {
    selectedDeviceSN,
    focusSequence,
    setSelectedDeviceSN,
    setFollowSelected,
    triggerFocus,
  } = useMapStore();
  const [typeFilter, setTypeFilter] = useState<(typeof alarmTypes)[number]>("all");

  const devices = devicesResult.devices;
  const livePoints = useMemo(
    () => deriveLivePoints(devices, realtimeResult.liveLocations),
    [devices, realtimeResult.liveLocations]
  );
  const mapDecorations = useMemo(
    () => buildMapDecorations(devices, alarmsResult.alarms),
    [alarmsResult.alarms, devices]
  );

  const alarms = useMemo(() => {
    if (typeFilter === "all") {
      return alarmsResult.alarms;
    }

    return alarmsResult.alarms.filter((alarm) => alarm.type === typeFilter);
  }, [alarmsResult.alarms, typeFilter]);

  useEffect(() => {
    if (!selectedDeviceSN && alarms[0]) {
      setSelectedDeviceSN(alarms[0].device_sn);
      setFollowSelected(true);
      triggerFocus();
    }
  }, [alarms, selectedDeviceSN, setFollowSelected, setSelectedDeviceSN, triggerFocus]);

  const selectedAlarm =
    alarms.find((alarm) => alarm.device_sn === selectedDeviceSN) ?? alarms[0] ?? null;
  const selectedDevice =
    devices.find((device) => device.device_sn === selectedAlarm?.device_sn) ?? null;

  const headerMetrics = [
    {
      label: text.total,
      value: `${alarmsResult.alarms.length}`,
      tone: alarmsResult.alarms.length > 0 ? ("warn" as const) : ("default" as const),
    },
    {
      label: "SOS",
      value: `${alarmsResult.alarms.filter((alarm) => alarm.type === "sos").length}`,
      tone:
        alarmsResult.alarms.some((alarm) => alarm.type === "sos")
          ? ("danger" as const)
          : ("default" as const),
    },
    {
      label: text.online,
      value: `${devices.filter((item) => item.status !== 0).length}`,
    },
  ];

  return (
    <main className="min-h-screen p-4 md:p-5">
      <div className="grid min-h-[calc(100vh-2rem)] grid-rows-[auto_1fr] gap-4">
        <AppHeader
          mode={dataSource.mode}
          title={text.title}
          description={text.desc}
          metrics={headerMetrics}
          active="alarms"
        >
          <Link
            to={buildModePath(dataSource.mode, "/map")}
            className="rounded-full border border-black/8 bg-white/72 px-4 py-2 text-sm font-semibold text-[#10212b] transition hover:bg-white"
          >
            {text.backMap}
          </Link>
        </AppHeader>

        <section className="grid min-h-0 gap-4 xl:grid-cols-[420px_minmax(0,1fr)]">
          <aside className="glass-panel flex min-h-0 flex-col rounded-[28px] p-4">
            <div className="flex flex-wrap gap-2">
              {alarmTypes.map((type) => {
                const label = type === "all" ? text.all : getAlarmTypeView(type).label;

                return (
                  <button
                    key={type}
                    type="button"
                    onClick={() => setTypeFilter(type)}
                    className={`rounded-full px-4 py-2 text-xs font-semibold transition ${
                      typeFilter === type
                        ? "bg-[#10212b] text-white"
                        : "border border-black/8 bg-white/72 text-[#10212b] hover:bg-white"
                    }`}
                  >
                    {label}
                  </button>
                );
              })}
            </div>

            <div className="mt-4 min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
              {alarmsResult.isError ? (
                <div className="rounded-[24px] border border-[#d94747]/20 bg-[#d94747]/8 px-4 py-6 text-sm leading-7 text-[#9d2323]">
                  {alarmsResult.errorMessage ?? text.loadError}
                </div>
              ) : null}

              {!alarmsResult.isError && alarms.length === 0 ? (
                <div className="rounded-[24px] border border-dashed border-black/10 bg-white/56 px-4 py-6 text-sm leading-7 text-[#546570]">
                  {text.empty}
                </div>
              ) : null}

              {alarms.map((alarm) => {
                const typeView = getAlarmTypeView(alarm.type);
                const device =
                  devices.find((item) => item.device_sn === alarm.device_sn) ?? null;
                const active = selectedAlarm?.created_at === alarm.created_at;

                return (
                  <button
                    key={`${alarm.device_sn}-${alarm.created_at}-${alarm.type}`}
                    type="button"
                    onClick={() => {
                      setSelectedDeviceSN(alarm.device_sn);
                      setFollowSelected(true);
                      triggerFocus();
                    }}
                    className={`w-full rounded-[24px] border px-4 py-4 text-left transition ${
                      active
                        ? "border-[#1f88c9]/30 bg-[#1f88c9]/8 shadow-[0_12px_24px_rgba(31,136,201,0.12)]"
                        : "border-black/6 bg-white/66 hover:border-[#1f88c9]/20 hover:bg-white/84"
                    }`}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="text-sm font-semibold text-[#10212b]">
                          {device?.name || alarm.device_sn}
                        </div>
                        <div className="mt-1 text-xs text-[#546570]">{alarm.device_sn}</div>
                      </div>
                      <span
                        className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${
                          typeView.tone === "danger"
                            ? "bg-[#d94747]/12 text-[#9d2323]"
                            : typeView.tone === "warn"
                              ? "bg-[#d48a1f]/12 text-[#9d6412]"
                              : typeView.tone === "offline"
                                ? "bg-[#7c8b94]/12 text-[#51616a]"
                                : "bg-[#1f88c9]/12 text-[#176794]"
                        }`}
                      >
                        {typeView.label}
                      </span>
                    </div>

                    <div className="mt-3 text-sm leading-7 text-[#546570]">{alarm.content}</div>
                    <div className="mt-4 text-xs text-[#6a7a84]">
                      {formatDateTime(alarm.created_at)} · {formatRelativeTime(alarm.created_at)}
                    </div>
                  </button>
                );
              })}
            </div>
          </aside>

          <div className="grid min-h-0 gap-4 xl:grid-rows-[minmax(0,1fr)_220px]">
            <div className="glass-panel min-h-[460px] overflow-hidden rounded-[28px] p-3">
              <div className="h-full rounded-[24px] bg-[#dce9ef]">
                <DeviceMap
                  devices={devices}
                  points={livePoints}
                  selectedDeviceSN={selectedDevice?.device_sn ?? null}
                  emergencyDeviceSN={
                    selectedAlarm?.type === "sos" ? selectedAlarm.device_sn : null
                  }
                  fenceAlertDeviceSNs={mapDecorations.fenceAlertDeviceSNs}
                  geofenceOverlays={mapDecorations.geofenceOverlays}
                  followSelected
                  focusSequence={focusSequence}
                  onSelect={(deviceSN) => {
                    setSelectedDeviceSN(deviceSN);
                    setFollowSelected(true);
                    triggerFocus();
                  }}
                />
              </div>
            </div>

            <section className="glass-panel rounded-[28px] p-5">
              {selectedAlarm && selectedDevice ? (
                <>
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.22em] text-[#1f88c9]">
                        {text.focus}
                      </p>
                      <h3 className="mt-2 text-2xl font-semibold text-[#10212b]">
                        {selectedDevice.name || selectedDevice.device_sn}
                      </h3>
                      <p className="mt-1 text-sm text-[#546570]">{selectedDevice.device_sn}</p>
                    </div>
                    <button
                      type="button"
                      onClick={() =>
                        navigate(
                          buildModePath(
                            dataSource.mode,
                            `/devices/${selectedDevice.device_sn}/history`
                          )
                        )
                      }
                      className="rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b] transition hover:bg-white"
                    >
                      {text.openHistory}
                    </button>
                  </div>

                  <div className="mt-5 grid gap-3 sm:grid-cols-2">
                    <InfoCard label={text.type} value={getAlarmTypeView(selectedAlarm.type).label} />
                    <InfoCard label={text.occurredAt} value={formatDateTime(selectedAlarm.created_at)} />
                    <InfoCard label={text.lastOnline} value={formatRelativeTime(selectedDevice.last_online)} />
                    <InfoCard label={text.deviceStatus} value={getGPSStateLabel(selectedDevice.gps_state)} />
                  </div>

                  <div className="mt-5 rounded-[24px] bg-white/58 p-4 text-sm leading-7 text-[#546570]">
                    {selectedAlarm.content}
                  </div>
                </>
              ) : (
                <div className="flex h-full items-center justify-center text-sm text-[#546570]">
                  {text.placeholder}
                </div>
              )}
            </section>
          </div>
        </section>
      </div>
    </main>
  );
}

function InfoCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[22px] border border-black/6 bg-white/64 p-4">
      <div className="text-xs uppercase tracking-[0.18em] text-[#7a8a94]">{label}</div>
      <div className="mt-2 text-sm font-semibold text-[#10212b]">{value}</div>
    </div>
  );
}
