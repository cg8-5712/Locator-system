import { useEffect, useMemo } from "react";
import L from "leaflet";
import {
  CircleMarker,
  MapContainer,
  Polyline,
  Popup,
  TileLayer,
  useMap,
} from "react-leaflet";
import type { TrackPoint } from "../../types/device";
import { formatDateTime, formatDurationSeconds } from "../../lib/time";

function FitTrackBounds({ tracks }: { tracks: TrackPoint[] }) {
  const map = useMap();

  useEffect(() => {
    if (tracks.length === 0) {
      return;
    }

    if (tracks.length === 1) {
      map.setView([tracks[0].lat, tracks[0].lng], 16);
      return;
    }

    const bounds = L.latLngBounds(
      tracks.map((track) => [track.lat, track.lng] as [number, number])
    );
    map.fitBounds(bounds.pad(0.2));
  }, [map, tracks]);

  return null;
}

export function TrackMap({
  deviceName,
  tracks,
  selectedTrackTime,
  onSelectTrack,
}: {
  deviceName: string;
  tracks: TrackPoint[];
  selectedTrackTime: string | null;
  onSelectTrack: (time: string) => void;
}) {
  const polyline = useMemo(
    () => tracks.map((track) => [track.lat, track.lng] as [number, number]),
    [tracks]
  );

  const startTrack = tracks[0];
  const endTrack = tracks[tracks.length - 1];

  return (
    <MapContainer
      center={[39.9042, 116.4074]}
      zoom={13}
      className="h-full w-full rounded-[28px]"
      zoomControl={false}
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />

      <FitTrackBounds tracks={tracks} />

      {polyline.length > 1 ? (
        <Polyline
          positions={polyline}
          pathOptions={{
            color: "#1f88c9",
            weight: 5,
            opacity: 0.82,
          }}
        />
      ) : null}

      {tracks.map((track) => {
        const selected = selectedTrackTime === track.time;
        const isStart = startTrack?.time === track.time;
        const isEnd = endTrack?.time === track.time;
        const color = isStart
          ? "#2f9e68"
          : isEnd
            ? "#10212b"
            : track.still_seconds > 0
              ? "#d48a1f"
              : "#1f88c9";

        return (
          <CircleMarker
            key={track.time}
            center={[track.lat, track.lng]}
            radius={selected ? 8 : track.still_seconds > 0 ? 7 : 5}
            pathOptions={{
              color,
              fillColor: color,
              fillOpacity: selected ? 0.96 : 0.82,
              weight: isStart || isEnd ? 2 : 1,
            }}
            eventHandlers={{
              click: () => onSelectTrack(track.time),
            }}
          >
            <Popup>
              <div className="space-y-1">
                <div className="font-semibold text-[#10212b]">{deviceName}</div>
                <div className="text-sm text-[#546570]">{formatDateTime(track.time)}</div>
                <div className="text-sm text-[#546570]">
                  停留时长 {formatDurationSeconds(track.still_seconds)}
                </div>
                {isStart ? <div className="text-sm text-[#2f9e68]">轨迹起点</div> : null}
                {isEnd ? <div className="text-sm text-[#10212b]">轨迹终点</div> : null}
              </div>
            </Popup>
          </CircleMarker>
        );
      })}
    </MapContainer>
  );
}
