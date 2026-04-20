import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";

import { authApi } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";

interface AuthOptions {
  allow_registration: boolean;
}

export default function CompleteSocialPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { applyTokensAndRefresh } = useAuth();
  const ticket = searchParams.get("ticket") ?? "";
  const checkedRef = useRef(false);
  const [form, setForm] = useState({
    username: "",
    email: "",
  });

  const { data: authOptions, isLoading } = useQuery<AuthOptions>({
    queryKey: ["auth", "options"],
    queryFn: () => authApi.options().then((r) => r.data.data),
    retry: 0,
  });

  useEffect(() => {
    if (checkedRef.current) return;
    checkedRef.current = true;
    if (!ticket) {
      toast.error("登录票据缺失，请重新发起第三方登录");
      navigate("/login", { replace: true });
    }
  }, [navigate, ticket]);

  const onboardMutation = useMutation({
    mutationFn: () =>
      authApi.onboardOAuth({
        ticket,
        username: form.username.trim(),
        email: form.email.trim(),
      }),
    onSuccess: async (res) => {
      const data = res.data.data;
      await applyTokensAndRefresh({
        access_token: data.access_token,
        refresh_token: data.refresh_token,
      });
      toast.success("账号创建成功，欢迎使用 Kite！");
      navigate(data.return_to || "/user/dashboard", { replace: true });
    },
    onError: (err: unknown) => {
      const backendMsg =
        (
          err as { response?: { data?: { message?: string } } }
        )?.response?.data?.message ?? "补全资料失败，请稍后重试";
      toast.error(backendMsg);
    },
  });

  if (isLoading) {
    return (
      <div className="flex min-h-[420px] items-center justify-center">
        <Loader2 className="size-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (authOptions?.allow_registration === false) {
    return (
      <div className="space-y-4 text-center">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">暂不可完成注册</h2>
          <p className="mt-2 text-sm text-muted-foreground">
            当前站点未开放注册，请联系管理员为你开通账号后再绑定第三方登录。
          </p>
        </div>
        <Button asChild>
          <Link to="/login">返回登录</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="space-y-2 text-center">
        <h2 className="text-2xl font-semibold tracking-tight">补全账号信息</h2>
        <p className="text-sm text-muted-foreground">
          本次第三方登录没有返回可直接使用的邮箱，请补全用户名和邮箱后继续。
        </p>
      </div>

      <form
        className="grid gap-4"
        onSubmit={(e) => {
          e.preventDefault();
          if (!form.username.trim() || !form.email.trim()) {
            toast.error("请完整填写用户名和邮箱");
            return;
          }
          onboardMutation.mutate();
        }}
      >
        <div className="grid gap-2">
          <Label htmlFor="username">用户名</Label>
          <Input
            id="username"
            value={form.username}
            onChange={(e) =>
              setForm((prev) => ({ ...prev, username: e.target.value }))
            }
            minLength={3}
            maxLength={32}
            placeholder="amigoer"
            required
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="email">邮箱</Label>
          <Input
            id="email"
            type="email"
            value={form.email}
            onChange={(e) =>
              setForm((prev) => ({ ...prev, email: e.target.value }))
            }
            placeholder="name@example.com"
            required
          />
        </div>

        <Button type="submit" disabled={onboardMutation.isPending}>
          {onboardMutation.isPending && (
            <Loader2 className="size-4 animate-spin" />
          )}
          完成并进入 Kite
        </Button>
      </form>
    </div>
  );
}
