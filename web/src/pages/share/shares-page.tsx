import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { AppHeader } from "../../components/shell/app-header";
import { useMapDataSource } from "../../features/map-view/map-data-context";
import { buildModePath } from "../../features/map-view/mode";
import { formatDateTime, formatRelativeTime } from "../../lib/time";
import type { DeviceSummary } from "../../types/device";

type ShareStatus = "active" | "expiring" | "expired" | "quota_used";
type ShareMode = "live_only" | "today_track";

interface ShareRecord {
  id: string;
  deviceSN: string;
  deviceName: string;
  mode: ShareMode;
  status: ShareStatus;
  passwordEnabled: boolean;
  password: string | null;
  maxVisits: number | null;
  usedVisits: number;
  expiresAt: string;
  createdAt: string;
  lastAccessAt: string | null;
  url: string;
  note: string;
}

const text = {
  title: "分享管理",
  desc: "统一查看位置分享链接、有效期、访问次数和预览入口。当前 live 模式先展示前端数据结构，待分享接口落地后可直接替换。",
  active: "生效中",
  expiring: "即将过期",
  expired: "已过期",
  usedUp: "额度用尽",
  createHint: "新建分享",
  backMap: "返回地图",
  empty: "当前没有可展示的分享记录。",
  previewMode: "前端预览",
  liveMode: "待接后端",
  copyLink: "复制链接",
  openPreview: "打开预览",
  visits: "访问额度",
  expiresAt: "过期时间",
  lastAccess: "最近访问",
  createdAt: "创建时间",
  policy: "访问策略",
  target: "分享对象",
  type: "分享内容",
  password: "访问密码",
  note: "备注",
  unlimited: "不限次数",
  noPassword: "未启用",
};

function createShareRecords(mode: "demo" | "live", devices: DeviceSummary[]): ShareRecord[] {
  const now = Date.now();

  function getName(deviceSN: string) {
    const device = devices.find((item) => item.device_sn === deviceSN);
    return device?.name || deviceSN;
  }

  function buildPreviewUrl(deviceSN: string, shareCode: string) {
    return `${window.location.origin}/demo/share/${deviceSN}?code=${shareCode}`;
  }

  const records: ShareRecord[] = [
    {
      id: "share-001",
      deviceSN: "locator-esp32s3-001",
      deviceName: getName("locator-esp32s3-001"),
      mode: "live_only",
      status: "active",
      passwordEnabled: true,
      password: "888999",
      maxVisits: 5,
      usedVisits: 1,
      expiresAt: new Date(now + 55 * 60 * 1000).toISOString(),
      createdAt: new Date(now - 5 * 60 * 1000).toISOString(),
      lastAccessAt: new Date(now - 2 * 60 * 1000).toISOString(),
      url: buildPreviewUrl("locator-esp32s3-001", "a7f9g2e"),
      note: "适合临时外部访客查看实时位置。",
    },
    {
      id: "share-002",
      deviceSN: "locator-esp32s3-002",
      deviceName: getName("locator-esp32s3-002"),
      mode: "today_track",
      status: "expiring",
      passwordEnabled: false,
      password: null,
      maxVisits: 20,
      usedVisits: 12,
      expiresAt: new Date(now + 18 * 60 * 1000).toISOString(),
      createdAt: new Date(now - 42 * 60 * 1000).toISOString(),
      lastAccessAt: new Date(now - 6 * 60 * 1000).toISOString(),
      url: buildPreviewUrl("locator-esp32s3-002", "k2m4v1q"),
      note: "开放今日轨迹，适合班组长回看巡检路线。",
    },
    {
      id: "share-003",
      deviceSN: "locator-esp32s3-003",
      deviceName: getName("locator-esp32s3-003"),
      mode: "live_only",
      status: "quota_used",
      passwordEnabled: true,
      password: "512314",
      maxVisits: 3,
      usedVisits: 3,
      expiresAt: new Date(now + 3 * 60 * 60 * 1000).toISOString(),
      createdAt: new Date(now - 88 * 60 * 1000).toISOString(),
      lastAccessAt: new Date(now - 11 * 60 * 1000).toISOString(),
      url: buildPreviewUrl("locator-esp32s3-003", "m8t2u6w"),
      note: "额度已耗尽，常见于链接被二次转发。",
    },
    {
      id: "share-004",
      deviceSN: "locator-esp32s3-001",
      deviceName: getName("locator-esp32s3-001"),
      mode: "today_track",
      status: "expired",
      passwordEnabled: true,
      password: "673821",
      maxVisits: null,
      usedVisits: 2,
      expiresAt: new Date(now - 90 * 60 * 1000).toISOString(),
      createdAt: new Date(now - 5 * 60 * 60 * 1000).toISOString(),
      lastAccessAt: new Date(now - 95 * 60 * 1000).toISOString(),
      url: buildPreviewUrl("locator-esp32s3-001", "q1x7d8n"),
      note: "已过期记录可用于审计和复盘。",
    },
  ];

  if (mode === "demo") {
    return records;
  }

  return records.map((record) => ({
    ...record,
    note: `${record.note} 当前为 live 前端占位数据，等待分享 API 接入。`,
  }));
}

function getStatusView(status: ShareStatus) {
  switch (status) {
    case "active":
      return { label: "生效中", className: "bg-[#2f9e68]/12 text-[#20724c]" };
    case "expiring":
      return { label: "即将过期", className: "bg-[#d48a1f]/12 text-[#9d6412]" };
    case "quota_used":
      return { label: "额度用尽", className: "bg-[#d94747]/12 text-[#9d2323]" };
    default:
      return { label: "已过期", className: "bg-[#7c8b94]/12 text-[#51616a]" };
  }
}

function getModeLabel(mode: ShareMode) {
  return mode === "today_track" ? "实时位置 + 今日轨迹" : "仅实时位置";
}

export function SharesPage() {
  const dataSource = useMapDataSource();
  const devicesResult = dataSource.useDevices();
  const records = useMemo(
    () => createShareRecords(dataSource.mode, devicesResult.devices),
    [dataSource.mode, devicesResult.devices]
  );
  const [selectedShareId, setSelectedShareId] = useState<string | null>(records[0]?.id ?? null);

  useEffect(() => {
    if (!records.some((record) => record.id === selectedShareId)) {
      setSelectedShareId(records[0]?.id ?? null);
    }
  }, [records, selectedShareId]);

  const selectedRecord =
    records.find((record) => record.id === selectedShareId) ?? records[0] ?? null;

  const headerMetrics = [
    {
      label: text.active,
      value: `${records.filter((record) => record.status === "active").length}`,
      tone: records.some((record) => record.status === "active")
        ? ("brand" as const)
        : ("default" as const),
    },
    {
      label: text.expiring,
      value: `${records.filter((record) => record.status === "expiring").length}`,
      tone: records.some((record) => record.status === "expiring")
        ? ("warn" as const)
        : ("default" as const),
    },
    {
      label: text.usedUp,
      value: `${records.filter((record) => record.status === "quota_used").length}`,
      tone: records.some((record) => record.status === "quota_used")
        ? ("danger" as const)
        : ("default" as const),
    },
  ];

  async function handleCopyLink() {
    if (!selectedRecord || !navigator.clipboard?.writeText) {
      return;
    }

    await navigator.clipboard.writeText(selectedRecord.url);
  }

  return (
    <main className="min-h-screen p-4 md:p-5">
      <div className="grid min-h-[calc(100vh-2rem)] grid-rows-[auto_1fr] gap-4">
        <AppHeader
          mode={dataSource.mode}
          title={text.title}
          description={text.desc}
          metrics={headerMetrics}
          active="shares"
        >
          <Link
            to={buildModePath(dataSource.mode, "/map")}
            className="rounded-full border border-black/8 bg-white/72 px-4 py-2 text-sm font-semibold text-[#10212b] transition hover:bg-white"
          >
            {text.backMap}
          </Link>
        </AppHeader>

        <section className="grid min-h-0 gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
          <aside className="glass-panel flex min-h-0 flex-col rounded-[28px] p-4">
            <div className="flex items-center justify-between gap-3 px-1">
              <div>
                <div className="text-sm font-semibold text-[#10212b]">{text.createHint}</div>
                <div className="mt-1 text-xs leading-5 text-[#546570]">
                  {dataSource.mode === "demo" ? text.previewMode : text.liveMode}
                </div>
              </div>
              <Link
                to={buildModePath(dataSource.mode, "/map")}
                className="rounded-full bg-[#10212b] px-4 py-2 text-sm font-semibold text-white transition hover:bg-[#163445]"
              >
                去地图创建
              </Link>
            </div>

            <div className="mt-4 min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
              {records.length === 0 ? (
                <div className="rounded-[24px] border border-dashed border-black/10 bg-white/56 px-4 py-6 text-sm leading-7 text-[#546570]">
                  {text.empty}
                </div>
              ) : null}

              {records.map((record) => {
                const active = record.id === selectedRecord?.id;
                const statusView = getStatusView(record.status);

                return (
                  <button
                    key={record.id}
                    type="button"
                    onClick={() => setSelectedShareId(record.id)}
                    className={`w-full rounded-[24px] border px-4 py-4 text-left transition ${
                      active
                        ? "border-[#1f88c9]/30 bg-[#1f88c9]/8 shadow-[0_12px_24px_rgba(31,136,201,0.12)]"
                        : "border-black/6 bg-white/66 hover:border-[#1f88c9]/20 hover:bg-white/84"
                    }`}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="text-sm font-semibold text-[#10212b]">
                          {record.deviceName}
                        </div>
                        <div className="mt-1 text-xs text-[#546570]">{record.deviceSN}</div>
                      </div>
                      <span
                        className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${statusView.className}`}
                      >
                        {statusView.label}
                      </span>
                    </div>

                    <div className="mt-3 text-sm text-[#10212b]">{getModeLabel(record.mode)}</div>
                    <div className="mt-2 text-xs text-[#6a7a84]">
                      过期于 {formatDateTime(record.expiresAt)}
                    </div>
                  </button>
                );
              })}
            </div>
          </aside>

          <section className="glass-panel flex min-h-0 flex-col rounded-[28px] p-5">
            {selectedRecord ? (
              <>
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.22em] text-[#1f88c9]">
                      Share Detail
                    </p>
                    <h2 className="mt-2 text-2xl font-semibold text-[#10212b]">
                      {selectedRecord.deviceName}
                    </h2>
                    <p className="mt-1 text-sm text-[#546570]">{selectedRecord.deviceSN}</p>
                  </div>
                  <span
                    className={`inline-flex rounded-full px-3 py-1.5 text-xs font-semibold ${
                      getStatusView(selectedRecord.status).className
                    }`}
                  >
                    {getStatusView(selectedRecord.status).label}
                  </span>
                </div>

                <div className="mt-5 space-y-3">
                  <InfoCard label={text.target} value={selectedRecord.deviceName} />
                  <InfoCard label={text.type} value={getModeLabel(selectedRecord.mode)} />
                  <InfoCard
                    label={text.policy}
                    value={
                      selectedRecord.maxVisits === null
                        ? "密码可选，不限访问次数"
                        : `最多 ${selectedRecord.maxVisits} 次访问`
                    }
                  />
                  <InfoCard
                    label={text.password}
                    value={
                      selectedRecord.passwordEnabled
                        ? selectedRecord.password || "--"
                        : text.noPassword
                    }
                  />
                  <InfoCard label={text.expiresAt} value={formatDateTime(selectedRecord.expiresAt)} />
                  <InfoCard
                    label={text.visits}
                    value={
                      selectedRecord.maxVisits === null
                        ? text.unlimited
                        : `${selectedRecord.usedVisits}/${selectedRecord.maxVisits}`
                    }
                  />
                  <InfoCard
                    label={text.lastAccess}
                    value={formatRelativeTime(selectedRecord.lastAccessAt ?? undefined)}
                  />
                  <InfoCard label={text.createdAt} value={formatDateTime(selectedRecord.createdAt)} />
                  <InfoCard label="URL" value={selectedRecord.url} />
                  <InfoCard label={text.note} value={selectedRecord.note} />
                </div>

                <div className="mt-6 flex flex-wrap gap-3">
                  <button
                    type="button"
                    onClick={() => void handleCopyLink()}
                    className="rounded-2xl bg-[#10212b] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#163445]"
                  >
                    {text.copyLink}
                  </button>
                  <a
                    href={selectedRecord.url}
                    target="_blank"
                    rel="noreferrer"
                    className="rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b] transition hover:bg-white"
                  >
                    {text.openPreview}
                  </a>
                </div>
              </>
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-[#546570]">
                {text.empty}
              </div>
            )}
          </section>
        </section>
      </div>
    </main>
  );
}

function InfoCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[22px] border border-black/6 bg-white/64 p-4">
      <div className="text-xs uppercase tracking-[0.18em] text-[#7a8a94]">{label}</div>
      <div className="mt-2 break-all text-sm font-semibold leading-6 text-[#10212b]">
        {value}
      </div>
    </div>
  );
}
