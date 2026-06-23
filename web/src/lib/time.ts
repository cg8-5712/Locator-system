export function formatRelativeTime(input?: string): string {
  if (!input) {
    return "暂无";
  }

  const value = new Date(input).getTime();
  if (Number.isNaN(value)) {
    return "暂无";
  }

  const diffSeconds = Math.max(0, Math.floor((Date.now() - value) / 1000));
  if (diffSeconds < 10) {
    return "刚刚";
  }
  if (diffSeconds < 60) {
    return `${diffSeconds} 秒前`;
  }
  if (diffSeconds < 3600) {
    return `${Math.floor(diffSeconds / 60)} 分钟前`;
  }
  if (diffSeconds < 86400) {
    return `${Math.floor(diffSeconds / 3600)} 小时前`;
  }
  return `${Math.floor(diffSeconds / 86400)} 天前`;
}

export function formatDateTime(input?: string): string {
  if (!input) {
    return "--";
  }

  const value = new Date(input);
  if (Number.isNaN(value.getTime())) {
    return "--";
  }

  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(value);
}

export function formatDurationSeconds(totalSeconds?: number) {
  if (!totalSeconds || totalSeconds <= 0) {
    return "0 秒";
  }

  if (totalSeconds < 60) {
    return `${totalSeconds} 秒`;
  }

  const totalMinutes = Math.floor(totalSeconds / 60);
  if (totalMinutes < 60) {
    return `${totalMinutes} 分钟`;
  }

  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (hours < 24) {
    return minutes > 0 ? `${hours} 小时 ${minutes} 分钟` : `${hours} 小时`;
  }

  const days = Math.floor(hours / 24);
  const remainHours = hours % 24;
  return remainHours > 0 ? `${days} 天 ${remainHours} 小时` : `${days} 天`;
}
