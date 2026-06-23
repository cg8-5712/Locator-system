import { FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { login } from "../../services/http/auth";
import { authStore } from "../../stores/auth-store";

export function LoginPage() {
  const navigate = useNavigate();
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("admin123456");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      const result = await login(username, password);
      authStore.getState().setSession({
        token: result.token,
        expiresAt: result.expires_at,
        user: result.user,
      });
      navigate("/app/map", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "登录失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-6 py-10">
      <section className="glass-panel w-full max-w-5xl overflow-hidden rounded-[32px]">
        <div className="grid min-h-[640px] lg:grid-cols-[1.1fr_0.9fr]">
          <div className="relative hidden overflow-hidden bg-[#10212b] px-10 py-12 text-white lg:flex lg:flex-col lg:justify-between">
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(31,136,201,0.35),transparent_28%),radial-gradient(circle_at_bottom_right,rgba(47,158,104,0.22),transparent_30%)]" />
            <div className="relative">
              <p className="text-sm uppercase tracking-[0.28em] text-white/65">
                Locator Hub
              </p>
              <h1 className="mt-6 max-w-md text-5xl font-semibold leading-tight">
                人员定位与安全管理工作台
              </h1>
              <p className="mt-6 max-w-md text-base leading-7 text-white/72">
                面向外勤、巡检、安保和现场作业团队的实时位置、围栏与安全响应系统。
              </p>
            </div>
            <div className="relative grid grid-cols-3 gap-4 text-sm text-white/78">
              <div className="rounded-3xl border border-white/10 bg-white/8 p-4">
                <div className="text-2xl font-semibold text-white">实时</div>
                <div className="mt-2">WebSocket 推送与地图联动</div>
              </div>
              <div className="rounded-3xl border border-white/10 bg-white/8 p-4">
                <div className="text-2xl font-semibold text-white">围栏</div>
                <div className="mt-2">越界检测与停留判读</div>
              </div>
              <div className="rounded-3xl border border-white/10 bg-white/8 p-4">
                <div className="text-2xl font-semibold text-white">SOS</div>
                <div className="mt-2">面向人员安全的响应优先级</div>
              </div>
            </div>
          </div>

          <div className="flex items-center px-6 py-10 sm:px-10">
            <div className="mx-auto w-full max-w-md">
              <p className="text-sm font-medium uppercase tracking-[0.24em] text-[#1f88c9]">
                内部登录
              </p>
              <h2 className="mt-4 text-4xl font-semibold tracking-tight text-[#10212b]">
                进入调度台
              </h2>
              <p className="mt-4 text-sm leading-7 text-[#546570]">
                当前版本已接通后端登录、设备列表和实时通道基础结构。若仅想验证界面交互，可以直接进入
                <Link className="ml-1 font-semibold text-[#1f88c9]" to="/demo/map">
                  死数据验证模式
                </Link>
                。
              </p>

              <form className="mt-10 space-y-5" onSubmit={handleSubmit}>
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-[#10212b]">
                    用户名
                  </span>
                  <input
                    className="w-full rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none transition focus:border-[#1f88c9] focus:ring-4 focus:ring-[#1f88c9]/10"
                    value={username}
                    onChange={(event) => setUsername(event.target.value)}
                  />
                </label>

                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-[#10212b]">
                    密码
                  </span>
                  <input
                    type="password"
                    className="w-full rounded-2xl border border-black/8 bg-white px-4 py-3 outline-none transition focus:border-[#1f88c9] focus:ring-4 focus:ring-[#1f88c9]/10"
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                  />
                </label>

                {error ? (
                  <div className="rounded-2xl border border-[#d94747]/20 bg-[#d94747]/8 px-4 py-3 text-sm text-[#9d2323]">
                    {error}
                  </div>
                ) : null}

                <button
                  type="submit"
                  disabled={submitting}
                  className="w-full rounded-2xl bg-[#10212b] px-4 py-3 text-sm font-semibold text-white transition hover:bg-[#163445] disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {submitting ? "登录中..." : "进入系统"}
                </button>
              </form>

              <div className="mt-4 rounded-2xl border border-[#1f88c9]/18 bg-[#1f88c9]/7 px-4 py-3 text-sm text-[#1b628a]">
                演示入口：
                <Link className="ml-2 font-semibold underline" to="/demo/map">
                  /demo/map
                </Link>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
