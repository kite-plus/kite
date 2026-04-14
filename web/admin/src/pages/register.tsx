import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { KiteLogo } from "@/components/kite-logo";
import { toast } from "sonner";

export default function RegisterPage() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({
    username: "",
    email: "",
    password: "",
    confirmPassword: "",
  });
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (form.password !== form.confirmPassword) {
      toast.error("两次输入的密码不一致，请重新核对");
      return;
    }

    setLoading(true);
    try {
      await register(form.username, form.email, form.password);
      toast.success("注册成功，请重新登录！");
      navigate("/login");
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? "注册失败，请稍后重试";
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  const update =
    (field: string) => (e: React.ChangeEvent<HTMLInputElement>) =>
      setForm((prev) => ({ ...prev, [field]: e.target.value }));

  return (
    <>
      <div className="mx-auto flex w-full flex-col justify-center space-y-2 py-8 sm:w-[480px] sm:p-8">
        <div className="mb-4 flex items-center justify-center">
          <KiteLogo className="me-2 size-6" />
          <h1 className="text-xl font-medium">Kite</h1>
        </div>
      </div>

      <div className="mx-auto flex w-full max-w-sm flex-col justify-center">
        <div className="flex flex-col space-y-2 text-start">
          <h2 className="text-2xl font-semibold tracking-tight">创建账号</h2>
          <p className="text-sm text-muted-foreground">
            填写信息，开启你的 Kite 之旅
          </p>
        </div>

        <form onSubmit={handleSubmit} className="grid gap-3 pt-2">
          <div className="grid gap-2">
            <Label htmlFor="username">用户名</Label>
            <Input
              id="username"
              autoCapitalize="none"
              autoCorrect="off"
              value={form.username}
              onChange={update("username")}
              placeholder="请输入用户名"
              required
              minLength={3}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="email">邮箱</Label>
            <Input
              id="email"
              type="email"
              autoCapitalize="none"
              autoComplete="email"
              autoCorrect="off"
              value={form.email}
              onChange={update("email")}
              placeholder="name@example.com"
              required
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="password">密码</Label>
            <Input
              id="password"
              type="password"
              value={form.password}
              onChange={update("password")}
              placeholder="••••••••"
              required
              minLength={6}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="confirmPassword">确认密码</Label>
            <Input
              id="confirmPassword"
              type="password"
              value={form.confirmPassword}
              onChange={update("confirmPassword")}
              placeholder="••••••••"
              required
            />
          </div>

          <Button type="submit" className="mt-2" disabled={loading}>
            {loading && <Loader2 className="size-4 animate-spin" />}
            {loading ? "创建中..." : "注册"}
          </Button>
        </form>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          已有账号？{" "}
          <Link
            to="/login"
            className="font-medium text-foreground underline-offset-4 hover:underline"
          >
            立即登录
          </Link>
        </p>

        <p className="mt-4 text-center text-xs text-muted-foreground">
          点击注册即表示您同意我们的{" "}
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
