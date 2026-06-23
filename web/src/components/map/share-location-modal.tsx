import { useEffect, useMemo, useState } from "react";
import type { DeviceSummary } from "../../types/device";
import type { AppMode } from "../../features/map-view/mode";
import { formatDateTime, formatDurationSeconds } from "../../lib/time";

type ShareMode = "live_only" | "today_track";
type ExpiryPreset = "30m" | "1h" | "24h" | "custom";

interface GeneratedShare {
  url: string;
  password: string | null;
  expiresAt: string;
  remainingSeconds: number;
  remainingVisits: string;
}

const text = {
  title: "\u521b\u5efa\u4f4d\u7f6e\u5206\u4eab\u94fe\u63a5",
  close: "\u5173\u95ed",
  shareMode: "\u5206\u4eab\u6a21\u5f0f",
  liveOnly: "\u4ec5\u663e\u793a\u5f53\u524d\u5b9e\u65f6\u4f4d\u7f6e",
  todayTrack: "\u5141\u8bb8\u67e5\u770b\u4eca\u65e5\u8f68\u8ff9",
  password: "\u8bbf\u95ee\u5bc6\u7801",
  enabled: "\u542f\u7528",
  expiry: "\u6709\u6548\u671f",
  thirtyMinutes: "30 \u5206\u949f",
  oneHour: "1 \u5c0f\u65f6",
  twentyFourHours: "24 \u5c0f\u65f6",
  customTime: "\u81ea\u5b9a\u4e49\u65f6\u95f4",
  visits: "\u8bbf\u95ee\u6b21\u6570\u9650\u5236",
  limitVisits: "\u9650\u5236\u6b21\u6570",
  generate: "\u751f\u6210\u5206\u4eab\u94fe\u63a5",
  cancel: "\u53d6\u6d88",
  result: "\u751f\u6210\u7ed3\u679c",
  copyAll: "\u590d\u5236\u5168\u90e8\u4fe1\u606f",
  unlimited: "\u4e0d\u9650\u6b21\u6570",
  noPassword: "\u672a\u542f\u7528",
  link: "\u94fe\u63a5",
  passwordLabel: "\u5bc6\u7801",
  expiresAt: "\u8fc7\u671f\u65f6\u95f4",
  remainingTime: "\u5269\u4f59\u65f6\u95f4",
  remainingVisits: "\u5269\u4f59\u989d\u5ea6",
  success:
    "\u5df2\u751f\u6210\u5b89\u5168\u5206\u4eab\u94fe\u63a5\uff0c\u53ef\u7528\u4e8e\u524d\u7aef\u9a8c\u6536\u4e0e\u4ea4\u4e92\u8bc4\u5ba1\u3002",
  empty:
    "\u5148\u5b8c\u6210\u5206\u4eab\u6a21\u5f0f\u3001\u5bc6\u7801\u3001\u6709\u6548\u671f\u548c\u6b21\u6570\u9650\u5236\u914d\u7f6e\uff0c\u7136\u540e\u751f\u6210\u94fe\u63a5\u3002",
  demoDesc:
    "\u5f53\u524d\u4e3a\u6b7b\u6570\u636e\u9a8c\u8bc1\u6a21\u5f0f\uff0c\u751f\u6210\u7ed3\u679c\u53ef\u76f4\u63a5\u7528\u4e8e UI \u8054\u8c03\u548c\u6d41\u7a0b\u9a8c\u6536\u3002",
  liveDesc:
    "\u5f53\u524d\u4e3a\u771f\u5b9e\u540e\u7aef\u6a21\u5f0f\uff0c\u4f46\u5206\u4eab\u63a5\u53e3\u5c1a\u672a\u843d\u5730\uff0c\u6b64\u5f39\u7a97\u7528\u4e8e\u5148\u9a8c\u8bc1\u524d\u7aef\u4ea4\u4e92\u4e0e\u5b57\u6bb5\u8bbe\u8ba1\u3002",
  copyLink: "\u4f4d\u7f6e\u5206\u4eab\u94fe\u63a5",
  copyPassword: "\u8bbf\u95ee\u5bc6\u7801",
  copyExpire: "\u6709\u6548\u671f\u81f3",
  copyVisits: "\u5269\u4f59\u8bbf\u95ee\u6b21\u6570",
};

export function ShareLocationModal({
  open,
  mode,
  device,
  onClose,
}: {
  open: boolean;
  mode: AppMode;
  device: DeviceSummary | null;
  onClose: () => void;
}) {
  const [shareMode, setShareMode] = useState<ShareMode>("live_only");
  const [requirePassword, setRequirePassword] = useState(true);
  const [password, setPassword] = useState("");
  const [expiryPreset, setExpiryPreset] = useState<ExpiryPreset>("1h");
  const [customExpiry, setCustomExpiry] = useState("");
  const [limitVisits, setLimitVisits] = useState(true);
  const [maxVisits, setMaxVisits] = useState(5);
  const [generated, setGenerated] = useState<GeneratedShare | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }

    const nextHour = new Date(Date.now() + 60 * 60 * 1000);
    const localDateTime = `${nextHour.getFullYear()}-${String(nextHour.getMonth() + 1).padStart(2, "0")}-${String(nextHour.getDate()).padStart(2, "0")}T${String(nextHour.getHours()).padStart(2, "0")}:${String(nextHour.getMinutes()).padStart(2, "0")}`;

    setShareMode("live_only");
    setRequirePassword(true);
    setPassword(createRandomDigits());
    setExpiryPreset("1h");
    setCustomExpiry(localDateTime);
    setLimitVisits(true);
    setMaxVisits(5);
    setGenerated(null);
  }, [open, device?.device_sn]);

  const expirationPreview = useMemo(() => {
    if (expiryPreset === "30m") {
      return new Date(Date.now() + 30 * 60 * 1000);
    }
    if (expiryPreset === "1h") {
      return new Date(Date.now() + 60 * 60 * 1000);
    }
    if (expiryPreset === "24h") {
      return new Date(Date.now() + 24 * 60 * 60 * 1000);
    }

    return customExpiry ? new Date(customExpiry) : null;
  }, [customExpiry, expiryPreset]);

  if (!open || !device) {
    return null;
  }

  function handleCreate() {
    const expirationDate = expirationPreview;
    if (!expirationDate || Number.isNaN(expirationDate.getTime())) {
      return;
    }

    const shareCode = createRandomCode();
    const previewUrl =
      mode === "demo"
        ? `${window.location.origin}/demo/share/${device.device_sn}?code=${shareCode}`
        : `https://maps.locatorhub.com/s/${shareCode}`;

    const remainingSeconds = Math.max(
      0,
      Math.floor((expirationDate.getTime() - Date.now()) / 1000)
    );

    setGenerated({
      url: previewUrl,
      password: requirePassword ? password : null,
      expiresAt: expirationDate.toISOString(),
      remainingSeconds,
      remainingVisits: limitVisits ? `${maxVisits}/${maxVisits}` : text.unlimited,
    });
  }

  async function handleCopy() {
    if (!generated) {
      return;
    }

    const content = [
      `${text.copyLink}\uff1a${generated.url}`,
      generated.password
        ? `${text.copyPassword}\uff1a${generated.password}`
        : `${text.copyPassword}\uff1a${text.noPassword}`,
      `${text.copyExpire}\uff1a${formatDateTime(generated.expiresAt)}`,
      `${text.copyVisits}\uff1a${generated.remainingVisits}`,
    ].join("\n");

    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(content);
    }
  }

  return (
    <div className="fixed inset-0 z-[1900] flex items-center justify-center bg-[#10212b]/42 px-4 py-8 backdrop-blur-sm">
      <div className="glass-panel w-full max-w-3xl rounded-[32px] p-6">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.22em] text-[#1f88c9]">
              Share Preview
            </p>
            <h2 className="mt-2 text-2xl font-semibold text-[#10212b]">
              {text.title} · {device.name || device.device_sn}
            </h2>
            <p className="mt-2 text-sm leading-7 text-[#546570]">
              {mode === "demo" ? text.demoDesc : text.liveDesc}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-2xl border border-black/8 bg-white/72 px-4 py-2 text-sm font-semibold text-[#10212b] transition hover:bg-white"
          >
            {text.close}
          </button>
        </div>

        <div className="mt-6 grid gap-6 lg:grid-cols-[1.05fr_0.95fr]">
          <section className="space-y-5 rounded-[28px] border border-black/6 bg-white/58 p-5">
            <div>
              <div className="text-sm font-semibold text-[#10212b]">{text.shareMode}</div>
              <div className="mt-3 grid gap-2">
                <RadioCard
                  title={text.liveOnly}
                  checked={shareMode === "live_only"}
                  onChange={() => setShareMode("live_only")}
                />
                <RadioCard
                  title={text.todayTrack}
                  checked={shareMode === "today_track"}
                  onChange={() => setShareMode("today_track")}
                />
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between gap-3">
                <div className="text-sm font-semibold text-[#10212b]">{text.password}</div>
                <label className="inline-flex items-center gap-2 text-sm text-[#546570]">
                  <input
                    type="checkbox"
                    checked={requirePassword}
                    onChange={(event) => setRequirePassword(event.target.checked)}
                  />
                  {text.enabled}
                </label>
              </div>
              <input
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                disabled={!requirePassword}
                className="mt-3 w-full rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none transition focus:border-[#1f88c9] focus:ring-4 focus:ring-[#1f88c9]/10 disabled:cursor-not-allowed disabled:bg-[#f4f6f8]"
              />
            </div>

            <div>
              <div className="text-sm font-semibold text-[#10212b]">{text.expiry}</div>
              <div className="mt-3 grid gap-2 sm:grid-cols-2">
                <RadioCard
                  title={text.thirtyMinutes}
                  checked={expiryPreset === "30m"}
                  onChange={() => setExpiryPreset("30m")}
                />
                <RadioCard
                  title={text.oneHour}
                  checked={expiryPreset === "1h"}
                  onChange={() => setExpiryPreset("1h")}
                />
                <RadioCard
                  title={text.twentyFourHours}
                  checked={expiryPreset === "24h"}
                  onChange={() => setExpiryPreset("24h")}
                />
                <RadioCard
                  title={text.customTime}
                  checked={expiryPreset === "custom"}
                  onChange={() => setExpiryPreset("custom")}
                />
              </div>
              {expiryPreset === "custom" ? (
                <input
                  type="datetime-local"
                  value={customExpiry}
                  onChange={(event) => setCustomExpiry(event.target.value)}
                  className="mt-3 w-full rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none transition focus:border-[#1f88c9] focus:ring-4 focus:ring-[#1f88c9]/10"
                />
              ) : null}
            </div>

            <div>
              <div className="flex items-center justify-between gap-3">
                <div className="text-sm font-semibold text-[#10212b]">{text.visits}</div>
                <label className="inline-flex items-center gap-2 text-sm text-[#546570]">
                  <input
                    type="checkbox"
                    checked={limitVisits}
                    onChange={(event) => setLimitVisits(event.target.checked)}
                  />
                  {text.limitVisits}
                </label>
              </div>
              <input
                type="number"
                min={1}
                max={99}
                value={maxVisits}
                onChange={(event) => setMaxVisits(Number(event.target.value) || 1)}
                disabled={!limitVisits}
                className="mt-3 w-full rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none transition focus:border-[#1f88c9] focus:ring-4 focus:ring-[#1f88c9]/10 disabled:cursor-not-allowed disabled:bg-[#f4f6f8]"
              />
            </div>

            <div className="flex gap-3">
              <button
                type="button"
                onClick={handleCreate}
                className="rounded-2xl bg-[#10212b] px-5 py-3 text-sm font-semibold text-white transition hover:bg-[#163445]"
              >
                {text.generate}
              </button>
              <button
                type="button"
                onClick={onClose}
                className="rounded-2xl border border-black/8 bg-white/72 px-5 py-3 text-sm font-semibold text-[#10212b] transition hover:bg-white"
              >
                {text.cancel}
              </button>
            </div>
          </section>

          <section className="rounded-[28px] border border-[#1f88c9]/14 bg-[linear-gradient(180deg,rgba(31,136,201,0.08),rgba(255,255,255,0.7))] p-5">
            <div className="text-sm font-semibold text-[#10212b]">{text.result}</div>
            {generated ? (
              <div className="mt-4 space-y-4">
                <div className="rounded-[24px] border border-[#2f9e68]/16 bg-[#2f9e68]/8 p-4 text-sm text-[#20724c]">
                  {text.success}
                </div>
                <InfoRow label={text.link} value={generated.url} />
                <InfoRow
                  label={text.passwordLabel}
                  value={generated.password ?? text.noPassword}
                />
                <InfoRow label={text.expiresAt} value={formatDateTime(generated.expiresAt)} />
                <InfoRow
                  label={text.remainingTime}
                  value={formatDurationSeconds(generated.remainingSeconds)}
                />
                <InfoRow label={text.remainingVisits} value={generated.remainingVisits} />
                <button
                  type="button"
                  onClick={() => void handleCopy()}
                  className="mt-2 rounded-2xl border border-black/8 bg-white px-4 py-2.5 text-sm font-semibold text-[#10212b] transition hover:bg-[#f8fafb]"
                >
                  {text.copyAll}
                </button>
              </div>
            ) : (
              <div className="mt-4 rounded-[24px] border border-dashed border-black/10 bg-white/48 p-5 text-sm leading-7 text-[#546570]">
                {text.empty}
              </div>
            )}
          </section>
        </div>
      </div>
    </div>
  );
}

function RadioCard({
  title,
  checked,
  onChange,
}: {
  title: string;
  checked: boolean;
  onChange: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onChange}
      className={`flex items-center justify-between rounded-2xl border px-4 py-3 text-left transition ${
        checked
          ? "border-[#1f88c9]/25 bg-[#1f88c9]/8 text-[#10212b]"
          : "border-black/8 bg-white/72 text-[#546570] hover:bg-white"
      }`}
    >
      <span className="text-sm font-medium">{title}</span>
      <span
        className={`h-4 w-4 rounded-full border ${
          checked ? "border-[#1f88c9] bg-[#1f88c9]" : "border-[#b7c4cc] bg-transparent"
        }`}
      />
    </button>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-white/72 px-4 py-3">
      <div className="text-xs uppercase tracking-[0.16em] text-[#7a8a94]">{label}</div>
      <div className="mt-2 break-all text-sm font-medium leading-6 text-[#10212b]">
        {value}
      </div>
    </div>
  );
}

function createRandomDigits() {
  return Array.from({ length: 6 }, () => Math.floor(Math.random() * 10)).join("");
}

function createRandomCode() {
  return Math.random().toString(36).slice(2, 9);
}
