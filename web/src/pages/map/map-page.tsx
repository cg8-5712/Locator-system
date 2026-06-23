import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { DeviceDetailPanel } from "../../components/map/device-detail-panel";
import { DeviceMap } from "../../components/map/device-map";
import { DeviceSidebar } from "../../components/map/device-sidebar";
import { EmergencyBanner } from "../../components/map/emergency-banner";
import { ShareLocationModal } from "../../components/map/share-location-modal";
import { AppHeader } from "../../components/shell/app-header";
import { buildMapDecorations } from "../../features/map-view/map-decorations";
import { useMapDataSource } from "../../features/map-view/map-data-context";
import { buildModePath } from "../../features/map-view/mode";
import { deriveLivePoints } from "../../features/map-view/points";
import { useMapStore } from "../../stores/map-store";
import type { AlarmEvent } from "../../types/realtime";

const text = {
  title: "团队位置管理系统",
  online: "在线",
  total: "总人数",
  realtime: "实时通道",
  realtimeConnected: "已连接",
  realtimeDisconnected: "未连接",
  followOn: "跟随已开启",
  followOff: "跟随已暂停",
  livePoints: "实时点位",
  loadError: "设备数据加载失败",
};

export function MapPage() {
  const navigate = useNavigate();
  const dataSource = useMapDataSource();
  const {
    selectedDeviceSN,
    searchText,
    followSelected,
    focusSequence,
    setSelectedDeviceSN,
    setSearchText,
    setFollowSelected,
    triggerFocus,
  } = useMapStore();
  const [shareOpen, setShareOpen] = useState(false);
  const [activeSOSAlarm, setActiveSOSAlarm] = useState<AlarmEvent | null>(null);

  const devicesResult = dataSource.useDevices();
  const realtimeResult = dataSource.useRealtimeFeed();
  const alarmsResult = dataSource.useAlarms({ limit: 50 });

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
  const selectedDevice = selectedDeviceResult.device;

  useEffect(() => {
    if (!selectedDeviceSN && filteredDevices[0]) {
      setSelectedDeviceSN(filteredDevices[0].device_sn);
    }
  }, [filteredDevices, selectedDeviceSN, setSelectedDeviceSN]);

  useEffect(() => {
    if (
      realtimeResult.lastMessage?.type === "alarm" &&
      realtimeResult.lastMessage.data.type === "sos"
    ) {
      setActiveSOSAlarm(realtimeResult.lastMessage.data);
      setSelectedDeviceSN(realtimeResult.lastMessage.data.device_sn);
      setFollowSelected(true);
      triggerFocus();
    }
  }, [realtimeResult.lastMessage, setFollowSelected, setSelectedDeviceSN, triggerFocus]);

  const livePoints = useMemo(
    () => deriveLivePoints(filteredDevices, realtimeResult.liveLocations),
    [filteredDevices, realtimeResult.liveLocations]
  );

  const deviceNameBySN = useMemo(
    () =>
      new Map(
        devices.map((device) => [device.device_sn, device.name || device.device_sn] as const)
      ),
    [devices]
  );

  const mapDecorations = useMemo(
    () => buildMapDecorations(devices, alarmsResult.alarms),
    [alarmsResult.alarms, devices]
  );

  const headerMetrics = [
    {
      label: text.online,
      value: `${devices.filter((item) => item.status !== 0).length}`,
    },
    {
      label: text.total,
      value: `${devices.length}`,
    },
    {
      label: text.realtime,
      value: realtimeResult.connected ? text.realtimeConnected : text.realtimeDisconnected,
      tone: realtimeResult.connected ? ("brand" as const) : ("warn" as const),
    },
  ];

  return (
    <main className="min-h-screen p-4 md:p-5 xl:h-screen xl:overflow-hidden">
      {activeSOSAlarm ? (
        <EmergencyBanner
          alarm={activeSOSAlarm}
          deviceName={deviceNameBySN.get(activeSOSAlarm.device_sn) ?? activeSOSAlarm.device_sn}
          onLocate={() => {
            setSelectedDeviceSN(activeSOSAlarm.device_sn);
            setFollowSelected(true);
            triggerFocus();
          }}
          onHistory={() =>
            navigate(
              buildModePath(dataSource.mode, `/devices/${activeSOSAlarm.device_sn}/history`)
            )
          }
          onDismiss={() => setActiveSOSAlarm(null)}
        />
      ) : null}

      <ShareLocationModal
        open={shareOpen}
        mode={dataSource.mode}
        device={selectedDevice}
        onClose={() => setShareOpen(false)}
      />

      <div className="grid min-h-[calc(100vh-2rem)] grid-rows-[auto_1fr] gap-4 xl:h-full xl:min-h-0">
        <AppHeader
          mode={dataSource.mode}
          title={text.title}
          metrics={headerMetrics}
          active="map"
        />

        <section className="grid min-h-0 gap-4 xl:h-full xl:grid-cols-[340px_minmax(0,1fr)_360px]">
          <DeviceSidebar
            devices={filteredDevices}
            selectedDeviceSN={selectedSN}
            searchText={searchText}
            onSearchTextChange={setSearchText}
            onSelect={(deviceSN) => {
              setSelectedDeviceSN(deviceSN);
              setFollowSelected(true);
              triggerFocus();
            }}
          />

          <div className="glass-panel min-h-[500px] overflow-hidden rounded-[28px] p-3 xl:h-full xl:min-h-0">
            <div className="relative h-full min-h-[470px] rounded-[24px] bg-[#dce9ef] xl:min-h-0">
              <div className="absolute left-4 top-4 z-[1000] flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => {
                    setFollowSelected(!followSelected);
                    if (!followSelected) {
                      triggerFocus();
                    }
                  }}
                  className={`rounded-full px-4 py-2 text-xs font-semibold shadow-sm ${
                    followSelected
                      ? "bg-[#10212b] text-white"
                      : "bg-white/90 text-[#10212b]"
                  }`}
                >
                  {followSelected ? text.followOn : text.followOff}
                </button>
                <div className="rounded-full bg-white/90 px-4 py-2 text-xs font-semibold text-[#10212b] shadow-sm">
                  {text.livePoints} {livePoints.length}
                </div>
              </div>

              {devicesResult.isError ? (
                <div className="flex h-full items-center justify-center rounded-[24px] bg-white/60 p-8 text-center text-sm leading-7 text-[#9d2323]">
                  {devicesResult.errorMessage ?? text.loadError}
                </div>
              ) : (
                <DeviceMap
                  devices={filteredDevices}
                  points={livePoints}
                  selectedDeviceSN={selectedSN}
                  emergencyDeviceSN={activeSOSAlarm?.device_sn ?? null}
                  fenceAlertDeviceSNs={mapDecorations.fenceAlertDeviceSNs}
                  geofenceOverlays={mapDecorations.geofenceOverlays}
                  followSelected={followSelected}
                  focusSequence={focusSequence}
                  onSelect={(deviceSN) => {
                    setSelectedDeviceSN(deviceSN);
                    setFollowSelected(true);
                    triggerFocus();
                  }}
                />
              )}
            </div>
          </div>

          <DeviceDetailPanel
            device={selectedDevice}
            mode={dataSource.mode}
            onOpenShare={() => setShareOpen(true)}
          />
        </section>
      </div>
    </main>
  );
}
