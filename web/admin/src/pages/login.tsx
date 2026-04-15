import { useState } from "react";
import { Link, useNavigate, useLocation } from "react-router-dom";
import { Eye, EyeOff, Loader2, LogIn } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { KiteLogo } from "@/components/kite-logo";
import { toast } from "sonner";

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    const minDelay = new Promise((r) => setTimeout(r, 600));
    try {
      await Promise.all([login(username, password), minDelay]);
      toast.success("登录成功啦！");
      const redirectTo =
        (location.state as { from?: string } | null)?.from ?? "/dashboard";
      navigate(redirectTo, { replace: true });
    } catch (err) {
      await minDelay;
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? "账号或密码错误，请重试";
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <div className="mx-auto flex w-full flex-col justify-center space-y-2 py-8 sm:w-[480px] sm:p-8">
        <div className="mb-4 flex items-center justify-center">
          <KiteLogo className="me-2 size-6" />
          <h1 className="text-xl font-medium">Kite</h1>
        </div>
      </div>

      <div className="mx-auto flex w-full max-w-sm flex-col justify-center">
        <div className="flex flex-col space-y-1.5 text-start">
          <h2 className="text-2xl font-semibold tracking-tight">欢迎回来</h2>
          <p className="text-sm text-muted-foreground">
            请登录你的 Kite 账号
          </p>
        </div>

        <form onSubmit={handleSubmit} className="grid gap-3 pt-2">
          <div className="grid gap-2">
            <Label htmlFor="username">账号</Label>
            <Input
              id="username"
              autoCapitalize="none"
              autoComplete="email"
              autoCorrect="off"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="name@example.com"
              required
            />
          </div>

          <div className="grid gap-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="password">密码</Label>
              <Link
                to="#"
                tabIndex={-1}
                className="text-xs font-medium text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
              >
                忘记密码？
              </Link>
            </div>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? "text" : "password"}
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="pr-9"
                required
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                tabIndex={-1}
                className="absolute right-1 top-1/2 flex size-7 -translate-y-1/2 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                aria-label={showPassword ? "隐藏密码" : "显示密码"}
              >
                {showPassword ? (
                  <EyeOff className="size-4" />
                ) : (
                  <Eye className="size-4" />
                )}
              </button>
            </div>
          </div>

          <Button type="submit" className="mt-2" disabled={loading}>
            {loading ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <LogIn className="size-4" />
            )}
            {loading ? "登录中..." : "登录"}
          </Button>

          <div className="relative my-2">
            <div className="absolute inset-0 flex items-center">
              <span className="w-full border-t" />
            </div>
            <div className="relative flex justify-center text-xs uppercase">
              <span className="bg-background px-2 text-muted-foreground">
                或使用以下方式继续
              </span>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <Button variant="outline" type="button" disabled={loading}>
              <svg
                className="size-4"
                viewBox="0 0 24 24"
                fill="currentColor"
                aria-hidden="true"
              >
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
              </svg>
              GitHub
            </Button>
            <Button variant="outline" type="button" disabled={loading}>
              <svg className="size-4" viewBox="0 0 24 24" aria-hidden="true">
                <path
                  fill="#4285F4"
                  d="M23.49 12.27c0-.79-.07-1.54-.19-2.27H12v4.51h6.44c-.29 1.48-1.14 2.73-2.4 3.58v2.97h3.86c2.26-2.09 3.59-5.17 3.59-8.79z"
                />
                <path
                  fill="#34A853"
                  d="M12 24c3.24 0 5.95-1.08 7.93-2.91l-3.86-2.97c-1.08.72-2.45 1.16-4.07 1.16-3.13 0-5.78-2.11-6.73-4.96H1.29v3.09C3.26 21.3 7.31 24 12 24z"
                />
                <path
                  fill="#FBBC05"
                  d="M5.27 14.31c-.25-.72-.38-1.49-.38-2.31s.14-1.59.38-2.31V6.6H1.29A11.954 11.954 0 000 12c0 1.94.46 3.78 1.29 5.4l3.98-3.09z"
                />
                <path
                  fill="#EA4335"
                  d="M12 4.75c1.77 0 3.35.61 4.6 1.8l3.42-3.42C17.95 1.19 15.24 0 12 0 7.31 0 3.26 2.7 1.29 6.6l3.98 3.09C6.22 6.86 8.87 4.75 12 4.75z"
                />
              </svg>
              Google
            </Button>
          </div>
        </form>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          还没有账号？{" "}
          <Link
            to="/register"
            className="font-medium text-foreground underline-offset-4 hover:underline"
          >
            立即注册
          </Link>
        </p>

        <p className="mt-4 text-center text-xs text-muted-foreground">
          点击登录即表示您同意我们的{" "}
          <a
            href="#"
            className="underline underline-offset-4 hover:text-foreground"
          >
            服务条款
          </a>{" "}
          和{" "}
          <a
            href="#"
            className="underline underline-offset-4 hover:text-foreground"
          >
            隐私政策
          </a>
          。
        </p>
      </div>
    </>
  );
}
