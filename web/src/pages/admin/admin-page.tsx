import { useEffect, useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AppHeader } from "../../components/shell/app-header";
import { useDeviceList } from "../../hooks/use-devices";
import { useFences } from "../../hooks/use-fences";
import { useMQTTMessages, useMQTTStatus } from "../../hooks/use-mqtt";
import { useShares } from "../../hooks/use-shares";
import { useUsers } from "../../hooks/use-users";
import {
  createDevice,
  deleteDevice,
  sendDeviceCommand,
  updateDevice,
} from "../../services/http/devices";
import {
  createFence,
  deleteFence,
  updateFence,
} from "../../services/http/fences";
import {
  createUser,
  deleteUser,
  updateUser,
} from "../../services/http/users";
import { formatDateTime, formatRelativeTime } from "../../lib/time";
import { getGPSStateLabel } from "../../lib/status";
import { authStore } from "../../stores/auth-store";
import type { DeviceSummary } from "../../types/device";
import type { FenceSummary } from "../../types/fence";

type AdminTab = "devices" | "fences" | "commands" | "users" | "mqtt";

function emptyFenceDraft() {
  return {
    name: "",
    polygonText: "39.9074,116.3975\n39.9090,116.4010\n39.9050,116.4040",
  };
}

export function AdminPage() {
  const queryClient = useQueryClient();
  const { user } = authStore();
  const devicesQuery = useDeviceList();
  const usersQuery = useUsers();
  const mqttStatusQuery = useMQTTStatus();
  const mqttMessagesQuery = useMQTTMessages(20);
  const sharesQuery = useShares();

  const [tab, setTab] = useState<AdminTab>("devices");
  const [selectedDeviceSN, setSelectedDeviceSN] = useState<string | null>(null);
  const [deviceForm, setDeviceForm] = useState({
    device_sn: "",
    imei: "",
    iccid: "",
    name: "",
    status: "1",
    battery: "100",
  });
  const [commandText, setCommandText] = useState('{"cmd":"get_status"}');
  const [commandResult, setCommandResult] = useState<string>("");
  const [fenceDraft, setFenceDraft] = useState(emptyFenceDraft());
  const [editingFence, setEditingFence] = useState<FenceSummary | null>(null);
  const [userForm, setUserForm] = useState({
    username: "",
    password: "",
    role: "user" as "admin" | "user",
  });
  const [userUpdatePassword, setUserUpdatePassword] = useState("");
  const [selectedUserID, setSelectedUserID] = useState<number | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const devices = devicesQuery.data?.devices ?? [];

  useEffect(() => {
    if (!selectedDeviceSN && devices[0]) {
      setSelectedDeviceSN(devices[0].device_sn);
    }
  }, [devices, selectedDeviceSN]);

  const selectedDevice =
    devices.find((device) => device.device_sn === selectedDeviceSN) ?? null;

  const fencesQuery = useFences(selectedDeviceSN);

  const metrics = [
    {
      label: "设备数",
      value: `${devices.length}`,
      tone: "brand" as const,
    },
    {
      label: "用户数",
      value: `${usersQuery.data?.users.length ?? 0}`,
    },
    {
      label: "MQTT",
      value: mqttStatusQuery.data?.connected ? "已连接" : "未连接",
      tone: mqttStatusQuery.data?.connected ? ("brand" as const) : ("warn" as const),
    },
  ];

  const invalidateCoreQueries = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["devices"] }),
      queryClient.invalidateQueries({ queryKey: ["device"] }),
      queryClient.invalidateQueries({ queryKey: ["fences"] }),
      queryClient.invalidateQueries({ queryKey: ["users"] }),
      queryClient.invalidateQueries({ queryKey: ["mqtt-status"] }),
      queryClient.invalidateQueries({ queryKey: ["mqtt-messages"] }),
      queryClient.invalidateQueries({ queryKey: ["shares"] }),
    ]);
  };

  const createDeviceMutation = useMutation({
    mutationFn: createDevice,
    onSuccess: async () => {
      await invalidateCoreQueries();
      setDeviceForm({
        device_sn: "",
        imei: "",
        iccid: "",
        name: "",
        status: "1",
        battery: "100",
      });
    },
  });

  const updateDeviceMutation = useMutation({
    mutationFn: (input: { deviceSN: string; payload: Parameters<typeof updateDevice>[1] }) =>
      updateDevice(input.deviceSN, input.payload),
    onSuccess: invalidateCoreQueries,
  });

  const deleteDeviceMutation = useMutation({
    mutationFn: deleteDevice,
    onSuccess: async () => {
      await invalidateCoreQueries();
      setSelectedDeviceSN(null);
    },
  });

  const sendCommandMutation = useMutation({
    mutationFn: (input: { deviceSN: string; payload: { cmd: string; params?: Record<string, unknown> } }) =>
      sendDeviceCommand(input.deviceSN, input.payload),
  });

  const createFenceMutation = useMutation({
    mutationFn: (input: { deviceSN: string; name: string; polygon: { lat: number; lng: number }[] }) =>
      createFence(input.deviceSN, {
        name: input.name,
        polygon: input.polygon,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["fences", selectedDeviceSN] });
      setFenceDraft(emptyFenceDraft());
      setEditingFence(null);
    },
  });

  const updateFenceMutation = useMutation({
    mutationFn: (input: {
      deviceSN: string;
      fenceID: number;
      name: string;
      polygon: { lat: number; lng: number }[];
    }) =>
      updateFence(input.deviceSN, input.fenceID, {
        name: input.name,
        polygon: input.polygon,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["fences", selectedDeviceSN] });
      setFenceDraft(emptyFenceDraft());
      setEditingFence(null);
    },
  });

  const deleteFenceMutation = useMutation({
    mutationFn: (input: { deviceSN: string; fenceID: number }) =>
      deleteFence(input.deviceSN, input.fenceID),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["fences", selectedDeviceSN] });
    },
  });

  const createUserMutation = useMutation({
    mutationFn: createUser,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["users"] });
      setUserForm({
        username: "",
        password: "",
        role: "user",
      });
    },
  });

  const updateUserMutation = useMutation({
    mutationFn: (input: {
      userID: number;
      payload: {
        password?: string;
        role?: "admin" | "user";
      };
    }) => updateUser(input.userID, input.payload),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["users"] });
      setUserUpdatePassword("");
    },
  });

  const deleteUserMutation = useMutation({
    mutationFn: deleteUser,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["users"] });
      setSelectedUserID(null);
    },
  });

  function readPolygon(text: string) {
    return text
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean)
      .map((line) => {
        const [latRaw, lngRaw] = line.split(",").map((item) => item.trim());
        return {
          lat: Number(latRaw),
          lng: Number(lngRaw),
        };
      });
  }

  async function handleCreateDevice() {
    setErrorMessage(null);
    try {
      await createDeviceMutation.mutateAsync({
        device_sn: deviceForm.device_sn.trim(),
        imei: deviceForm.imei.trim() || undefined,
        iccid: deviceForm.iccid.trim() || undefined,
        name: deviceForm.name.trim() || undefined,
        status: Number(deviceForm.status),
        battery: Number(deviceForm.battery),
      });
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "创建设备失败");
    }
  }

  async function handleUpdateSelectedDevice(device: DeviceSummary) {
    setErrorMessage(null);
    try {
      await updateDeviceMutation.mutateAsync({
        deviceSN: device.device_sn,
        payload: {
          name: device.name,
          imei: device.imei || null,
          iccid: device.iccid || null,
          status: device.status,
          battery: device.battery,
        },
      });
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "更新设备失败");
    }
  }

  async function handleSendCommand() {
    if (!selectedDeviceSN) {
      return;
    }

    setErrorMessage(null);
    try {
      const parsed = JSON.parse(commandText) as { cmd: string } & Record<string, unknown>;
      const { cmd, ...rest } = parsed;
      const result = await sendCommandMutation.mutateAsync({
        deviceSN: selectedDeviceSN,
        payload: {
          cmd,
          params: rest,
        },
      });
      setCommandResult(JSON.stringify(result, null, 2));
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "下发命令失败");
    }
  }

  async function handleSubmitFence() {
    if (!selectedDeviceSN) {
      return;
    }

    setErrorMessage(null);
    try {
      const polygon = readPolygon(fenceDraft.polygonText);
      if (editingFence) {
        await updateFenceMutation.mutateAsync({
          deviceSN: selectedDeviceSN,
          fenceID: editingFence.id,
          name: fenceDraft.name.trim(),
          polygon,
        });
      } else {
        await createFenceMutation.mutateAsync({
          deviceSN: selectedDeviceSN,
          name: fenceDraft.name.trim(),
          polygon,
        });
      }
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "保存围栏失败");
    }
  }

  async function handleCreateUser() {
    setErrorMessage(null);
    try {
      await createUserMutation.mutateAsync(userForm);
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "创建用户失败");
    }
  }

  async function handleUpdateUser(role: "admin" | "user") {
    if (!selectedUserID) {
      return;
    }

    setErrorMessage(null);
    try {
      await updateUserMutation.mutateAsync({
        userID: selectedUserID,
        payload: {
          role,
          password: userUpdatePassword.trim() || undefined,
        },
      });
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "更新用户失败");
    }
  }

  const selectedUser =
    usersQuery.data?.users.find((item) => item.id === selectedUserID) ?? null;

  const recentShares = sharesQuery.data?.shares.slice(0, 5) ?? [];

  return (
    <main className="min-h-screen p-4 md:p-5">
      <div className="grid min-h-[calc(100vh-2rem)] grid-rows-[auto_1fr] gap-4">
        <AppHeader
          mode="live"
          title="管理台"
          description="补齐设备、围栏、远程命令、用户和 MQTT 运维入口。这里是后台管理面，不是地图值守面。"
          metrics={metrics}
          active="admin"
        />

        <section className="grid min-h-0 gap-4 xl:grid-cols-[260px_minmax(0,1fr)_380px]">
          <aside className="glass-panel flex min-h-0 flex-col rounded-[28px] p-4">
            <div className="text-sm font-semibold text-[#10212b]">管理分区</div>
            <div className="mt-4 space-y-2">
              {[
                ["devices", "设备管理"],
                ["fences", "围栏管理"],
                ["commands", "远程命令"],
                ["users", "用户管理"],
                ["mqtt", "MQTT 监控"],
              ].map(([value, label]) => (
                <button
                  key={value}
                  type="button"
                  onClick={() => setTab(value as AdminTab)}
                  className={`w-full rounded-2xl px-4 py-3 text-left text-sm font-semibold transition ${
                    tab === value
                      ? "bg-[#10212b] text-white"
                      : "border border-black/8 bg-white/72 text-[#10212b] hover:bg-white"
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>

            <div className="mt-6 min-h-0 flex-1 overflow-y-auto pr-1">
              <div className="rounded-[24px] border border-black/6 bg-white/64 p-4">
                <div className="text-xs uppercase tracking-[0.18em] text-[#7a8a94]">
                  当前账号
                </div>
                <div className="mt-2 text-sm font-semibold text-[#10212b]">
                  {user?.username || "--"}
                </div>
                <div className="mt-1 text-xs text-[#546570]">角色 {user?.role || "--"}</div>
              </div>

              <div className="mt-4 rounded-[24px] border border-black/6 bg-white/64 p-4">
                <div className="text-xs uppercase tracking-[0.18em] text-[#7a8a94]">
                  最近分享
                </div>
                <div className="mt-3 space-y-2">
                  {recentShares.length === 0 ? (
                    <div className="text-sm text-[#546570]">暂无分享记录</div>
                  ) : (
                    recentShares.map((share) => (
                      <div
                        key={share.id}
                        className="rounded-2xl bg-white/72 px-3 py-3 text-sm text-[#10212b]"
                      >
                        <div className="font-semibold">
                          {share.device_name || share.device_sn}
                        </div>
                        <div className="mt-1 text-xs text-[#546570]">
                          {share.share_mode === "today_track" ? "实时+轨迹" : "仅实时"} ·{" "}
                          {share.status}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>
          </aside>

          <section className="glass-panel min-h-0 rounded-[28px] p-5">
            {errorMessage ? (
              <div className="mb-4 rounded-[24px] border border-[#d94747]/20 bg-[#d94747]/8 px-4 py-3 text-sm text-[#9d2323]">
                {errorMessage}
              </div>
            ) : null}

            {tab === "devices" ? (
              <div className="grid min-h-0 gap-4 xl:grid-cols-[minmax(0,1fr)_340px]">
                <section className="min-h-0">
                  <div className="text-lg font-semibold text-[#10212b]">设备管理</div>
                  <div className="mt-4 min-h-0 space-y-3 overflow-y-auto pr-1 xl:max-h-[720px]">
                    {devices.map((device) => {
                      const active = device.device_sn === selectedDeviceSN;
                      return (
                        <button
                          key={device.device_sn}
                          type="button"
                          onClick={() => setSelectedDeviceSN(device.device_sn)}
                          className={`w-full rounded-[24px] border px-4 py-4 text-left transition ${
                            active
                              ? "border-[#1f88c9]/30 bg-[#1f88c9]/8"
                              : "border-black/6 bg-white/66 hover:bg-white/84"
                          }`}
                        >
                          <div className="flex items-start justify-between gap-3">
                            <div>
                              <div className="text-sm font-semibold text-[#10212b]">
                                {device.name || device.device_sn}
                              </div>
                              <div className="mt-1 text-xs text-[#546570]">
                                {device.device_sn}
                              </div>
                            </div>
                            <span className="rounded-full bg-[#10212b]/6 px-2.5 py-1 text-xs font-semibold text-[#3e505a]">
                              {getGPSStateLabel(device.gps_state)}
                            </span>
                          </div>
                          <div className="mt-3 text-xs text-[#6a7a84]">
                            电量 {device.battery}% · 最近在线{" "}
                            {formatRelativeTime(device.last_online)}
                          </div>
                        </button>
                      );
                    })}
                  </div>
                </section>

                <section className="space-y-4">
                  <div className="rounded-[24px] border border-black/6 bg-white/64 p-4">
                    <div className="text-sm font-semibold text-[#10212b]">新建设备</div>
                    <div className="mt-3 grid gap-3">
                      <input
                        value={deviceForm.device_sn}
                        onChange={(event) =>
                          setDeviceForm((current) => ({
                            ...current,
                            device_sn: event.target.value,
                          }))
                        }
                        placeholder="device_sn"
                        className="rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                      />
                      <input
                        value={deviceForm.name}
                        onChange={(event) =>
                          setDeviceForm((current) => ({ ...current, name: event.target.value }))
                        }
                        placeholder="名称"
                        className="rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                      />
                      <input
                        value={deviceForm.imei}
                        onChange={(event) =>
                          setDeviceForm((current) => ({ ...current, imei: event.target.value }))
                        }
                        placeholder="IMEI"
                        className="rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                      />
                      <input
                        value={deviceForm.iccid}
                        onChange={(event) =>
                          setDeviceForm((current) => ({ ...current, iccid: event.target.value }))
                        }
                        placeholder="ICCID"
                        className="rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                      />
                      <div className="grid gap-3 sm:grid-cols-2">
                        <input
                          value={deviceForm.status}
                          onChange={(event) =>
                            setDeviceForm((current) => ({ ...current, status: event.target.value }))
                          }
                          placeholder="status"
                          className="rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                        />
                        <input
                          value={deviceForm.battery}
                          onChange={(event) =>
                            setDeviceForm((current) => ({ ...current, battery: event.target.value }))
                          }
                          placeholder="battery"
                          className="rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                        />
                      </div>
                      <button
                        type="button"
                        onClick={() => void handleCreateDevice()}
                        disabled={createDeviceMutation.isPending}
                        className="rounded-2xl bg-[#10212b] px-4 py-3 text-sm font-semibold text-white"
                      >
                        创建设备
                      </button>
                    </div>
                  </div>

                  {selectedDevice ? (
                    <div className="rounded-[24px] border border-black/6 bg-white/64 p-4">
                      <div className="text-sm font-semibold text-[#10212b]">选中设备</div>
                      <div className="mt-3 space-y-3 text-sm text-[#546570]">
                        <InfoRow label="名称" value={selectedDevice.name || "--"} />
                        <InfoRow label="SN" value={selectedDevice.device_sn} />
                        <InfoRow label="IMEI" value={selectedDevice.imei || "--"} />
                        <InfoRow label="ICCID" value={selectedDevice.iccid || "--"} />
                        <InfoRow label="电量" value={`${selectedDevice.battery}%`} />
                        <InfoRow label="GPS" value={getGPSStateLabel(selectedDevice.gps_state)} />
                        <InfoRow
                          label="最近在线"
                          value={formatDateTime(selectedDevice.last_online)}
                        />
                      </div>
                      <div className="mt-4 flex flex-wrap gap-3">
                        <button
                          type="button"
                          onClick={() => void handleUpdateSelectedDevice(selectedDevice)}
                          disabled={updateDeviceMutation.isPending}
                          className="rounded-2xl bg-[#10212b] px-4 py-2.5 text-sm font-semibold text-white"
                        >
                          刷新写回
                        </button>
                        <button
                          type="button"
                          onClick={() => deleteDeviceMutation.mutate(selectedDevice.device_sn)}
                          disabled={deleteDeviceMutation.isPending}
                          className="rounded-2xl border border-[#d94747]/20 bg-[#d94747]/8 px-4 py-2.5 text-sm font-semibold text-[#9d2323]"
                        >
                          删除设备
                        </button>
                      </div>
                    </div>
                  ) : null}
                </section>
              </div>
            ) : null}

            {tab === "fences" ? (
              <div className="grid min-h-0 gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
                <section>
                  <div className="text-lg font-semibold text-[#10212b]">围栏设备选择</div>
                  <div className="mt-4 space-y-3">
                    {devices.map((device) => (
                      <button
                        key={device.device_sn}
                        type="button"
                        onClick={() => setSelectedDeviceSN(device.device_sn)}
                        className={`w-full rounded-[24px] border px-4 py-4 text-left transition ${
                          selectedDeviceSN === device.device_sn
                            ? "border-[#1f88c9]/30 bg-[#1f88c9]/8"
                            : "border-black/6 bg-white/66"
                        }`}
                      >
                        <div className="text-sm font-semibold text-[#10212b]">
                          {device.name || device.device_sn}
                        </div>
                        <div className="mt-1 text-xs text-[#546570]">{device.device_sn}</div>
                      </button>
                    ))}
                  </div>
                </section>

                <section className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
                  <div className="rounded-[24px] border border-black/6 bg-white/64 p-4">
                    <div className="text-sm font-semibold text-[#10212b]">当前围栏</div>
                    <div className="mt-4 space-y-3">
                      {(fencesQuery.data?.fences ?? []).map((fence) => (
                        <div
                          key={fence.id}
                          className="rounded-[22px] border border-black/6 bg-white/72 p-4"
                        >
                          <div className="flex items-start justify-between gap-3">
                            <div>
                              <div className="text-sm font-semibold text-[#10212b]">
                                {fence.name}
                              </div>
                              <div className="mt-1 text-xs text-[#546570]">
                                点数 {fence.polygon.length} · 最近检查{" "}
                                {formatRelativeTime(fence.last_checked_at)}
                              </div>
                            </div>
                            <div className="flex gap-2">
                              <button
                                type="button"
                                onClick={() => {
                                  setEditingFence(fence);
                                  setFenceDraft({
                                    name: fence.name,
                                    polygonText: fence.polygon
                                      .map((point) => `${point.lat},${point.lng}`)
                                      .join("\n"),
                                  });
                                }}
                                className="rounded-full border border-black/8 bg-white px-3 py-1.5 text-xs font-semibold text-[#10212b]"
                              >
                                编辑
                              </button>
                              {selectedDeviceSN ? (
                                <button
                                  type="button"
                                  onClick={() =>
                                    deleteFenceMutation.mutate({
                                      deviceSN: selectedDeviceSN,
                                      fenceID: fence.id,
                                    })
                                  }
                                  className="rounded-full border border-[#d94747]/20 bg-[#d94747]/8 px-3 py-1.5 text-xs font-semibold text-[#9d2323]"
                                >
                                  删除
                                </button>
                              ) : null}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>

                  <div className="rounded-[24px] border border-black/6 bg-white/64 p-4">
                    <div className="text-sm font-semibold text-[#10212b]">
                      {editingFence ? "编辑围栏" : "新增围栏"}
                    </div>
                    <div className="mt-3 grid gap-3">
                      <input
                        value={fenceDraft.name}
                        onChange={(event) =>
                          setFenceDraft((current) => ({ ...current, name: event.target.value }))
                        }
                        placeholder="围栏名称"
                        className="rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                      />
                      <textarea
                        value={fenceDraft.polygonText}
                        onChange={(event) =>
                          setFenceDraft((current) => ({
                            ...current,
                            polygonText: event.target.value,
                          }))
                        }
                        rows={8}
                        placeholder="每行一个点，格式 lat,lng"
                        className="rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                      />
                      <button
                        type="button"
                        onClick={() => void handleSubmitFence()}
                        className="rounded-2xl bg-[#10212b] px-4 py-3 text-sm font-semibold text-white"
                      >
                        保存围栏
                      </button>
                      {editingFence ? (
                        <button
                          type="button"
                          onClick={() => {
                            setEditingFence(null);
                            setFenceDraft(emptyFenceDraft());
                          }}
                          className="rounded-2xl border border-black/8 bg-white/72 px-4 py-3 text-sm font-semibold text-[#10212b]"
                        >
                          取消编辑
                        </button>
                      ) : null}
                    </div>
                  </div>
                </section>
              </div>
            ) : null}

            {tab === "commands" ? (
              <div className="grid min-h-0 gap-4 xl:grid-cols-[300px_minmax(0,1fr)]">
                <section>
                  <div className="text-lg font-semibold text-[#10212b]">目标设备</div>
                  <div className="mt-4 space-y-3">
                    {devices.map((device) => (
                      <button
                        key={device.device_sn}
                        type="button"
                        onClick={() => setSelectedDeviceSN(device.device_sn)}
                        className={`w-full rounded-[24px] border px-4 py-4 text-left transition ${
                          selectedDeviceSN === device.device_sn
                            ? "border-[#1f88c9]/30 bg-[#1f88c9]/8"
                            : "border-black/6 bg-white/66"
                        }`}
                      >
                        <div className="text-sm font-semibold text-[#10212b]">
                          {device.name || device.device_sn}
                        </div>
                        <div className="mt-1 text-xs text-[#546570]">{device.device_sn}</div>
                      </button>
                    ))}
                  </div>
                </section>

                <section className="rounded-[24px] border border-black/6 bg-white/64 p-4">
                  <div className="text-lg font-semibold text-[#10212b]">远程命令</div>
                  <div className="mt-3 text-sm leading-7 text-[#546570]">
                    直接下发到 `locator/&lt;device_id&gt;/cmd`。支持 `get_status`、`get_config`、`set_config`。
                  </div>
                  <textarea
                    value={commandText}
                    onChange={(event) => setCommandText(event.target.value)}
                    rows={12}
                    className="mt-4 w-full rounded-2xl border border-black/8 bg-white px-4 py-3 font-mono text-sm outline-none"
                  />
                  <div className="mt-4 flex flex-wrap gap-3">
                    <button
                      type="button"
                      onClick={() => void handleSendCommand()}
                      disabled={!selectedDeviceSN || sendCommandMutation.isPending}
                      className="rounded-2xl bg-[#10212b] px-4 py-2.5 text-sm font-semibold text-white"
                    >
                      下发命令
                    </button>
                    <button
                      type="button"
                      onClick={() => setCommandText('{"cmd":"get_status"}')}
                      className="rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b]"
                    >
                      填入 get_status
                    </button>
                    <button
                      type="button"
                      onClick={() => setCommandText('{"cmd":"get_config"}')}
                      className="rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b]"
                    >
                      填入 get_config
                    </button>
                  </div>

                  <pre className="mt-4 overflow-x-auto rounded-[24px] bg-[#10212b] px-4 py-4 text-xs leading-6 text-white/88">
                    {commandResult || "// 命令响应会显示在这里"}
                  </pre>
                </section>
              </div>
            ) : null}

            {tab === "users" ? (
              <div className="grid min-h-0 gap-4 xl:grid-cols-[minmax(0,1fr)_340px]">
                <section className="min-h-0">
                  <div className="text-lg font-semibold text-[#10212b]">用户管理</div>
                  <div className="mt-4 min-h-0 space-y-3 overflow-y-auto pr-1 xl:max-h-[720px]">
                    {(usersQuery.data?.users ?? []).map((item) => (
                      <button
                        key={item.id}
                        type="button"
                        onClick={() => setSelectedUserID(item.id)}
                        className={`w-full rounded-[24px] border px-4 py-4 text-left transition ${
                          selectedUserID === item.id
                            ? "border-[#1f88c9]/30 bg-[#1f88c9]/8"
                            : "border-black/6 bg-white/66 hover:bg-white/84"
                        }`}
                      >
                        <div className="flex items-center justify-between gap-3">
                          <div>
                            <div className="text-sm font-semibold text-[#10212b]">
                              {item.username}
                            </div>
                            <div className="mt-1 text-xs text-[#546570]">ID {item.id}</div>
                          </div>
                          <span className="rounded-full bg-[#10212b]/6 px-2.5 py-1 text-xs font-semibold text-[#3e505a]">
                            {item.role}
                          </span>
                        </div>
                      </button>
                    ))}
                  </div>
                </section>

                <section className="space-y-4">
                  <div className="rounded-[24px] border border-black/6 bg-white/64 p-4">
                    <div className="text-sm font-semibold text-[#10212b]">新增用户</div>
                    <div className="mt-3 grid gap-3">
                      <input
                        value={userForm.username}
                        onChange={(event) =>
                          setUserForm((current) => ({
                            ...current,
                            username: event.target.value,
                          }))
                        }
                        placeholder="用户名"
                        className="rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                      />
                      <input
                        type="password"
                        value={userForm.password}
                        onChange={(event) =>
                          setUserForm((current) => ({
                            ...current,
                            password: event.target.value,
                          }))
                        }
                        placeholder="密码"
                        className="rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                      />
                      <select
                        value={userForm.role}
                        onChange={(event) =>
                          setUserForm((current) => ({
                            ...current,
                            role: event.target.value as "admin" | "user",
                          }))
                        }
                        className="rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                      >
                        <option value="user">user</option>
                        <option value="admin">admin</option>
                      </select>
                      <button
                        type="button"
                        onClick={() => void handleCreateUser()}
                        className="rounded-2xl bg-[#10212b] px-4 py-3 text-sm font-semibold text-white"
                      >
                        创建用户
                      </button>
                    </div>
                  </div>

                  {selectedUser ? (
                    <div className="rounded-[24px] border border-black/6 bg-white/64 p-4">
                      <div className="text-sm font-semibold text-[#10212b]">编辑用户</div>
                      <div className="mt-3 space-y-3">
                        <InfoRow label="用户名" value={selectedUser.username} />
                        <InfoRow label="角色" value={selectedUser.role} />
                        <input
                          type="password"
                          value={userUpdatePassword}
                          onChange={(event) => setUserUpdatePassword(event.target.value)}
                          placeholder="新密码，可留空"
                          className="w-full rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none"
                        />
                        <div className="flex flex-wrap gap-3">
                          <button
                            type="button"
                            onClick={() => void handleUpdateUser("user")}
                            className="rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b]"
                          >
                            设为 user
                          </button>
                          <button
                            type="button"
                            onClick={() => void handleUpdateUser("admin")}
                            className="rounded-2xl bg-[#10212b] px-4 py-2.5 text-sm font-semibold text-white"
                          >
                            设为 admin
                          </button>
                          <button
                            type="button"
                            onClick={() => deleteUserMutation.mutate(selectedUser.id)}
                            className="rounded-2xl border border-[#d94747]/20 bg-[#d94747]/8 px-4 py-2.5 text-sm font-semibold text-[#9d2323]"
                          >
                            删除用户
                          </button>
                        </div>
                      </div>
                    </div>
                  ) : null}
                </section>
              </div>
            ) : null}

            {tab === "mqtt" ? (
              <div className="grid min-h-0 gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
                <section className="rounded-[24px] border border-black/6 bg-white/64 p-4">
                  <div className="text-lg font-semibold text-[#10212b]">MQTT 状态</div>
                  <div className="mt-4 space-y-3">
                    <InfoRow
                      label="启用状态"
                      value={mqttStatusQuery.data?.enabled ? "已启用" : "未启用"}
                    />
                    <InfoRow
                      label="连接状态"
                      value={mqttStatusQuery.data?.connected ? "已连接" : "未连接"}
                    />
                    <div className="rounded-[22px] border border-black/6 bg-white/72 p-4">
                      <div className="text-xs uppercase tracking-[0.18em] text-[#7a8a94]">
                        订阅主题
                      </div>
                      <div className="mt-3 space-y-2 text-sm text-[#10212b]">
                        {(mqttStatusQuery.data?.topics ?? []).map((topic) => (
                          <div key={topic} className="rounded-full bg-[#10212b]/6 px-3 py-2">
                            {topic}
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                </section>

                <section className="rounded-[24px] border border-black/6 bg-white/64 p-4">
                  <div className="text-lg font-semibold text-[#10212b]">最近消息</div>
                  <div className="mt-4 min-h-0 space-y-3 overflow-y-auto pr-1 xl:max-h-[720px]">
                    {(mqttMessagesQuery.data?.messages ?? []).map((message) => (
                      <div
                        key={`${message.topic}-${message.received_at}`}
                        className="rounded-[22px] border border-black/6 bg-white/72 p-4"
                      >
                        <div className="text-sm font-semibold text-[#10212b]">{message.topic}</div>
                        <div className="mt-1 text-xs text-[#546570]">
                          {formatDateTime(message.received_at)} · QoS {message.qos}
                        </div>
                        <pre className="mt-3 overflow-x-auto whitespace-pre-wrap rounded-2xl bg-[#10212b] px-3 py-3 text-xs leading-6 text-white/88">
                          {message.payload}
                        </pre>
                      </div>
                    ))}
                  </div>
                </section>
              </div>
            ) : null}
          </section>

          <aside className="glass-panel flex min-h-0 flex-col rounded-[28px] p-5">
            <div className="text-sm font-semibold text-[#10212b]">当前设备上下文</div>
            {selectedDevice ? (
              <div className="mt-4 space-y-3">
                <InfoRow label="名称" value={selectedDevice.name || "--"} />
                <InfoRow label="SN" value={selectedDevice.device_sn} />
                <InfoRow label="GPS" value={getGPSStateLabel(selectedDevice.gps_state)} />
                <InfoRow label="状态" value={`${selectedDevice.status}`} />
                <InfoRow label="电量" value={`${selectedDevice.battery}%`} />
                <InfoRow label="最近在线" value={formatRelativeTime(selectedDevice.last_online)} />
                <InfoRow label="最后可信定位" value={formatRelativeTime(selectedDevice.last_fix_at)} />
                <InfoRow
                  label="状态更新时间"
                  value={formatDateTime(selectedDevice.status_updated_at)}
                />
              </div>
            ) : (
              <div className="mt-4 text-sm leading-7 text-[#546570]">
                选择一个设备后，这里会同步展示当前设备上下文，便于围栏管理和命令下发。
              </div>
            )}
          </aside>
        </section>
      </div>
    </main>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[22px] border border-black/6 bg-white/64 p-4">
      <div className="text-xs uppercase tracking-[0.18em] text-[#7a8a94]">{label}</div>
      <div className="mt-2 break-all text-sm font-semibold text-[#10212b]">{value}</div>
    </div>
  );
}
