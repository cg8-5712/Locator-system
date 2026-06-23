import { useEffect, useMemo, useRef } from "react";
import L from "leaflet";
import {
  Circle,
  MapContainer,
  Marker,
  Popup,
  TileLayer,
  useMap,
} from "react-leaflet";
import { getGPSStateLabel, getMarkerAccent } from "../../lib/status";
import type { DeviceSummary } from "../../types/device";
import type { LiveDevicePoint } from "../../features/map-view/map-types";

function markerIcon(device: DeviceSummary, emergency: boolean) {
  const accent = getMarkerAccent(device);
  const label = (device.name || device.device_sn).trim().slice(0, 1).toUpperCase();

  return L.divIcon({
    className: "person-marker",
    html: `
      <div class="person-marker__outer${emergency ? " person-marker__outer--sos" : ""}" style="background:${accent}; box-shadow:0 0 0 6px ${accent}22;">
        <span class="person-marker__pulse${emergency ? " person-marker__pulse--sos" : ""}" style="background:${accent}33;"></span>
        <span class="person-marker__inner">${label || "?"}</span>
      </div>
    `,
    iconSize: [56, 56],
    iconAnchor: [28, 28],
  });
}

function MapFollower({
  selectedPoint,
  followSelected,
}: {
  selectedPoint: LiveDevicePoint | null;
  followSelected: boolean;
}) {
  const map = useMap();

  useEffect(() => {
    if (!selectedPoint || !followSelected) {
      return;
    }

    map.panTo([selectedPoint.lat, selectedPoint.lng], {
      animate: true,
      duration: 1,
    });
  }, [followSelected, map, selectedPoint]);

  return null;
}

function MapAutoFit({ points }: { points: LiveDevicePoint[] }) {
  const map = useMap();
  const initialized = useRef(false);

  useEffect(() => {
    if (initialized.current || points.length === 0) {
      return;
    }

    if (points.length === 1) {
      map.setView([points[0].lat, points[0].lng], 14);
    } else {
      const bounds = L.latLngBounds(
        points.map((point) => [point.lat, point.lng] as [number, number])
      );
      map.fitBounds(bounds.pad(0.28));
    }

    initialized.current = true;
  }, [map, points]);

  return null;
}

export function DeviceMap({
  devices,
  points,
  selectedDeviceSN,
  emergencyDeviceSN,
  followSelected,
  onSelect,
}: {
  devices: DeviceSummary[];
  points: LiveDevicePoint[];
  selectedDeviceSN: string | null;
  emergencyDeviceSN?: string | null;
  followSelected: boolean;
  onSelect: (deviceSN: string) => void;
}) {
  const pointsByDeviceSN = useMemo(
    () => new Map(points.map((point) => [point.deviceSN, point])),
    [points]
  );
  const selectedPoint =
    (selectedDeviceSN && pointsByDeviceSN.get(selectedDeviceSN)) || null;

  return (
    <MapContainer
      center={[39.9042, 116.4074]}
      zoom={11}
      className="h-full w-full rounded-[28px]"
      zoomControl={false}
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />

      <MapAutoFit points={points} />
      <MapFollower selectedPoint={selectedPoint} followSelected={followSelected} />

      {devices.map((device) => {
        const point = pointsByDeviceSN.get(device.device_sn);
        if (!point) {
          return null;
        }

        const emergency = emergencyDeviceSN === device.device_sn;

        return (
          <Marker
            key={device.device_sn}
            position={[point.lat, point.lng]}
            icon={markerIcon(device, emergency)}
            eventHandlers={{
              click: () => onSelect(device.device_sn),
            }}
          >
            <Popup>
              <div className="min-w-[200px] space-y-1">
                <div className="font-semibold text-[#10212b]">
                  {device.name || device.device_sn}
                </div>
                <div className="text-sm text-[#546570]">电量 {device.battery}%</div>
                <div className="text-sm text-[#546570]">
                  {getGPSStateLabel(device.gps_state)}
                </div>
              </div>
            </Popup>

            {selectedDeviceSN === device.device_sn ? (
              <Circle
                center={[point.lat, point.lng]}
                radius={point.accuracyMeters ?? 5}
                pathOptions={{
                  color: emergency ? "#d94747" : "#1f88c9",
                  fillColor: emergency ? "#d94747" : "#1f88c9",
                  fillOpacity: 0.08,
                  weight: 1,
                }}
              />
            ) : null}
          </Marker>
        );
      })}
    </MapContainer>
  );
}
