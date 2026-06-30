import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import L from "leaflet";
import { Circle, MapContainer, Marker, Polyline, Popup } from "react-leaflet";
import { useParams } from "react-router-dom";
import { AppTileLayer } from "../../components/map/app-tile-layer";
import { MapStyleSwitcher } from "../../components/map/map-style-switcher";
import { getGPSStateLabel, getMarkerAccent } from "../../lib/status";
import { formatDateTime, formatRelativeTime } from "../../lib/time";
import {
  buildPublicShareWsUrl,
  fetchPublicLocation,
  fetchPublicShare,
  fetchPublicTrack,
  verifyPublicShare,
} from "../../services/http/shares";
import { connectRealtimeByUrl } from "../../services/websocket/client";
import type { PublicLocationResult, ShareVerifyResult } from "../../types/share";
import type { RealtimeEnvelope } from "../../types/realtime";

function createMarkerIcon(label: string, accent: string) {
  return L.divIcon({
    className: "person-marker",
    html: `
      <div class="person-marker__outer" style="background:${accent}; box-shadow:0 0 0 6px ${accent}22;">
        <span class="person-marker__pulse" style="background:${accent}33;"></span>
        <span class="person-marker__inner">${label}</span>
      </div>
    `,
    iconSize: [56, 56],
    iconAnchor: [28, 28],
  });
}

function getViewerId(shareCode: string) {
  const key = `locator-public-viewer:${shareCode}`;
  const existing = window.localStorage.getItem(key);
  if (existing) {
    return existing;
  }

  const next = `viewer-${Math.random().toString(36).slice(2, 10)}`;
  window.localStorage.setItem(key, next);
  return next;
}

function getStoredAccessToken(shareCode: string) {
  return window.sessionStorage.getItem(`locator-public-token:${shareCode}`);
}

function storeAccessToken(shareCode: string, token: string) {
  window.sessionStorage.setItem(`locator-public-token:${shareCode}`, token);
}

function clearAccessToken(shareCode: string) {
  window.sessionStorage.removeItem(`locator-public-token:${shareCode}`);
}

export function PublicSharePage() {
  const { shareCode } = useParams<{ shareCode: string }>();
  const [password, setPassword] = useState("");
  const [accessToken, setAccessToken] = useState<string | null>(
    shareCode ? getStoredAccessToken(shareCode) : null
  );
  const [verifyError, setVerifyError] = useState<string | null>(null);
  const [verifying, setVerifying] = useState(false);
  const [liveLocation, setLiveLocation] = useState<PublicLocationResult | null>(null);

  const publicShareQuery = useQuery({
    queryKey: ["public-share", shareCode],
    queryFn: () => fetchPublicShare(shareCode!),
    enabled: Boolean(shareCode),
    retry: false,
  });

  const locationQuery = useQuery({
    queryKey: ["public-share-location", shareCode, accessToken],
    queryFn: () => fetchPublicLocation(shareCode!, accessToken!),
    enabled: Boolean(shareCode && accessToken),
    retry: false,
    refetchInterval: 15_000,
  });

  const trackQuery = useQuery({
    queryKey: ["public-share-track", shareCode, accessToken],
    queryFn: () => fetchPublicTrack(shareCode!, accessToken!),
    enabled:
      Boolean(shareCode && accessToken) &&
      publicShareQuery.data?.share_mode === "today_track",
    retry: false,
    refetchInterval: 60_000,
  });

  useEffect(() => {
    if (!shareCode || !publicShareQuery.data || accessToken) {
      return;
    }

    if (!publicShareQuery.data.requires_password) {
      void handleVerify("");
    }
  }, [accessToken, publicShareQuery.data, shareCode]);

  useEffect(() => {
    if (!shareCode || !accessToken) {
      return;
    }

    let socket: WebSocket | null = null;
    socket = connectRealtimeByUrl(buildPublicShareWsUrl(shareCode, accessToken), {
      onMessage: (message: RealtimeEnvelope) => {
        if (message.type === "location") {
          setLiveLocation((current) => ({
            device_sn: message.data.device_sn,
            device_name: current?.device_name || publicShareQuery.data?.device_name || message.data.device_sn,
            battery: current?.battery ?? 0,
            gps_state: message.data.gps_state,
            status: message.data.status,
            last_online: message.data.time,
            last_fix_at: message.data.time,
            lat: message.data.lat,
            lng: message.data.lng,
            time: message.data.time,
            still_seconds: message.data.still_seconds,
            accuracy_m: current?.accuracy_m ?? 5,
            address: current?.address,
            activity: current?.activity,
          }));
        }

        if (message.type === "device_status") {
          setLiveLocation((current) => ({
            device_sn: message.data.device_sn,
            device_name: current?.device_name || publicShareQuery.data?.device_name || message.data.device_sn,
            battery: message.data.battery,
            gps_state: message.data.gps_state,
            status: message.data.status,
            last_online: message.data.last_online ?? current?.last_online,
            last_fix_at: message.data.last_fix_at ?? current?.last_fix_at,
            lat: current?.lat,
            lng: current?.lng,
            time: current?.time,
            still_seconds: current?.still_seconds ?? 0,
            accuracy_m: current?.accuracy_m,
            address:
              typeof message.data.status_payload?.address === "string"
                ? message.data.status_payload.address
                : current?.address,
            activity:
              typeof message.data.status_payload?.activity === "string"
                ? message.data.status_payload.activity
                : current?.activity,
          }));
        }
      },
      onClose: () => undefined,
      onError: () => undefined,
    });

    return () => {
      socket?.close();
    };
  }, [accessToken, publicShareQuery.data?.device_name, shareCode]);

  useEffect(() => {
    if (locationQuery.error && shareCode) {
      clearAccessToken(shareCode);
      setAccessToken(null);
    }
  }, [locationQuery.error, shareCode]);

  const effectiveLocation = liveLocation ?? locationQuery.data ?? null;

  const marker = useMemo(() => {
    if (!effectiveLocation) {
      return null;
    }

    const label = (effectiveLocation.device_name || effectiveLocation.device_sn)
      .trim()
      .slice(0, 1)
      .toUpperCase();
    return createMarkerIcon(
      label,
      getMarkerAccent({
        status: effectiveLocation.status,
        gps_state: effectiveLocation.gps_state as never,
        battery: effectiveLocation.battery,
      })
    );
  }, [effectiveLocation]);

  const trackPolyline = useMemo(
    () =>
      (trackQuery.data?.tracks ?? []).map((track) => [track.lat, track.lng] as [number, number]),
    [trackQuery.data?.tracks]
  );

  async function handleVerify(nextPassword?: string) {
    if (!shareCode) {
      return;
    }

    setVerifying(true);
    setVerifyError(null);
    try {
      const result: ShareVerifyResult = await verifyPublicShare(shareCode, {
        viewer_id: getViewerId(shareCode),
        password: nextPassword ?? password,
      });

      storeAccessToken(shareCode, result.access_token);
      setAccessToken(result.access_token);
    } catch (error) {
      setVerifyError(error instanceof Error ? error.message : "分享验证失败");
    } finally {
      setVerifying(false);
    }
  }

  if (!shareCode) {
    return null;
  }

  if (publicShareQuery.isLoading) {
    return (
      <main className="flex min-h-screen items-center justify-center px-6 py-10">
        <section className="glass-panel w-full max-w-xl rounded-[32px] p-8 text-center text-sm text-[#546570]">
          正在加载分享信息...
        </section>
      </main>
    );
  }

  if (publicShareQuery.isError || !publicShareQuery.data) {
    return (
      <main className="flex min-h-screen items-center justify-center px-6 py-10">
        <section className="glass-panel w-full max-w-xl rounded-[32px] p-8 text-center">
          <h1 className="text-3xl font-semibold text-[#10212b]">分享链接不可用</h1>
          <p className="mt-4 text-sm leading-7 text-[#546570]">
            {publicShareQuery.error instanceof Error
              ? publicShareQuery.error.message
              : "当前分享链接不存在或已失效。"}
          </p>
        </section>
      </main>
    );
  }

  return (
    <main className="min-h-screen p-4 md:p-5">
      <div className="grid min-h-[calc(100vh-2rem)] grid-rows-[auto_1fr_auto] gap-4">
        <header className="glass-panel flex flex-col gap-4 rounded-[28px] px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-[#1f88c9]">
              Locator Hub
            </p>
            <h1 className="mt-2 text-3xl font-semibold tracking-tight text-[#10212b]">
              安全位置分享
            </h1>
            <p className="mt-2 text-sm leading-7 text-[#546570]">
              仅展示被授权对象的位置与轨迹，不包含后台管理能力。
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <MetricCard label="分享对象" value={publicShareQuery.data.device_name || publicShareQuery.data.device_sn} />
            <MetricCard label="模式" value={publicShareQuery.data.share_mode === "today_track" ? "实时 + 今日轨迹" : "仅实时"} />
            <MetricCard label="有效期至" value={formatDateTime(publicShareQuery.data.expires_at)} />
          </div>
        </header>

        {!accessToken ? (
          <section className="glass-panel mx-auto flex w-full max-w-lg flex-col rounded-[28px] p-6">
            <h2 className="text-2xl font-semibold text-[#10212b]">验证访问权限</h2>
            <p className="mt-3 text-sm leading-7 text-[#546570]">
              此分享链接受密码、次数与有效期控制。验证通过后，同一浏览器在有效期内刷新不会重复消耗访问次数。
            </p>

            {publicShareQuery.data.requires_password ? (
              <label className="mt-6 block">
                <span className="mb-2 block text-sm font-medium text-[#10212b]">访问密码</span>
                <input
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  className="w-full rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none transition focus:border-[#1f88c9] focus:ring-4 focus:ring-[#1f88c9]/10"
                />
              </label>
            ) : null}

            {verifyError ? (
              <div className="mt-4 rounded-2xl border border-[#d94747]/20 bg-[#d94747]/8 px-4 py-3 text-sm text-[#9d2323]">
                {verifyError}
              </div>
            ) : null}

            <button
              type="button"
              onClick={() => void handleVerify()}
              disabled={verifying}
              className="mt-6 rounded-2xl bg-[#10212b] px-5 py-3 text-sm font-semibold text-white transition hover:bg-[#163445] disabled:cursor-not-allowed disabled:opacity-60"
            >
              {verifying ? "验证中..." : "进入分享看板"}
            </button>
          </section>
        ) : (
          <section className="glass-panel overflow-hidden rounded-[28px] p-3">
            <div className="grid min-h-[560px] gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
              <div className="relative overflow-hidden rounded-[24px] bg-[#dce9ef]">
                <div className="absolute bottom-4 left-4 right-4 z-[1000]">
                  <MapStyleSwitcher compact />
                </div>

                {effectiveLocation?.lat != null && effectiveLocation.lng != null && marker ? (
                  <MapContainer
                    center={[effectiveLocation.lat, effectiveLocation.lng]}
                    zoom={16}
                    className="h-full w-full"
                    zoomControl={false}
                  >
                    <AppTileLayer />
                    {trackPolyline.length > 1 ? (
                      <Polyline
                        positions={trackPolyline}
                        pathOptions={{
                          color: "#1f88c9",
                          weight: 4,
                          opacity: 0.68,
                        }}
                      />
                    ) : null}
                    <Marker position={[effectiveLocation.lat, effectiveLocation.lng]} icon={marker}>
                      <Popup>
                        <div className="space-y-1">
                          <div className="font-semibold text-[#10212b]">
                            {effectiveLocation.device_name || effectiveLocation.device_sn}
                          </div>
                          <div className="text-sm text-[#546570]">
                            电量 {effectiveLocation.battery}%
                          </div>
                        </div>
                      </Popup>
                    </Marker>
                    <Circle
                      center={[effectiveLocation.lat, effectiveLocation.lng]}
                      radius={effectiveLocation.accuracy_m ?? 5}
                      pathOptions={{
                        color: "#1f88c9",
                        fillColor: "#1f88c9",
                        fillOpacity: 0.08,
                        weight: 1,
                      }}
                    />
                  </MapContainer>
                ) : (
                  <div className="flex h-full items-center justify-center px-8 text-center text-sm text-[#546570]">
                    当前位置暂不可用，设备可能在线但尚未获得有效定位。
                  </div>
                )}
              </div>

              <aside className="rounded-[24px] border border-black/6 bg-white/64 p-5">
                <p className="text-xs font-semibold uppercase tracking-[0.22em] text-[#1f88c9]">
                  临时看板
                </p>
                <h2 className="mt-3 text-2xl font-semibold text-[#10212b]">
                  {publicShareQuery.data.device_name || publicShareQuery.data.device_sn}
                </h2>

                <div className="mt-5 space-y-3">
                  <MetaRow label="电量" value={`${effectiveLocation?.battery ?? 0}%`} />
                  <MetaRow
                    label="状态"
                    value={getGPSStateLabel(effectiveLocation?.gps_state)}
                  />
                  <MetaRow
                    label="最近更新"
                    value={formatRelativeTime(effectiveLocation?.last_online ?? undefined)}
                  />
                  <MetaRow
                    label="位置"
                    value={effectiveLocation?.address || "未提供地址描述"}
                  />
                  <MetaRow
                    label="精度"
                    value={`${effectiveLocation?.accuracy_m ?? 5} 米`}
                  />
                </div>
              </aside>
            </div>
          </section>
        )}

        <footer className="glass-panel rounded-[24px] px-5 py-4 text-sm leading-7 text-[#546570]">
          该分享链接受隐私保护。若页面提示访问被拒绝、次数用尽或链接过期，请联系分享人重新获取授权。
        </footer>
      </div>
    </main>
  );
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-[#1f88c9]/18 bg-[#1f88c9]/8 px-4 py-2.5">
      <div className="text-[11px] uppercase tracking-[0.18em] text-[#7a8a94]">{label}</div>
      <div className="mt-1 text-sm font-semibold text-[#10212b]">{value}</div>
    </div>
  );
}

function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-white/72 px-4 py-3">
      <div className="text-xs uppercase tracking-[0.16em] text-[#7a8a94]">{label}</div>
      <div className="mt-2 text-sm font-medium leading-6 text-[#10212b]">{value}</div>
    </div>
  );
}
