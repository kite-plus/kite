import { useState, type FormEvent, type ReactNode } from "react";
import { useNavigate } from "@tanstack/react-router";
import { ArrowLeft, ArrowRight, Check, Loader2, XCircle } from "lucide-react";

import { ApiError } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useInstall, useSetup } from "@/hooks/useSetup";
import { cn } from "@/lib/utils";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LanguageSelect } from "@/components/LanguageSelect";
import { PasswordInput } from "@/components/password-input";
import { AuthLayout } from "../auth-layout";

/**
 * Installing a site, for the half of the world that has no terminal.
 *
 * `kite init` asks these same questions where a project is being created by
 * hand. A server has nobody standing in its directory: it was started by a
 * compose file, and whoever did that is looking at a browser. This is the
 * same installation asked for through a form.
 *
 * There is nobody to check a request against yet, so this page is open to
 * whoever reaches it first. What the server does instead is answer nothing
 * else until the form is finished, which is why it says so little.
 */
export function Setup() {
  const { t } = useI18n();
  const problem = useProblem();
  const navigate = useNavigate();
  const state = useSetup().data;
  const install = useInstall();

  const [step, setStep] = useState<"site" | "account">("site");

  const [title, setTitle] = useState(state?.site?.title ?? "");
  const [baseURL, setBaseURL] = useState(state?.site?.base_url ?? "");
  const [language, setLanguage] = useState(state?.site?.language || "en");

  const [user, setUser] = useState(state?.user ?? "admin");
  const [password, setPassword] = useState("");
  const [again, setAgain] = useState("");

  const minimum = state?.min_password_length ?? 8;
  const missing = minimum - password.length;
  const longEnough = password.length >= minimum;
  const tooShort = password.length > 0 && !longEnough;
  const matches = again.length > 0 && again === password;
  const mismatch = again.length > 0 && again !== password;
  // What the form needs before it is worth sending, which is also what the
  // hints under each field are saying one at a time.
  const ready = Boolean(user) && longEnough && matches;

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (step === "site") {
      setStep("account");
      return;
    }
    if (install.isPending || !ready) return;
    install.mutate(
      { user, password, site: { title, base_url: baseURL, language } },
      { onSuccess: () => void navigate({ to: "/", replace: true }) },
    );
  };

  const failure = install.error;
  // The server's own sentence is kept underneath the translated headline,
  // because which field was wrong is something only the server knows.
  const refusal =
    failure instanceof ApiError
      ? problem(failure.code, failure.message)
      : failure
        ? { title: t("setup.failed"), detail: String(failure) }
        : null;

  return (
    <AuthLayout>
      <Card className="w-full max-w-md gap-4">
        <CardHeader>
          <CardTitle className="text-lg tracking-tight">
            {t(step === "site" ? "setup.site.title" : "setup.account.title")}
          </CardTitle>
          <CardDescription>
            {t(step === "site" ? "setup.site.description" : "setup.account.description")}
          </CardDescription>
          <Steps step={step} />
        </CardHeader>

        <CardContent>
          <form onSubmit={submit} className="grid gap-4">
            {step === "site" ? (
              <>
                <div className="grid gap-2">
                  <Label htmlFor="title">{t("setup.siteTitle")}</Label>
                  <Input
                    id="title"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    autoFocus
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="base-url">{t("setup.baseURL")}</Label>
                  <Input
                    id="base-url"
                    type="url"
                    value={baseURL}
                    onChange={(e) => setBaseURL(e.target.value)}
                    placeholder="https://example.com"
                    required
                  />
                  <p className="text-sm text-muted-foreground">{t("setup.baseURLHelp")}</p>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="language">{t("setup.language")}</Label>
                  <LanguageSelect id="language" value={language} onChange={setLanguage} />
                </div>
                <Button type="submit">
                  {t("setup.next")}
                  <ArrowRight />
                </Button>
              </>
            ) : (
              <>
                <div className="grid gap-2">
                  <Label htmlFor="user">{t("setup.user")}</Label>
                  <Input
                    id="user"
                    value={user}
                    onChange={(e) => setUser(e.target.value)}
                    autoComplete="username"
                    autoFocus
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="password">{t("setup.password")}</Label>
                  <PasswordInput
                    id="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    autoComplete="new-password"
                    required
                  />
                  {tooShort ? (
                    <Hint tone="wrong">{t("setup.passwordNeeds", { count: missing })}</Hint>
                  ) : longEnough ? (
                    <Hint tone="right">{t("setup.passwordOK")}</Hint>
                  ) : (
                    <Hint>{t("setup.passwordHelp", { n: minimum })}</Hint>
                  )}
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="again">{t("setup.again")}</Label>
                  <PasswordInput
                    id="again"
                    value={again}
                    onChange={(e) => setAgain(e.target.value)}
                    autoComplete="new-password"
                    required
                  />
                  {mismatch ? (
                    <Hint tone="wrong">{t("setup.mismatch")}</Hint>
                  ) : matches ? (
                    <Hint tone="right">{t("setup.match")}</Hint>
                  ) : null}
                </div>

                {refusal && (
                  <Alert variant="destructive">
                    <XCircle />
                    <AlertTitle>{refusal.title}</AlertTitle>
                    {refusal.detail && <AlertDescription>{refusal.detail}</AlertDescription>}
                  </Alert>
                )}

                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setStep("site")}
                    disabled={install.isPending}
                  >
                    <ArrowLeft />
                    {t("setup.back")}
                  </Button>
                  <Button type="submit" className="flex-1" disabled={install.isPending || !ready}>
                    {install.isPending ? <Loader2 className="animate-spin" /> : <Check />}
                    {t(install.isPending ? "setup.installing" : "setup.install")}
                  </Button>
                </div>
              </>
            )}
          </form>
        </CardContent>
      </Card>
    </AuthLayout>
  );
}

/**
 * One line of feedback under a field.
 *
 * It says when an answer is right as well as when it is wrong. A form that
 * only speaks up to complain leaves somebody staring at a password they
 * cannot see, wondering whether the two boxes agree.
 */
function Hint({ tone, children }: { tone?: "right" | "wrong"; children: ReactNode }) {
  return (
    // Polite rather than assertive: this changes on every keystroke, and a
    // screen reader interrupting each one would be unusable.
    <p
      aria-live="polite"
      className={cn(
        "text-sm text-muted-foreground",
        tone === "wrong" && "text-destructive",
        tone === "right" && "text-emerald-600 dark:text-emerald-400",
      )}
    >
      {tone === "right" && <Check aria-hidden className="me-1 inline size-3.5 align-[-0.15em]" />}
      {children}
    </p>
  );
}

/** Steps says how far through this is, because a form in two halves should. */
function Steps({ step }: { step: "site" | "account" }) {
  const { t } = useI18n();
  const steps = [
    { id: "site", label: t("setup.step.site") },
    { id: "account", label: t("setup.step.account") },
  ] as const;

  return (
    <ol className="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
      {steps.map((s, i) => {
        const done = s.id === "site" && step === "account";
        const here = s.id === step;
        return (
          <li key={s.id} className="flex items-center gap-2">
            <span
              aria-current={here ? "step" : undefined}
              className={cn(
                "flex size-5 items-center justify-center rounded-full border text-[0.625rem] font-medium",
                here || done ? "border-primary bg-primary text-primary-foreground" : "border-border",
              )}
            >
              {done ? <Check className="size-3" /> : i + 1}
            </span>
            <span className={cn(here && "text-foreground")}>{s.label}</span>
            {i < steps.length - 1 && (
              <span aria-hidden className="text-border">
                —
              </span>
            )}
          </li>
        );
      })}
    </ol>
  );
}
