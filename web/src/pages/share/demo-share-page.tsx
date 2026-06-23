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

const text = {
  invalid: "\u5206\u4eab\u94fe\u63a5\u65e0\u6548",
  invalidDesc:
    "\u5f53\u524d\u6f14\u793a\u94fe\u63a5\u6ca1\u6709\u5bf9\u5e94\u7684 mock \u8bbe\u5907\u6570\u636e\u3002",
  title: "\u5b89\u5168\u4f4d\u7f6e\u5206\u4eab",
  desc:
    "\u8be5\u9875\u9762\u7528\u4e8e\u6f14\u793a\u5206\u4eab\u770b\u677f\u4ea4\u4e92\uff0c\u4e0d\u5305\u542b\u771f\u5b9e\u5bc6\u7801\u6821\u9a8c\u3001\u8bbf\u95ee\u6b21\u6570\u6263\u51cf\u548c\u540e\u7aef\u5206\u4eab\u4f1a\u8bdd\u3002",
  expires: "\u6709\u6548\u671f\u5269\u4f59",
  expiresValue: "42 \u5206\u949f",
  lightMap: "\u5207\u6362\u6d45\u8272\u5730\u56fe",
  darkMap: "\u5207\u6362\u6697\u8272\u5730\u56fe",
  board: "\u4e34\u65f6\u770b\u677f",
  realtime: "\u7684\u5b9e\u65f6\u4f4d\u7f6e",
  battery: "\u7535\u91cf",
  state: "\u72b6\u6001",
  lastUpdate: "\u6700\u8fd1\u66f4\u65b0",
  location: "\u4f4d\u7f6e",
  accuracy: "\u5b9a\u4f4d\u7cbe\u5ea6",
  footer:
    "\u6b64\u94fe\u63a5\u53d7\u9690\u79c1\u4fdd\u62a4\u7684\u5b8c\u6574\u80fd\u529b\u4f9d\u8d56\u540e\u7aef\u5206\u4eab\u4f1a\u8bdd\u3001\u5bc6\u7801\u6821\u9a8c\u4e0e\u8bbf\u95ee\u9650\u6d41\u3002\u5f53\u524d demo \u4ec5\u7528\u4e8e\u524d\u7aef\u89c6\u89c9\u548c\u8def\u7531\u6d41\u7a0b\u9a8c\u8bc1\u3002",
};

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
          <h1 className="text-3xl font-semibold text-[#10212b]">{text.invalid}</h1>
          <p className="mt-4 text-sm leading-7 text-[#546570]">{text.invalidDesc}</p>
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
              {text.title}
            </h1>
            <p className="mt-2 text-sm leading-7 text-[#546570]">{text.desc}</p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <div className="rounded-2xl border border-[#1f88c9]/18 bg-[#1f88c9]/8 px-4 py-2.5">
              <div className="text-[11px] uppercase tracking-[0.18em] text-[#7a8a94]">
                {text.expires}
              </div>
              <div className="mt-1 text-sm font-semibold text-[#10212b]">
                {text.expiresValue}
              </div>
            </div>
            <button
              type="button"
              onClick={() => setDarkMap((current) => !current)}
              className="rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b] transition hover:bg-white"
            >
              {darkMap ? text.lightMap : text.darkMap}
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
                      <div className="text-sm text-[#546570]">
                        {text.battery} {device.battery}%
                      </div>
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
                {text.board}
              </p>
              <h2 className="mt-3 text-2xl font-semibold text-[#10212b]">
                {device.name}
                {text.realtime}
              </h2>

              <div className="mt-5 space-y-3">
                <MetaRow label={text.battery} value={`${device.battery}%`} />
                <MetaRow
                  label={text.state}
                  value={activity ?? getGPSStateLabel(device.gps_state)}
                />
                <MetaRow
                  label={text.lastUpdate}
                  value={formatRelativeTime(device.last_online)}
                />
                <MetaRow label={text.location} value={address ?? `${lat}, ${lng}`} />
                <MetaRow label={text.accuracy} value={`${accuracy} \u7c73`} />
              </div>
            </aside>
          </div>
        </section>

        <footer className="glass-panel rounded-[24px] px-5 py-4 text-sm leading-7 text-[#546570]">
          {text.footer}
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
