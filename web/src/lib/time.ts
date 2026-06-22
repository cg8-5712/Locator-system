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
    return `${diffSeconds}秒前`;
  }
  if (diffSeconds < 3600) {
    return `${Math.floor(diffSeconds / 60)}分钟前`;
  }
  if (diffSeconds < 86400) {
    return `${Math.floor(diffSeconds / 3600)}小时前`;
  }
  return `${Math.floor(diffSeconds / 86400)}天前`;
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
