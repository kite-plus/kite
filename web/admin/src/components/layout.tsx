import { Navigate } from "react-router-dom";
import { FolderKanban, Images, ShieldCheck } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { PageTransition } from "@/components/page-transition";
import { KiteLogo } from "@/components/kite-logo";
import { ThemeToggle } from "@/components/theme-toggle";

const features = [
  {
    icon: FolderKanban,
    title: "统一文件管理",
    desc: "集中管理多后端存储，一处掌控全部资产。",
  },
  {
    icon: Images,
    title: "智能相册归集",
    desc: "自动整理相册与缩略图，浏览轻盈流畅。",
  },
  {
    icon: ShieldCheck,
    title: "精细权限控制",
    desc: "用户、Token、配额一站式配置，安全无忧。",
  },
];

export function AuthLayout() {
  const { user, loading } = useAuth();

  if (loading) {
    return (
      <div className="flex h-svh items-center justify-center bg-background">
        <svg
          className="size-8 animate-[splash-pulse_1.4s_ease-in-out_infinite]"
          viewBox="0 0 32 32"
          fill="currentColor"
          aria-hidden="true"
        >
          <path d="M16 3L28 15L16 27L4 15L16 3Z" />
        </svg>
      </div>
    );
  }

  if (user) {
    return <Navigate to="/dashboard" replace />;
  }

  return (
    <div className="relative grid h-svh lg:grid-cols-2">
      {/* Floating theme toggle — top-right */}
      <ThemeToggle className="absolute top-4 right-4 z-20" />

      {/* Left: form */}
      <div className="flex items-center justify-center px-4 py-8 lg:p-8">
        <PageTransition />
      </div>

      {/* Right: hero — desktop only */}
      <div className="relative hidden h-full flex-col overflow-hidden border-l p-10 lg:flex">
        {/* Layered gradient backdrop */}
        <div
          aria-hidden="true"
          className="absolute inset-0 bg-gradient-to-br from-indigo-50 via-sky-50 to-rose-50 dark:from-indigo-950/40 dark:via-slate-950 dark:to-rose-950/40"
        />
        <div
          aria-hidden="true"
          className="absolute inset-0 bg-[radial-gradient(circle_at_25%_20%,rgba(129,140,248,0.28),transparent_45%),radial-gradient(circle_at_80%_75%,rgba(251,113,133,0.2),transparent_50%),radial-gradient(circle_at_60%_40%,rgba(56,189,248,0.2),transparent_50%)] dark:bg-[radial-gradient(circle_at_25%_20%,rgba(99,102,241,0.22),transparent_45%),radial-gradient(circle_at_80%_75%,rgba(244,63,94,0.18),transparent_50%),radial-gradient(circle_at_60%_40%,rgba(14,165,233,0.2),transparent_50%)]"
        />
        {/* Dot grid texture */}
        <div
          aria-hidden="true"
          className="absolute inset-0 opacity-40 [background-image:radial-gradient(circle,rgba(15,23,42,0.08)_1px,transparent_1px)] [background-size:22px_22px] [mask-image:radial-gradient(ellipse_at_center,black_40%,transparent_85%)] dark:opacity-30 dark:[background-image:radial-gradient(circle,rgba(255,255,255,0.12)_1px,transparent_1px)]"
        />

        {/* Brand */}
        <div className="relative z-10 flex items-center text-lg font-medium text-foreground">
          <KiteLogo className="me-2 size-6" />
          Kite
        </div>

        {/* Slogan + features */}
        <div className="relative z-10 mt-auto max-w-md space-y-8">
          <div>
            <h2 className="text-3xl font-semibold tracking-tight text-foreground">
              乘风而起
            </h2>
            <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
              轻盈自如地管理你的文件、相册与资产。
              <br />
              让存储像风筝一样，灵动且触手可及。
            </p>
          </div>

          <ul className="space-y-4">
            {features.map(({ icon: Icon, title, desc }) => (
              <li key={title} className="flex items-start gap-3">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-border/60 bg-background/60 text-foreground shadow-sm backdrop-blur-sm">
                  <Icon className="size-4" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">{title}</p>
                  <p className="mt-0.5 text-xs leading-relaxed text-muted-foreground">
                    {desc}
                  </p>
                </div>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
}
