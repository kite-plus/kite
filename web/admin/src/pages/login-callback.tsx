import { useEffect, useRef } from "react";
import { Loader2 } from "lucide-react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { authApi } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { toast } from "sonner";

export default function LoginCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { applyTokensAndRefresh } = useAuth();
  const startedRef = useRef(false);

  useEffect(() => {
    if (startedRef.current) return;
    startedRef.current = true;

    const ticket = searchParams.get("ticket");
    if (!ticket) {
      toast.error("登录票据缺失，请重新尝试");
      navigate("/login", { replace: true });
      return;
    }

    authApi
      .exchangeOAuth(ticket)
      .then(async (res) => {
        const data = res.data.data;
        await applyTokensAndRefresh({
          access_token: data.access_token,
          refresh_token: data.refresh_token,
        });
        navigate(data.return_to || "/user/dashboard", { replace: true });
      })
      .catch((err: unknown) => {
        const backendMsg =
          (
            err as { response?: { data?: { message?: string } } }
          )?.response?.data?.message ?? "第三方登录失败，请稍后重试";
        toast.error(backendMsg);
        navigate(`/login?oauth_error=${encodeURIComponent(backendMsg)}`, {
          replace: true,
        });
      });
  }, [applyTokensAndRefresh, navigate, searchParams]);

  return (
    <div className="flex min-h-[420px] flex-col items-center justify-center gap-3 text-center">
      <Loader2 className="size-8 animate-spin text-muted-foreground" />
      <div>
        <p className="text-sm font-medium">正在完成第三方登录</p>
        <p className="text-xs text-muted-foreground">
          请稍候，我们马上带你进入 Kite。
        </p>
      </div>
    </div>
  );
}
