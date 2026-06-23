import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { TrackMap } from "../../components/map/track-map";
import { AppHeader } from "../../components/shell/app-header";
import { useMapDataSource } from "../../features/map-view/map-data-context";
import { buildModePath, getModeLabel } from "../../features/map-view/mode";
import { formatDateTime, formatDurationSeconds, formatRelativeTime } from "../../lib/time";

const ranges = [
  { label: "\u8fd1 1 \u5c0f\u65f6", hours: 1 },
  { label: "\u8fd1 6 \u5c0f\u65f6", hours: 6 },
  { label: "\u8fd1 24 \u5c0f\u65f6", hours: 24 },
] as const;

const text = {
  mode: "\u6a21\u5f0f",
  points: "\u8f68\u8ff9\u70b9",
  stops: "\u505c\u7559\u70b9",
  relatedAlarms: "\u76f8\u5173\u544a\u8b66",
  desc:
    "\u652f\u6301 demo \u6b7b\u6570\u636e\u9a8c\u8bc1\u548c live \u540e\u7aef\u8054\u8c03\u4e24\u5957\u5165\u53e3\uff0c\u5171\u4eab\u540c\u4e00\u5957\u8f68\u8ff9\u9875\u9762\u4e0e Leaflet \u8f68\u8ff9\u6e32\u67d3\u903b\u8f91\u3002",
  backMap: "\u8fd4\u56de\u5730\u56fe",
  viewAlarms: "\u67e5\u770b\u544a\u8b66",
  loadError: "\u8f68\u8ff9\u6570\u636e\u52a0\u8f7d\u5931\u8d25",
  empty: "\u5f53\u524d\u65f6\u95f4\u8303\u56f4\u5185\u6ca1\u6709\u8f68\u8ff9\u70b9\u3002",
  lastOnline: "\u6700\u8fd1\u5728\u7ebf",
  trustedFix: "\u53ef\u4fe1\u5b9a\u4f4d",
  totalStill: "\u505c\u7559\u603b\u65f6\u957f",
  currentPoint: "\u5f53\u524d\u9009\u4e2d\u70b9",
  timeline: "\u8f68\u8ff9\u65f6\u95f4\u7ebf",
  coord: "\u5750\u6807",
  still: "\u505c\u7559\u65f6\u957f",
};

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
      label: text.mode,
      value: getModeLabel(dataSource.mode),
      tone: dataSource.mode === "demo" ? ("warn" as const) : ("brand" as const),
    },
    {
      label: text.points,
      value: `${tracks.length}`,
    },
    {
      label: text.stops,
      value: `${stopCount}`,
      tone: stopCount > 0 ? ("warn" as const) : ("default" as const),
    },
    {
      label: text.relatedAlarms,
      value: `${alarmsResult.alarms.length}`,
    },
  ];

  const deviceName = deviceResult.device?.name || deviceSN || "\u8bbe\u5907";

  return (
    <main className="min-h-screen p-4 md:p-5">
      <div className="grid min-h-[calc(100vh-2rem)] grid-rows-[auto_1fr] gap-4">
        <AppHeader
          mode={dataSource.mode}
          title={`${deviceName} · \u5386\u53f2\u8f68\u8ff9`}
          description={text.desc}
          metrics={headerMetrics}
          active="history"
        >
          <Link
            to={buildModePath(dataSource.mode, "/map")}
            className="rounded-full border border-black/8 bg-white/72 px-4 py-2 text-sm font-semibold text-[#10212b] transition hover:bg-white"
          >
            {text.backMap}
          </Link>
          <Link
            to={buildModePath(dataSource.mode, "/alarms")}
            className="rounded-full border border-black/8 bg-white/72 px-4 py-2 text-sm font-semibold text-[#10212b] transition hover:bg-white"
          >
            {text.viewAlarms}
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
                    {trackResult.errorMessage ?? text.loadError}
                  </div>
                ) : tracks.length === 0 ? (
                  <div className="flex h-full items-center justify-center px-8 text-center text-sm leading-7 text-[#546570]">
                    {text.empty}
                  </div>
                ) : (
                  <TrackMap
                    deviceName={deviceName}
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
                label={text.lastOnline}
                value={formatRelativeTime(deviceResult.device?.last_online)}
              />
              <InfoCard
                label={text.trustedFix}
                value={formatRelativeTime(deviceResult.device?.last_fix_at)}
              />
              <InfoCard label={text.totalStill} value={formatDurationSeconds(totalStillSeconds)} />
              <InfoCard
                label={text.currentPoint}
                value={selectedTrack ? formatDateTime(selectedTrack.time) : "--"}
              />
            </div>

            <div className="mt-6">
              <h3 className="text-sm font-semibold text-[#10212b]">{text.timeline}</h3>
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
                        {text.coord} {track.lat.toFixed(6)}, {track.lng.toFixed(6)}
                      </div>
                      <div className="mt-2 text-xs text-[#6a7a84]">
                        {text.still} {formatDurationSeconds(track.still_seconds)}
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
