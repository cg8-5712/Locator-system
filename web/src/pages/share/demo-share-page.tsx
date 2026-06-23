import { useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import L from "leaflet";
import { Circle, MapContainer, Marker, Popup, TileLayer } from "react-leaflet";
import { mockDevices } from "../../features/map-view/mock-devices";
import { getActivityLabel, getGPSStateLabel, getMarkerAccent } from "../../lib/status";
import { formatRelativeTime } from "../../lib/time";

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

function readNumber(value: unknown) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function readString(value: unknown) {
  return typeof value === "string" && value.trim() ? value.trim() : null;
}

export function DemoSharePage() {
  const { deviceSN } = useParams<{ deviceSN: string }>();
  const [darkMap, setDarkMap] = useState(false);

  const device = mockDevices.find((item) => item.device_sn === deviceSN) ?? null;
  const payload = device?.status_payload ?? {};
  const lat = readNumber(payload.lat);
  const lng = readNumber(payload.lng);
  const accuracy = readNumber(payload.accuracy_m ?? 5) ?? 5;
  const address = readString(payload.address);
  const activity = getActivityLabel(payload.activity);

  const marker = useMemo(() => {
    if (!device) {
      return null;
    }

    const label = (device.name || device.device_sn).slice(0, 1).toUpperCase();
    return createMarkerIcon(label, getMarkerAccent(device));
  }, [device]);

  if (!device || lat === null || lng === null || !marker) {
    return (
      <main className="flex min-h-screen items-center justify-center px-6 py-10">
        <section className="glass-panel w-full max-w-xl rounded-[32px] p-8 text-center">
          <h1 className="text-3xl font-semibold text-[#10212b]">分享链接无效</h1>
          <p className="mt-4 text-sm leading-7 text-[#546570]">
            当前演示链接没有对应的 mock 设备数据。
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
              该页面用于演示分享看板交互，不包含真实密码校验、访问次数扣减和后端分享会话。
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <div className="rounded-2xl border border-[#1f88c9]/18 bg-[#1f88c9]/8 px-4 py-2.5">
              <div className="text-[11px] uppercase tracking-[0.18em] text-[#7a8a94]">
                有效期剩余
              </div>
              <div className="mt-1 text-sm font-semibold text-[#10212b]">42 分钟</div>
            </div>
            <button
              type="button"
              onClick={() => setDarkMap((current) => !current)}
              className="rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b] transition hover:bg-white"
            >
              {darkMap ? "切换浅色地图" : "切换暗色地图"}
            </button>
          </div>
        </header>

        <section className="glass-panel overflow-hidden rounded-[28px] p-3">
          <div className="grid min-h-[560px] gap-4 lg:grid-cols-[minmax(0,1fr)_340px]">
            <div className="overflow-hidden rounded-[24px] bg-[#dce9ef]">
              <MapContainer
                center={[lat, lng]}
                zoom={16}
                className="h-full w-full"
                zoomControl={false}
              >
                <TileLayer
                  attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
                  url={
                    darkMap
                      ? "https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
                      : "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                  }
                />
                <Marker position={[lat, lng]} icon={marker}>
                  <Popup>
                    <div className="space-y-1">
                      <div className="font-semibold text-[#10212b]">
                        {device.name || device.device_sn}
                      </div>
                      <div className="text-sm text-[#546570]">电量 {device.battery}%</div>
                    </div>
                  </Popup>
                </Marker>
                <Circle
                  center={[lat, lng]}
                  radius={accuracy}
                  pathOptions={{
                    color: "#1f88c9",
                    fillColor: "#1f88c9",
                    fillOpacity: 0.08,
                    weight: 1,
                  }}
                />
              </MapContainer>
            </div>

            <aside className="rounded-[24px] border border-black/6 bg-white/64 p-5">
              <p className="text-xs font-semibold uppercase tracking-[0.22em] text-[#1f88c9]">
                临时看板
              </p>
              <h2 className="mt-3 text-2xl font-semibold text-[#10212b]">
                {device.name} 的实时位置
              </h2>

              <div className="mt-5 space-y-3">
                <MetaRow label="电量" value={`${device.battery}%`} />
                <MetaRow label="状态" value={activity ?? getGPSStateLabel(device.gps_state)} />
                <MetaRow
                  label="最近更新"
                  value={formatRelativeTime(device.last_online)}
                />
                <MetaRow label="位置" value={address ?? `${lat}, ${lng}`} />
                <MetaRow label="定位精度" value={`${accuracy} 米`} />
              </div>
            </aside>
          </div>
        </section>

        <footer className="glass-panel rounded-[24px] px-5 py-4 text-sm leading-7 text-[#546570]">
          此链接受隐私保护的完整能力依赖后端分享会话、密码校验与访问限流。当前 demo 仅用于前端视觉和路由流程验证。
        </footer>
      </div>
    </main>
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
