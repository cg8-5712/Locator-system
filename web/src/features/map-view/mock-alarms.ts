import type { AlarmSummary } from "../../types/alarm";

const now = Date.now();

export const mockAlarms: AlarmSummary[] = [
  {
    device_sn: "locator-esp32s3-003",
    type: "sos",
    content: "人员触发 SOS 求救，请立即联系并确认位置。",
    created_at: new Date(now - 90_000).toISOString(),
  },
  {
    device_sn: "locator-esp32s3-003",
    type: "low_battery",
    content: "设备电量低于 15%，建议尽快安排充电。",
    created_at: new Date(now - 12 * 60_000).toISOString(),
  },
  {
    device_sn: "locator-esp32s3-002",
    type: "out_of_fence",
    content: "巡检人员离开核心服务区围栏，请核查现场任务状态。",
    created_at: new Date(now - 23 * 60_000).toISOString(),
  },
  {
    device_sn: "locator-esp32s3-001",
    type: "offline",
    content: "设备曾短暂离线，当前已自动恢复在线。",
    created_at: new Date(now - 55 * 60_000).toISOString(),
  },
];
