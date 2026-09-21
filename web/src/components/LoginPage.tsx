import { useState, type FormEvent } from "react";
import { Eye, EyeOff, LogIn, XCircle } from "lucide-react";

import { ApiError } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useSignIn } from "@/hooks/useSession";

import { KiteMark } from "@/components/KiteMark";
import { LanguagePicker } from "@/components/LanguagePicker";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Spinner } from "@/components/ui/spinner";

/**
 * The door.
 *
 * It says as little as it can: no site title, no account name, nothing about
 * what is behind it. Everything a studio knows comes from an API that will
 * not answer until somebody has signed in, and a sign-in page that filled
 * itself in from somewhere else would be telling a stranger what this server
 * holds.
 */
export function LoginPage() {
  const { t } = useI18n();
  const problem = useProblem();
  const signIn = useSignIn();

  const [user, setUser] = useState("");
  const [password, setPassword] = useState("");
  const [remember, setRemember] = useState(false);
  const [shown, setShown] = useState(false);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!user || !password || signIn.isPending) return;
    signIn.mutate({ user, password, remember });
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
    <div className="flex min-h-svh flex-col items-center justify-center gap-6 bg-muted p-6 md:p-10">
      <div className="flex w-full max-w-sm flex-col gap-6">
        <div className="flex items-center justify-center gap-2">
          <KiteMark className="size-8" />
          <span className="text-2xl font-bold tracking-tight">Kite</span>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>{t("login.title")}</CardTitle>
            <CardDescription>{t("login.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={submit}>
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="user">{t("login.user")}</FieldLabel>
                  <Input
                    id="user"
                    value={user}
                    onChange={(e) => setUser(e.target.value)}
                    autoComplete="username"
                    autoFocus
                    required
                  />
                </Field>

                <Field>
                  <FieldLabel htmlFor="password">{t("login.password")}</FieldLabel>
                  <InputGroup>
                    <InputGroupInput
                      id="password"
                      type={shown ? "text" : "password"}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      autoComplete="current-password"
                      required
                    />
                    <InputGroupAddon align="inline-end">
                      <InputGroupButton
                        aria-label={t(shown ? "login.hidePassword" : "login.showPassword")}
                        onClick={() => setShown((s) => !s)}
                      >
                        {shown ? <EyeOff /> : <Eye />}
                      </InputGroupButton>
                    </InputGroupAddon>
                  </InputGroup>
                </Field>

                <Field orientation="horizontal">
                  <Checkbox
                    id="remember"
                    checked={remember}
                    onCheckedChange={(checked) => setRemember(checked === true)}
                  />
                  <FieldLabel htmlFor="remember" className="font-normal">
                    {t("login.remember")}
                  </FieldLabel>
                </Field>

                {refusal && (
                  <Alert variant="destructive">
                    <XCircle />
                    <AlertTitle>{refusal.title}</AlertTitle>
                    {refusal.detail && (
                      <AlertDescription>{refusal.detail}</AlertDescription>
                    )}
                  </Alert>
                )}

                <Field>
                  <Button type="submit" disabled={signIn.isPending}>
                    {signIn.isPending ? (
                      <Spinner data-icon="inline-start" />
                    ) : (
                      <LogIn data-icon="inline-start" />
                    )}
                    {t(signIn.isPending ? "login.submitting" : "login.submit")}
                  </Button>
                </Field>
              </FieldGroup>
            </form>
          </CardContent>
        </Card>

        <div className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
          <a href="/" className="transition-colors hover:text-foreground">
            {t("login.backToSite")}
          </a>
          <span aria-hidden>·</span>
          <LanguagePicker showLabel />
        </div>
      </div>
    </div>
  );
}
