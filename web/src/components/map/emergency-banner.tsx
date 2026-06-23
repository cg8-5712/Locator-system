import { useEffect } from "react";
import type { AlarmEvent } from "../../types/realtime";
import { formatRelativeTime } from "../../lib/time";

export function EmergencyBanner({
  alarm,
  deviceName,
  onLocate,
  onHistory,
  onDismiss,
}: {
  alarm: AlarmEvent;
  deviceName: string;
  onLocate: () => void;
  onHistory: () => void;
  onDismiss: () => void;
}) {
  useEffect(() => {
    const AudioContextCtor = window.AudioContext;
    if (!AudioContextCtor) {
      return;
    }

    const context = new AudioContextCtor();
    const oscillator = context.createOscillator();
    const gainNode = context.createGain();

    oscillator.type = "sawtooth";
    oscillator.frequency.value = 880;
    gainNode.gain.value = 0.0001;

    oscillator.connect(gainNode);
    gainNode.connect(context.destination);

    oscillator.start();
    gainNode.gain.exponentialRampToValueAtTime(0.12, context.currentTime + 0.02);
    gainNode.gain.exponentialRampToValueAtTime(0.0001, context.currentTime + 0.45);
    oscillator.stop(context.currentTime + 0.48);

    return () => {
      void context.close();
    };
  }, [alarm.created_at]);

  return (
    <div className="fixed left-1/2 top-4 z-[2000] w-[min(92vw,920px)] -translate-x-1/2">
      <div className="rounded-[28px] border border-[#d94747]/25 bg-[linear-gradient(135deg,rgba(217,71,71,0.96),rgba(111,16,16,0.96))] px-5 py-4 text-white shadow-[0_24px_48px_rgba(111,16,16,0.28)]">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <div className="text-xs font-semibold uppercase tracking-[0.24em] text-white/72">
              SOS Emergency
            </div>
            <div className="mt-2 text-2xl font-semibold">{deviceName} 触发了紧急求救</div>
            <div className="mt-2 text-sm leading-7 text-white/82">
              {alarm.content} · {formatRelativeTime(alarm.created_at)}
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            <ActionButton onClick={onLocate}>定位到地图</ActionButton>
            <ActionButton onClick={onHistory}>历史轨迹</ActionButton>
            <ActionButton onClick={onDismiss} tone="ghost">
              关闭
            </ActionButton>
          </div>
        </div>
      </div>
    </div>
  );
}

function ActionButton({
  children,
  onClick,
  tone = "solid",
}: {
  children: string;
  onClick: () => void;
  tone?: "solid" | "ghost";
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        tone === "ghost"
          ? "rounded-2xl border border-white/18 bg-white/8 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-white/12"
          : "rounded-2xl bg-white px-4 py-2.5 text-sm font-semibold text-[#891c1c] transition hover:bg-white/92"
      }
    >
      {children}
    </button>
  );
}
