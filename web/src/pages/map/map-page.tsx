import { useEffect, useMemo } from "react";
import { DeviceDetailPanel } from "../../components/map/device-detail-panel";
import { DeviceMap } from "../../components/map/device-map";
import { DeviceSidebar } from "../../components/map/device-sidebar";
import { useAuth } from "../../hooks/use-auth";
import { isValidCoordinate } from "../../lib/geo";
import { authStore } from "../../stores/auth-store";
import { useMapStore } from "../../stores/map-store";
import { useMapDataSource } from "../../features/map-view/map-data-context";
import type { LiveDevicePoint } from "../../features/map-view/map-types";

export function MapPage() {
  const dataSource = useMapDataSource();
  const { user } = useAuth();
  const {
    selectedDeviceSN,
    searchText,
    followSelected,
    setSelectedDeviceSN,
    setSearchText,
    setFollowSelected,
  } = useMapStore();

  const devicesResult = dataSource.useDevices();
  const realtimeResult = dataSource.useRealtimeFeed();

  const devices = devicesResult.devices;
  const filteredDevices = useMemo(() => {
    const keyword = searchText.trim().toLowerCase();
    if (!keyword) {
      return devices;
    }

    return devices.filter((device) => {
      const haystack =
        `${device.name} ${device.device_sn} ${device.imei} ${device.iccid}`.toLowerCase();
      return haystack.includes(keyword);
    });
  }, [devices, searchText]);

  const selectedSN = selectedDeviceSN ?? filteredDevices[0]?.device_sn ?? null;
  const selectedDeviceResult = dataSource.useDeviceDetail(selectedSN);

  useEffect(() => {
    if (!selectedDeviceSN && filteredDevices[0]) {
      setSelectedDeviceSN(filteredDevices[0].device_sn);
    }
  }, [filteredDevices, selectedDeviceSN, setSelectedDeviceSN]);

  const livePoints = useMemo(() => {
    const points: LiveDevicePoint[] = [];

    for (const device of filteredDevices) {
      const liveLocation = realtimeResult.liveLocations[device.device_sn];
      if (liveLocation) {
        points.push({
          deviceSN: device.device_sn,
          name: device.name,
          lat: liveLocation.lat,
          lng: liveLocation.lng,
          battery: device.battery,
          status: device.status,
          gpsState: device.gps_state,
          lastUpdate: liveLocation.time ?? device.last_online,
        });
        continue;
      }

      const payload = device.status_payload ?? {};
      const lat = Number(payload.lat ?? payload.latitude);
      const lng = Number(payload.lng ?? payload.lon ?? payload.longitude);
      const point = {
        deviceSN: device.device_sn,
        name: device.name,
        lat,
        lng,
        battery: device.battery,
        status: device.status,
        gpsState: device.gps_state,
        lastUpdate: device.last_online,
      };

      if (isValidCoordinate(point)) {
        points.push(point);
      }
    }

    return points;
  }, [filteredDevices, realtimeResult.liveLocations]);

  const modeLabel = dataSource.mode === "demo" ? "死数据验证" : "后端联调";

  return (
    <main className="min-h-screen p-4 md:p-5">
      <div className="grid min-h-[calc(100vh-2rem)] grid-rows-[auto_1fr] gap-4">
        <header className="glass-panel flex flex-col gap-4 rounded-[28px] px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-[#1f88c9]">
              Locator Hub
            </p>
            <h1 className="mt-2 text-3xl font-semibold tracking-tight text-[#10212b]">
              团队位置管理系统
            </h1>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <TopMetric
              label="模式"
              value={modeLabel}
              tone={dataSource.mode === "demo" ? "warn" : "brand"}
            />
            <TopMetric
              label="在线"
              value={`${devices.filter((item) => item.status !== 0).length}`}
            />
            <TopMetric label="总人数" value={`${devices.length}`} />
            <TopMetric label="当前用户" value={user?.username ?? "演示访客"} />
            <TopMetric
              label="实时通道"
              value={realtimeResult.connected ? "已连接" : "未连接"}
            />
            {dataSource.mode === "live" ? (
              <button
                type="button"
                onClick={() => authStore.getState().clearSession()}
                className="rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b] transition hover:bg-white"
              >
                退出登录
              </button>
            ) : null}
          </div>
        </header>

        <section className="grid min-h-0 gap-4 xl:grid-cols-[340px_minmax(0,1fr)_360px]">
          <DeviceSidebar
            devices={filteredDevices}
            selectedDeviceSN={selectedSN}
            searchText={searchText}
            onSearchTextChange={setSearchText}
            onSelect={(deviceSN) => {
              setSelectedDeviceSN(deviceSN);
              setFollowSelected(true);
            }}
          />

          <div className="glass-panel min-h-[500px] overflow-hidden rounded-[28px] p-3">
            <div className="relative h-full min-h-[470px] rounded-[24px] bg-[#dce9ef]">
              <div className="absolute left-4 top-4 z-[1000] flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => setFollowSelected(!followSelected)}
                  className={`rounded-full px-4 py-2 text-xs font-semibold shadow-sm ${
                    followSelected
                      ? "bg-[#10212b] text-white"
                      : "bg-white/90 text-[#10212b]"
                  }`}
                >
                  {followSelected ? "跟随已开启" : "跟随已暂停"}
                </button>
                <div className="rounded-full bg-white/90 px-4 py-2 text-xs font-semibold text-[#10212b] shadow-sm">
                  OSM 模式
                </div>
                <div className="rounded-full bg-white/90 px-4 py-2 text-xs font-semibold text-[#10212b] shadow-sm">
                  已显示 {livePoints.length} 个实时点
                </div>
              </div>

              {devicesResult.isError ? (
                <div className="flex h-full items-center justify-center rounded-[24px] bg-white/60 p-8 text-center text-sm leading-7 text-[#9d2323]">
                  {devicesResult.errorMessage ?? "设备数据加载失败"}
                </div>
              ) : (
                <DeviceMap
                  devices={filteredDevices}
                  points={livePoints}
                  selectedDeviceSN={selectedSN}
                  followSelected={followSelected}
                  onSelect={(deviceSN) => {
                    setSelectedDeviceSN(deviceSN);
                    setFollowSelected(true);
                  }}
                />
              )}
            </div>
          </div>

          <DeviceDetailPanel device={selectedDeviceResult.device} />
        </section>
      </div>
    </main>
  );
}

function TopMetric({
  label,
  value,
  tone = "default",
}: {
  label: string;
  value: string;
  tone?: "default" | "brand" | "warn";
}) {
  const toneClassName =
    tone === "brand"
      ? "border-[#1f88c9]/18 bg-[#1f88c9]/8"
      : tone === "warn"
        ? "border-[#d48a1f]/18 bg-[#d48a1f]/10"
        : "border-black/6 bg-white/66";

  return (
    <div className={`rounded-2xl border px-4 py-2.5 ${toneClassName}`}>
      <div className="text-[11px] uppercase tracking-[0.18em] text-[#7a8a94]">
        {label}
      </div>
      <div className="mt-1 text-sm font-semibold text-[#10212b]">{value}</div>
    </div>
  );
}
