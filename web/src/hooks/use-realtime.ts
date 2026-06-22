import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { connectRealtime } from "../services/websocket/client";
import { useAuth } from "./use-auth";
import { useMapStore } from "../stores/map-store";
import type { DeviceListResult, DeviceSummary } from "../types/device";
import type { AlarmListResult } from "../types/alarm";
import type { RealtimeEnvelope } from "../types/realtime";

function mergeDeviceSummary(
  current: DeviceSummary,
  incoming: Partial<DeviceSummary> & { device_sn: string }
): DeviceSummary {
  return {
    ...current,
    ...incoming,
  };
}

export function useRealtime() {
  const { token } = useAuth();
  const queryClient = useQueryClient();
  const setWsConnected = useMapStore((state) => state.setWsConnected);
  const upsertLiveLocation = useMapStore((state) => state.upsertLiveLocation);

  useEffect(() => {
    if (!token) {
      setWsConnected(false);
      return;
    }

    let reconnectTimer: number | undefined;
    let isDisposed = false;
    let socket: WebSocket | null = null;

    function applyEnvelope(message: RealtimeEnvelope) {
      if (message.type === "location") {
        upsertLiveLocation(message.data.device_sn, {
          lat: message.data.lat,
          lng: message.data.lng,
          time: message.data.time,
          stillSeconds: message.data.still_seconds,
        });

        queryClient.setQueryData<DeviceListResult | undefined>(
          ["devices"],
          (current) => {
            if (!current) {
              return current;
            }

            return {
              ...current,
              devices: current.devices.map((device) =>
                device.device_sn === message.data.device_sn
                  ? mergeDeviceSummary(device, {
                      device_sn: message.data.device_sn,
                      gps_state: message.data.gps_state as DeviceSummary["gps_state"],
                      status: message.data.status,
                      last_fix_at: message.data.time,
                      last_online: message.data.time,
                    })
                  : device
              ),
            };
          }
        );

        queryClient.setQueryData<DeviceSummary | undefined>(
          ["device", message.data.device_sn],
          (current) => {
            if (!current) {
              return current;
            }

            return mergeDeviceSummary(current, {
              device_sn: message.data.device_sn,
              gps_state: message.data.gps_state as DeviceSummary["gps_state"],
              status: message.data.status,
              last_fix_at: message.data.time,
              last_online: message.data.time,
            });
          }
        );
      }

      if (message.type === "device_status") {
        queryClient.setQueryData<DeviceListResult | undefined>(
          ["devices"],
          (current) => {
            if (!current) {
              return current;
            }

            return {
              ...current,
              devices: current.devices.map((device) =>
                device.device_sn === message.data.device_sn
                  ? mergeDeviceSummary(device, {
                      device_sn: message.data.device_sn,
                      gps_state: message.data.gps_state as DeviceSummary["gps_state"],
                      status: message.data.status,
                      battery: message.data.battery,
                      imei: message.data.imei ?? device.imei,
                      iccid: message.data.iccid ?? device.iccid,
                      status_payload:
                        message.data.status_payload ?? device.status_payload,
                      config_payload:
                        message.data.config_payload ?? device.config_payload,
                      status_updated_at:
                        message.data.status_updated_at ?? device.status_updated_at,
                      config_updated_at:
                        message.data.config_updated_at ?? device.config_updated_at,
                      last_online: message.data.last_online ?? device.last_online,
                      last_fix_at: message.data.last_fix_at ?? device.last_fix_at,
                    })
                  : device
              ),
            };
          }
        );

        queryClient.setQueryData<DeviceSummary | undefined>(
          ["device", message.data.device_sn],
          (current) => {
            if (!current) {
              return current;
            }

            return mergeDeviceSummary(current, {
              device_sn: message.data.device_sn,
              gps_state: message.data.gps_state as DeviceSummary["gps_state"],
              status: message.data.status,
              battery: message.data.battery,
              imei: message.data.imei ?? current.imei,
              iccid: message.data.iccid ?? current.iccid,
              status_payload: message.data.status_payload ?? current.status_payload,
              config_payload: message.data.config_payload ?? current.config_payload,
              status_updated_at:
                message.data.status_updated_at ?? current.status_updated_at,
              config_updated_at:
                message.data.config_updated_at ?? current.config_updated_at,
              last_online: message.data.last_online ?? current.last_online,
              last_fix_at: message.data.last_fix_at ?? current.last_fix_at,
            });
          }
        );
      }

      if (message.type === "alarm") {
        queryClient.setQueryData<AlarmListResult | undefined>(
          ["recent-alarms"],
          (current) => {
            if (!current) {
              return current;
            }

            return {
              ...current,
              alarms: [
                {
                  device_sn: message.data.device_sn,
                  type: message.data.type,
                  content: message.data.content,
                  created_at: message.data.created_at,
                },
                ...current.alarms,
              ].slice(0, 10),
            };
          }
        );
      }
    }

    function openSocket() {
      socket = connectRealtime(token, {
        onMessage: applyEnvelope,
        onOpen: () => {
          setWsConnected(true);
        },
        onClose: () => {
          setWsConnected(false);
          if (isDisposed) {
            return;
          }

          reconnectTimer = window.setTimeout(openSocket, 3000);
        },
        onError: () => {
          setWsConnected(false);
          socket?.close();
        },
      });
    }

    openSocket();

    return () => {
      isDisposed = true;
      setWsConnected(false);
      if (reconnectTimer) {
        window.clearTimeout(reconnectTimer);
      }
      socket?.close();
    };
  }, [queryClient, setWsConnected, token, upsertLiveLocation]);
}
