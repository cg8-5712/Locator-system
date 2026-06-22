import { useQuery } from "@tanstack/react-query";
import { fetchDevice, fetchDevices } from "../services/http/devices";

export function useDeviceList() {
  return useQuery({
    queryKey: ["devices"],
    queryFn: () =>
      fetchDevices({
        page: 1,
        page_size: 100,
      }),
    staleTime: 15_000,
  });
}

export function useDeviceDetail(deviceSN: string | null) {
  return useQuery({
    queryKey: ["device", deviceSN],
    queryFn: () => fetchDevice(deviceSN!),
    enabled: Boolean(deviceSN),
    staleTime: 15_000,
  });
}
