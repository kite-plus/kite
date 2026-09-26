import { useState, type FormEvent } from "react";
import { useNavigate } from "@tanstack/react-router";
import { ArrowLeft, ArrowRight, Check, Loader2, XCircle } from "lucide-react";

import { ApiError, api } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useInstall, useSetup } from "@/hooks/useSetup";
import { cn } from "@/lib/utils";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LanguageSelect } from "@/components/LanguageSelect";
import { NewPassword, newPasswordReady } from "@/components/new-password";
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
  const { t, locale } = useI18n();
  const problem = useProblem();
  const navigate = useNavigate();
  const state = useSetup().data;
  const install = useInstall();
  // A folder with no site in it yet: the form names the site and stops
  // there, since a studio on this machine alone has no account to create.
  // Read once, because finishing rewrites the answer this page came with.
  const [newSite] = useState(() => Boolean(state?.new_site));
  // Set once the site exists and this page is waiting for it to be served.
  const [opening, setOpening] = useState<"waiting" | "failed">();

  const [step, setStep] = useState<"site" | "account">("site");
  useDocumentTitle(
    t(newSite ? "setup.new.title" : step === "site" ? "setup.site.title" : "setup.account.title"),
  );

  const [title, setTitle] = useState(state?.site?.title ?? "");
  const [baseURL, setBaseURL] = useState(state?.site?.base_url ?? "");
  const [language, setLanguage] = useState(state?.site?.language || (newSite ? locale : "en"));

  const [user, setUser] = useState(state?.user ?? "admin");
  const [password, setPassword] = useState("");
  const [again, setAgain] = useState("");

  const minimum = state?.min_password_length ?? 8;
  // What the form needs before it is worth sending, which is also what the
  // hints under each field are saying one at a time.
  const ready = Boolean(user) && newPasswordReady(password, again, minimum);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (newSite) {
      if (install.isPending || opening === "waiting") return;
      install.mutate(
        { user: "", password: "", site: { title, base_url: baseURL, language } },
        {
          onSuccess: async () => {
            // The command that made the site now serves it at this address,
            // and takes a moment to.
            setOpening("waiting");
            if (await served()) void navigate({ to: "/", replace: true });
            else setOpening("failed");
          },
        },
      );
      return;
    }
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

  const creating = install.isPending || opening === "waiting";

  return (
    <AuthLayout
      title={t(newSite ? "setup.new.title" : step === "site" ? "setup.site.title" : "setup.account.title")}
      description={t(
        newSite
          ? "setup.new.description"
          : step === "site"
            ? "setup.site.description"
            : "setup.account.description",
      )}
      noSite={newSite}
    >
      {!newSite && <Steps step={step} />}
      <form onSubmit={submit} className="grid gap-3">
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
            {newSite ? (
              <>
                {(refusal || opening === "failed") && (
                  <Alert variant="destructive">
                    <XCircle />
                    <AlertTitle>{refusal?.title ?? t("setup.startFailed")}</AlertTitle>
                    {refusal?.detail && <AlertDescription>{refusal.detail}</AlertDescription>}
                  </Alert>
                )}
                <Button type="submit" className="mt-2" disabled={creating || opening === "failed"}>
                  {creating ? <Loader2 className="animate-spin" /> : <Check />}
                  {t(opening === "waiting" ? "setup.starting" : install.isPending ? "setup.creating" : "setup.create")}
                </Button>
              </>
            ) : (
              <Button type="submit" className="mt-2">
                {t("setup.next")}
                <ArrowRight />
              </Button>
            )}
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
            <NewPassword
              password={password}
              again={again}
              onPassword={setPassword}
              onAgain={setAgain}
              minimum={minimum}
            />

            {refusal && (
              <Alert variant="destructive">
                <XCircle />
                <AlertTitle>{refusal.title}</AlertTitle>
                {refusal.detail && <AlertDescription>{refusal.detail}</AlertDescription>}
              </Alert>
            )}

            <div className="mt-2 flex gap-2">
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
    </AuthLayout>
  );
}

/**
 * served waits for the new site to be served here, which it is once the
 * command that created it has handed the address over: until then the
 * address answers that there is no site yet, or not at all.
 */
async function served(): Promise<boolean> {
  for (const until = Date.now() + 20_000; Date.now() < until; ) {
    try {
      const { response } = await api.GET("/site", {});
      if (response.ok) return true;
    } catch {
      // Nothing is listening for the moment between the two servers.
    }
    await new Promise((resolve) => setTimeout(resolve, 300));
  }
  return false;
}

/** Steps says how far through this is, because a form in two halves should. */
function Steps({ step }: { step: "site" | "account" }) {
  const { t } = useI18n();
  const steps = [
    { id: "site", label: t("setup.step.site") },
    { id: "account", label: t("setup.step.account") },
  ] as const;

  return (
    <ol className="flex items-center gap-2 pb-2 text-xs text-muted-foreground">
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
