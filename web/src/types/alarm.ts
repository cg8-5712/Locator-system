export interface AlarmSummary {
  device_sn: string;
  type: string;
  content: string;
  created_at: string;
}

export interface AlarmListResult {
  alarms: AlarmSummary[];
  pagination: import("./api").Pagination;
}
