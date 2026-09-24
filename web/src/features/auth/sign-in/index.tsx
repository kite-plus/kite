import { useState, type FormEvent } from "react";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { Loader2, LogIn, XCircle } from "lucide-react";

import { ApiError } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useSignIn } from "@/hooks/useSession";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { KiteMark } from "@/components/KiteMark";
import { PasswordInput } from "@/components/password-input";
import { AuthFooter } from "../auth-layout";

/**
 * The sign-in page is shadcn-admin's second one without its picture: the
 * form alone, in the middle of the page.
 */
export function SignIn() {
  const { t } = useI18n();
  const problem = useProblem();
  const navigate = useNavigate();
  const { redirect } = useSearch({ from: "/(auth)/sign-in" });
  const signIn = useSignIn();

  const [user, setUser] = useState("");
  const [password, setPassword] = useState("");
  const [remember, setRemember] = useState(false);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!user || !password || signIn.isPending) return;
    signIn.mutate(
      { user, password, remember },
      // Back to where the studio was when it asked, which a link from outside
      // could only have made a path within it.
      { onSuccess: () => void navigate({ href: redirect || "/", replace: true }) },
    );
  };

  const failure = signIn.error;
  // The server's own sentence is kept for everything except a wrong password,
  // where the translated line already says the whole of it. A refusal after
  // too many attempts, for instance, is the only place the wait is written.
  const refusal =
    failure instanceof ApiError
      ? problem(failure.code, failure.code === "unauthorized" ? undefined : failure.message)
      : failure
        ? { title: t("login.failed"), detail: String(failure) }
        : null;

  return (
    <div className="relative container grid h-svh flex-col items-center justify-center">
      <div className="lg:p-8">
        <div className="mx-auto flex w-full flex-col justify-center space-y-2 py-8 sm:w-120 sm:p-8">
          <div className="mb-4 flex items-center justify-center">
            <KiteMark className="me-2 size-6" />
            <h1 className="text-xl font-medium">Kite</h1>
          </div>
        </div>
        <div className="mx-auto flex w-full max-w-sm flex-col justify-center space-y-2">
          <div className="flex flex-col space-y-2 text-start">
            <h2 className="text-lg font-semibold tracking-tight">{t("login.title")}</h2>
            <p className="text-sm text-muted-foreground">{t("login.description")}</p>
          </div>
          <form onSubmit={submit} className="grid gap-3">
            <div className="grid gap-2">
              <Label htmlFor="user">{t("login.user")}</Label>
              <Input
                id="user"
                value={user}
                onChange={(e) => setUser(e.target.value)}
                autoComplete="username"
                autoFocus
                required
              />
            </div>
            <div className="relative grid gap-2">
              <Label htmlFor="password">{t("login.password")}</Label>
              <PasswordInput
                id="password"
                placeholder="********"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                required
              />
              <ForgotPassword />
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                id="remember"
                checked={remember}
                onCheckedChange={(checked) => setRemember(checked === true)}
              />
              <Label htmlFor="remember" className="font-normal">
                {t("login.remember")}
              </Label>
            </div>

            {refusal && (
              <Alert variant="destructive">
                <XCircle />
                <AlertTitle>{refusal.title}</AlertTitle>
                {refusal.detail && <AlertDescription>{refusal.detail}</AlertDescription>}
              </Alert>
            )}

            <Button type="submit" className="mt-2" disabled={signIn.isPending}>
              {signIn.isPending ? <Loader2 className="animate-spin" /> : <LogIn />}
              {t(signIn.isPending ? "login.submitting" : "login.submit")}
            </Button>
          </form>
          <AuthFooter />
        </div>
      </div>
    </div>
  );
}

/**
 * There is no mail to send a reset link to: the account lives on the server,
 * and a new password is set there.
 */
function ForgotPassword() {
  const { t } = useI18n();
  return (
    <Popover>
      <PopoverTrigger
        type="button"
        className="absolute inset-e-0 -top-0.5 text-sm font-medium text-muted-foreground hover:opacity-75"
      >
        {t("login.forgot")}
      </PopoverTrigger>
      <PopoverContent align="end" className="grid w-80 gap-2 text-sm">
        <p>{t("login.forgotNote")}</p>
        <code className="rounded-md bg-muted px-2 py-1.5 font-mono text-xs">
          kite auth set-password
        </code>
        <p className="text-muted-foreground">{t("login.forgotEnv")}</p>
      </PopoverContent>
    </Popover>
  );
}
