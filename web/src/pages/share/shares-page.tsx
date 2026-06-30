import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { AppHeader } from "../../components/shell/app-header";
import { useMapDataSource } from "../../features/map-view/map-data-context";
import { buildModePath } from "../../features/map-view/mode";
import { useShares } from "../../hooks/use-shares";
import { formatDateTime, formatRelativeTime } from "../../lib/time";
import { revokeShare } from "../../services/http/shares";
import type { ShareSummary } from "../../types/share";
import { mockDevices } from "../../features/map-view/mock-devices";

function createDemoShares(): ShareSummary[] {
  const now = Date.now();
  const deviceBySN = new Map(
    mockDevices.map((device) => [device.device_sn, device.name || device.device_sn] as const)
  );

  return [
    {
      id: 1,
      device_sn: "locator-esp32s3-001",
      device_name: deviceBySN.get("locator-esp32s3-001") ?? "locator-esp32s3-001",
      share_code: "demoa7f9",
      share_mode: "live_only",
      requires_password: true,
      note: "demo 分享：仅实时位置。",
      expires_at: new Date(now + 55 * 60 * 1000).toISOString(),
      max_visits: 5,
      visit_count: 1,
      remaining_visits: 4,
      last_access_at: new Date(now - 2 * 60 * 1000).toISOString(),
      created_at: new Date(now - 5 * 60 * 1000).toISOString(),
      status: "active",
    },
    {
      id: 2,
      device_sn: "locator-esp32s3-002",
      device_name: deviceBySN.get("locator-esp32s3-002") ?? "locator-esp32s3-002",
      share_code: "demok2m4",
      share_mode: "today_track",
      requires_password: false,
      note: "demo 分享：允许查看今日轨迹。",
      expires_at: new Date(now + 18 * 60 * 1000).toISOString(),
      max_visits: 20,
      visit_count: 12,
      remaining_visits: 8,
      last_access_at: new Date(now - 6 * 60 * 1000).toISOString(),
      created_at: new Date(now - 42 * 60 * 1000).toISOString(),
      status: "expiring",
    },
    {
      id: 3,
      device_sn: "locator-esp32s3-003",
      device_name: deviceBySN.get("locator-esp32s3-003") ?? "locator-esp32s3-003",
      share_code: "demom8t2",
      share_mode: "live_only",
      requires_password: true,
      note: "demo 分享：额度已耗尽。",
      expires_at: new Date(now + 3 * 60 * 60 * 1000).toISOString(),
      max_visits: 3,
      visit_count: 3,
      remaining_visits: 0,
      last_access_at: new Date(now - 11 * 60 * 1000).toISOString(),
      created_at: new Date(now - 88 * 60 * 1000).toISOString(),
      status: "quota_used",
    },
  ];
}

function getStatusView(status: ShareSummary["status"]) {
  switch (status) {
    case "active":
      return { label: "生效中", className: "bg-[#2f9e68]/12 text-[#20724c]" };
    case "expiring":
      return { label: "即将过期", className: "bg-[#d48a1f]/12 text-[#9d6412]" };
    case "quota_used":
      return { label: "额度用尽", className: "bg-[#d94747]/12 text-[#9d2323]" };
    case "revoked":
      return { label: "已撤销", className: "bg-[#7c8b94]/12 text-[#51616a]" };
    default:
      return { label: "已过期", className: "bg-[#7c8b94]/12 text-[#51616a]" };
  }
}

function getModeLabel(mode: ShareSummary["share_mode"]) {
  return mode === "today_track" ? "实时位置 + 今日轨迹" : "仅实时位置";
}

function buildPreviewUrl(mode: "demo" | "live", share: ShareSummary) {
  return mode === "demo"
    ? `${window.location.origin}/demo/share/${share.device_sn}?code=${share.share_code}`
    : `${window.location.origin}/share/${share.share_code}`;
}

export function SharesPage() {
  const dataSource = useMapDataSource();
  const sharesQuery = useShares(dataSource.mode === "live" ? undefined : null);
  const queryClient = useQueryClient();
  const [selectedShareId, setSelectedShareId] = useState<number | null>(null);

  const revokeMutation = useMutation({
    mutationFn: revokeShare,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["shares"] });
    },
  });

  const shares = useMemo(
    () => (dataSource.mode === "demo" ? createDemoShares() : sharesQuery.data?.shares ?? []),
    [dataSource.mode, sharesQuery.data?.shares]
  );

  const selectedShare =
    shares.find((share) => share.id === selectedShareId) ?? shares[0] ?? null;

  const headerMetrics = [
    {
      label: "生效中",
      value: `${shares.filter((share) => share.status === "active").length}`,
      tone: shares.some((share) => share.status === "active")
        ? ("brand" as const)
        : ("default" as const),
    },
    {
      label: "即将过期",
      value: `${shares.filter((share) => share.status === "expiring").length}`,
      tone: shares.some((share) => share.status === "expiring")
        ? ("warn" as const)
        : ("default" as const),
    },
    {
      label: "异常/停用",
      value: `${shares.filter((share) => ["quota_used", "expired", "revoked"].includes(share.status)).length}`,
      tone: shares.some((share) => ["quota_used", "expired", "revoked"].includes(share.status))
        ? ("danger" as const)
        : ("default" as const),
    },
  ];

  async function handleCopyLink() {
    if (!selectedShare || !navigator.clipboard?.writeText) {
      return;
    }

    await navigator.clipboard.writeText(buildPreviewUrl(dataSource.mode, selectedShare));
  }

  return (
    <main className="min-h-screen p-4 md:p-5">
      <div className="grid min-h-[calc(100vh-2rem)] grid-rows-[auto_1fr] gap-4">
        <AppHeader
          mode={dataSource.mode}
          title="分享管理"
          description="查看真实分享记录、有效期、访问额度和公开入口。地图页里的“分享实时位置”会直接写入这里。"
          metrics={headerMetrics}
          active="shares"
        >
          <Link
            to={buildModePath(dataSource.mode, "/map")}
            className="rounded-full border border-black/8 bg-white/72 px-4 py-2 text-sm font-semibold text-[#10212b] transition hover:bg-white"
          >
            返回地图
          </Link>
        </AppHeader>

        <section className="grid min-h-0 gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
          <aside className="glass-panel flex min-h-0 flex-col rounded-[28px] p-4">
            <div className="flex items-center justify-between gap-3 px-1">
              <div>
                <div className="text-sm font-semibold text-[#10212b]">分享记录</div>
                <div className="mt-1 text-xs leading-5 text-[#546570]">
                  {dataSource.mode === "demo"
                    ? "demo 模式仍保留假数据流程验证。"
                    : "live 模式已接入真实后端分享记录。"}
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
              {dataSource.mode === "live" && sharesQuery.isLoading ? (
                <div className="rounded-[24px] border border-dashed border-black/10 bg-white/56 px-4 py-6 text-sm leading-7 text-[#546570]">
                  正在加载分享记录...
                </div>
              ) : null}

              {dataSource.mode === "live" && sharesQuery.isError ? (
                <div className="rounded-[24px] border border-[#d94747]/20 bg-[#d94747]/8 px-4 py-6 text-sm leading-7 text-[#9d2323]">
                  {sharesQuery.error instanceof Error
                    ? sharesQuery.error.message
                    : "加载分享记录失败"}
                </div>
              ) : null}

              {!(dataSource.mode === "live" && sharesQuery.isLoading) &&
              !(dataSource.mode === "live" && sharesQuery.isError) &&
              shares.length === 0 ? (
                <div className="rounded-[24px] border border-dashed border-black/10 bg-white/56 px-4 py-6 text-sm leading-7 text-[#546570]">
                  当前没有分享记录。
                </div>
              ) : null}

              {shares.map((share) => {
                const active = share.id === selectedShare?.id;
                const statusView = getStatusView(share.status);

                return (
                  <button
                    key={share.id}
                    type="button"
                    onClick={() => setSelectedShareId(share.id)}
                    className={`w-full rounded-[24px] border px-4 py-4 text-left transition ${
                      active
                        ? "border-[#1f88c9]/30 bg-[#1f88c9]/8 shadow-[0_12px_24px_rgba(31,136,201,0.12)]"
                        : "border-black/6 bg-white/66 hover:border-[#1f88c9]/20 hover:bg-white/84"
                    }`}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="text-sm font-semibold text-[#10212b]">
                          {share.device_name || share.device_sn}
                        </div>
                        <div className="mt-1 text-xs text-[#546570]">{share.device_sn}</div>
                      </div>
                      <span
                        className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${statusView.className}`}
                      >
                        {statusView.label}
                      </span>
                    </div>

                    <div className="mt-3 text-sm text-[#10212b]">
                      {getModeLabel(share.share_mode)}
                    </div>
                    <div className="mt-2 text-xs text-[#6a7a84]">
                      过期于 {formatDateTime(share.expires_at)}
                    </div>
                  </button>
                );
              })}
            </div>
          </aside>

          <section className="glass-panel flex min-h-0 flex-col rounded-[28px] p-5">
            {selectedShare ? (
              <>
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.22em] text-[#1f88c9]">
                      Share Detail
                    </p>
                    <h2 className="mt-2 text-2xl font-semibold text-[#10212b]">
                      {selectedShare.device_name || selectedShare.device_sn}
                    </h2>
                    <p className="mt-1 text-sm text-[#546570]">{selectedShare.device_sn}</p>
                  </div>
                  <span
                    className={`inline-flex rounded-full px-3 py-1.5 text-xs font-semibold ${
                      getStatusView(selectedShare.status).className
                    }`}
                  >
                    {getStatusView(selectedShare.status).label}
                  </span>
                </div>

                <div className="mt-5 space-y-3">
                  <InfoCard label="分享对象" value={selectedShare.device_name || selectedShare.device_sn} />
                  <InfoCard label="分享内容" value={getModeLabel(selectedShare.share_mode)} />
                  <InfoCard label="分享码" value={selectedShare.share_code} />
                  <InfoCard
                    label="访问策略"
                    value={
                      selectedShare.max_visits == null
                        ? "密码可选，不限访问次数"
                        : `最多 ${selectedShare.max_visits} 次访问`
                    }
                  />
                  <InfoCard
                    label="访问密码"
                    value={selectedShare.requires_password ? "已启用" : "未启用"}
                  />
                  <InfoCard label="过期时间" value={formatDateTime(selectedShare.expires_at)} />
                  <InfoCard
                    label="访问额度"
                    value={
                      selectedShare.max_visits == null
                        ? "不限次数"
                        : `${selectedShare.visit_count}/${selectedShare.max_visits}`
                    }
                  />
                  <InfoCard
                    label="最近访问"
                    value={formatRelativeTime(selectedShare.last_access_at ?? undefined)}
                  />
                  <InfoCard label="创建时间" value={formatDateTime(selectedShare.created_at)} />
                  <InfoCard label="URL" value={buildPreviewUrl(dataSource.mode, selectedShare)} />
                  <InfoCard label="备注" value={selectedShare.note || "--"} />
                </div>

                <div className="mt-6 flex flex-wrap gap-3">
                  <button
                    type="button"
                    onClick={() => void handleCopyLink()}
                    className="rounded-2xl bg-[#10212b] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#163445]"
                  >
                    复制链接
                  </button>
                  <a
                    href={buildPreviewUrl(dataSource.mode, selectedShare)}
                    target="_blank"
                    rel="noreferrer"
                    className="rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b] transition hover:bg-white"
                  >
                    打开预览
                  </a>
                  {dataSource.mode === "live" ? (
                    <button
                      type="button"
                      onClick={() => revokeMutation.mutate(selectedShare.id)}
                      disabled={revokeMutation.isPending || selectedShare.status === "revoked"}
                      className="rounded-2xl border border-[#d94747]/20 bg-[#d94747]/8 px-4 py-2.5 text-sm font-semibold text-[#9d2323] transition hover:bg-[#d94747]/12 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                      撤销分享
                    </button>
                  ) : null}
                </div>
              </>
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-[#546570]">
                当前没有分享记录。
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
