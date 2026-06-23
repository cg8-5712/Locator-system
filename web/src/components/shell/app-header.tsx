import type { ReactNode } from "react";
import { NavLink } from "react-router-dom";
import { useAuth } from "../../hooks/use-auth";
import { buildModePath } from "../../features/map-view/mode";
import type { AppMode } from "../../features/map-view/mode";
import { authStore } from "../../stores/auth-store";

type MetricTone = "default" | "brand" | "warn" | "danger";

export interface HeaderMetric {
  label: string;
  value: string;
  tone?: MetricTone;
}

const text = {
  currentUser: "\u5f53\u524d\u7528\u6237",
  demoVisitor: "\u6f14\u793a\u8bbf\u5ba2",
  logout: "\u9000\u51fa\u767b\u5f55",
  map: "\u5730\u56fe\u603b\u89c8",
  alarms: "\u544a\u8b66\u4e2d\u5fc3",
};

export function AppHeader({
  mode,
  title,
  description,
  metrics,
  active,
  children,
}: {
  mode: AppMode;
  title: string;
  description?: string;
  metrics: HeaderMetric[];
  active: "map" | "alarms" | "history";
  children?: ReactNode;
}) {
  const { user } = useAuth();

  return (
    <header className="glass-panel flex flex-col gap-4 rounded-[28px] px-5 py-4">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.24em] text-[#1f88c9]">
            Locator Hub
          </p>
          <h1 className="mt-2 text-3xl font-semibold tracking-tight text-[#10212b]">
            {title}
          </h1>
          {description ? (
            <p className="mt-2 max-w-2xl text-sm leading-7 text-[#546570]">
              {description}
            </p>
          ) : null}
        </div>

        <div className="flex flex-wrap items-center gap-3">
          {metrics.map((metric) => (
            <TopMetric
              key={`${metric.label}-${metric.value}`}
              label={metric.label}
              value={metric.value}
              tone={metric.tone}
            />
          ))}

          <TopMetric
            label={text.currentUser}
            value={mode === "demo" ? text.demoVisitor : user?.username ?? "--"}
          />

          {mode === "live" ? (
            <button
              type="button"
              onClick={() => authStore.getState().clearSession()}
              className="rounded-2xl border border-black/8 bg-white/72 px-4 py-2.5 text-sm font-semibold text-[#10212b] transition hover:bg-white"
            >
              {text.logout}
            </button>
          ) : null}
        </div>
      </div>

      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <nav className="flex flex-wrap gap-2">
          <ModeNavItem to={buildModePath(mode, "/map")} active={active === "map"}>
            {text.map}
          </ModeNavItem>
          <ModeNavItem
            to={buildModePath(mode, "/alarms")}
            active={active === "alarms"}
          >
            {text.alarms}
          </ModeNavItem>
        </nav>
        {children ? <div className="flex flex-wrap items-center gap-2">{children}</div> : null}
      </div>
    </header>
  );
}

function ModeNavItem({
  to,
  active,
  children,
}: {
  to: string;
  active: boolean;
  children: ReactNode;
}) {
  return (
    <NavLink
      to={to}
      className={`rounded-full px-4 py-2 text-sm font-semibold transition ${
        active
          ? "bg-[#10212b] text-white"
          : "border border-black/8 bg-white/72 text-[#10212b] hover:bg-white"
      }`}
    >
      {children}
    </NavLink>
  );
}

function TopMetric({
  label,
  value,
  tone = "default",
}: {
  label: string;
  value: string;
  tone?: MetricTone;
}) {
  const toneClassName =
    tone === "brand"
      ? "border-[#1f88c9]/18 bg-[#1f88c9]/8"
      : tone === "warn"
        ? "border-[#d48a1f]/18 bg-[#d48a1f]/10"
        : tone === "danger"
          ? "border-[#d94747]/18 bg-[#d94747]/10"
          : "border-black/6 bg-white/66";

  return (
    <div className={`rounded-2xl border px-4 py-2.5 ${toneClassName}`}>
      <div className="text-[11px] uppercase tracking-[0.18em] text-[#7a8a94]">
        {label}
      </div>
      <div className="mt-1 text-sm font-semibold text-[#10212b]">{value}</div>
    </div>
  );
}
