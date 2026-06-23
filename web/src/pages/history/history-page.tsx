import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { TrackMap } from "../../components/map/track-map";
import { AppHeader } from "../../components/shell/app-header";
import { useMapDataSource } from "../../features/map-view/map-data-context";
import { buildModePath, getModeLabel } from "../../features/map-view/mode";
import { formatDateTime, formatDurationSeconds, formatRelativeTime } from "../../lib/time";

const ranges = [
  { label: "近 1 小时", hours: 1 },
  { label: "近 6 小时", hours: 6 },
  { label: "近 24 小时", hours: 24 },
] as const;

export function HistoryPage() {
  const { deviceSN } = useParams<{ deviceSN: string }>();
  const dataSource = useMapDataSource();
  const [rangeHours, setRangeHours] = useState<number>(6);
  const [selectedTrackTime, setSelectedTrackTime] = useState<string | null>(null);

  const deviceResult = dataSource.useDeviceDetail(deviceSN ?? null);
  const trackResult = dataSource.useTrack(deviceSN ?? null, { rangeHours });
  const alarmsResult = dataSource.useAlarms({ deviceSN: deviceSN ?? null, limit: 8 });

  const tracks = trackResult.tracks;
  const selectedTrack =
    tracks.find((track) => track.time === selectedTrackTime) ??
    tracks[tracks.length - 1] ??
    null;

  useEffect(() => {
    if (tracks.length > 0) {
      setSelectedTrackTime(tracks[tracks.length - 1].time);
    }
  }, [rangeHours, tracks]);

  const stopCount = useMemo(
    () => tracks.filter((track) => track.still_seconds > 0).length,
    [tracks]
  );

  const totalStillSeconds = useMemo(
    () => tracks.reduce((sum, track) => sum + Math.max(0, track.still_seconds), 0),
    [tracks]
  );

  const headerMetrics = [
    {
      label: "模式",
      value: getModeLabel(dataSource.mode),
      tone: dataSource.mode === "demo" ? ("warn" as const) : ("brand" as const),
    },
    {
      label: "轨迹点",
      value: `${tracks.length}`,
    },
    {
      label: "停留点",
      value: `${stopCount}`,
      tone: stopCount > 0 ? ("warn" as const) : ("default" as const),
    },
    {
      label: "相关告警",
      value: `${alarmsResult.alarms.length}`,
    },
  ];

  return (
    <main className="min-h-screen p-4 md:p-5">
      <div className="grid min-h-[calc(100vh-2rem)] grid-rows-[auto_1fr] gap-4">
        <AppHeader
          mode={dataSource.mode}
          title={`${deviceResult.device?.name || deviceSN || "设备"} · 历史轨迹`}
          description="支持 demo 死数据验证和 live 后端联调两套入口，共享同一套轨迹页面与 Leaflet 轨迹渲染逻辑。"
          metrics={headerMetrics}
          active="history"
        >
          <Link
            to={buildModePath(dataSource.mode, "/map")}
            className="rounded-full border border-black/8 bg-white/72 px-4 py-2 text-sm font-semibold text-[#10212b] transition hover:bg-white"
          >
            返回地图
          </Link>
          <Link
            to={buildModePath(dataSource.mode, "/alarms")}
            className="rounded-full border border-black/8 bg-white/72 px-4 py-2 text-sm font-semibold text-[#10212b] transition hover:bg-white"
          >
            查看告警
          </Link>
        </AppHeader>

        <section className="grid min-h-0 gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
          <div className="glass-panel min-h-[540px] overflow-hidden rounded-[28px] p-3">
            <div className="flex h-full flex-col gap-3">
              <div className="flex flex-wrap gap-2 px-1 pt-1">
                {ranges.map((range) => (
                  <button
                    key={range.hours}
                    type="button"
                    onClick={() => setRangeHours(range.hours)}
                    className={`rounded-full px-4 py-2 text-xs font-semibold transition ${
                      rangeHours === range.hours
                        ? "bg-[#10212b] text-white"
                        : "border border-black/8 bg-white/72 text-[#10212b] hover:bg-white"
                    }`}
                  >
                    {range.label}
                  </button>
                ))}
              </div>

              <div className="min-h-[420px] flex-1 rounded-[24px] bg-[#dce9ef]">
                {trackResult.isError ? (
                  <div className="flex h-full items-center justify-center px-8 text-center text-sm leading-7 text-[#9d2323]">
                    {trackResult.errorMessage ?? "轨迹数据加载失败"}
                  </div>
                ) : tracks.length === 0 ? (
                  <div className="flex h-full items-center justify-center px-8 text-center text-sm leading-7 text-[#546570]">
                    当前时间范围内没有轨迹点。
                  </div>
                ) : (
                  <TrackMap
                    deviceName={deviceResult.device?.name || deviceSN || "设备"}
                    tracks={tracks}
                    selectedTrackTime={selectedTrackTime}
                    onSelectTrack={setSelectedTrackTime}
                  />
                )}
              </div>
            </div>
          </div>

          <aside className="glass-panel flex min-h-0 flex-col rounded-[28px] p-5">
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
              <InfoCard
                label="最近在线"
                value={formatRelativeTime(deviceResult.device?.last_online)}
              />
              <InfoCard
                label="可信定位"
                value={formatRelativeTime(deviceResult.device?.last_fix_at)}
              />
              <InfoCard label="停留总时长" value={formatDurationSeconds(totalStillSeconds)} />
              <InfoCard
                label="当前选中点"
                value={selectedTrack ? formatDateTime(selectedTrack.time) : "--"}
              />
            </div>

            <div className="mt-6">
              <h3 className="text-sm font-semibold text-[#10212b]">轨迹时间线</h3>
              <div className="mt-3 min-h-0 space-y-3 overflow-y-auto pr-1 xl:max-h-[420px]">
                {tracks.map((track) => {
                  const active = track.time === selectedTrack?.time;

                  return (
                    <button
                      key={track.time}
                      type="button"
                      onClick={() => setSelectedTrackTime(track.time)}
                      className={`w-full rounded-[22px] border px-4 py-3 text-left transition ${
                        active
                          ? "border-[#1f88c9]/30 bg-[#1f88c9]/8 shadow-[0_12px_24px_rgba(31,136,201,0.12)]"
                          : "border-black/6 bg-white/66 hover:border-[#1f88c9]/20 hover:bg-white/84"
                      }`}
                    >
                      <div className="text-sm font-semibold text-[#10212b]">
                        {formatDateTime(track.time)}
                      </div>
                      <div className="mt-2 text-xs text-[#546570]">
                        坐标 {track.lat.toFixed(6)}, {track.lng.toFixed(6)}
                      </div>
                      <div className="mt-2 text-xs text-[#6a7a84]">
                        停留时长 {formatDurationSeconds(track.still_seconds)}
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>
          </aside>
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
