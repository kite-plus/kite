import { useState, type FormEvent } from "react";
import { ArrowLeft, ArrowRight, Check, Eye, EyeOff, XCircle } from "lucide-react";

import { ApiError } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useInstall, type SetupState } from "@/hooks/useSetup";

import { KiteMark } from "@/components/KiteMark";
import { LanguagePicker } from "@/components/LanguagePicker";
import { LanguageSelect } from "@/components/LanguageSelect";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Spinner } from "@/components/ui/spinner";

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
export function SetupPage({ state }: { state: SetupState }) {
  const { t } = useI18n();
  const problem = useProblem();
  const install = useInstall();

  const [step, setStep] = useState<"site" | "account">("site");

  const [title, setTitle] = useState(state.site?.title ?? "");
  const [baseURL, setBaseURL] = useState(state.site?.base_url ?? "");
  const [language, setLanguage] = useState(state.site?.language || "en");

  const [user, setUser] = useState(state.user ?? "admin");
  const [password, setPassword] = useState("");
  const [again, setAgain] = useState("");
  const [shown, setShown] = useState(false);

  const minimum = state.min_password_length ?? 8;
  const tooShort = password.length > 0 && password.length < minimum;
  const mismatch = again.length > 0 && again !== password;

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (step === "site") {
      setStep("account");
      return;
    }
    if (install.isPending || tooShort || mismatch || !password || !user) return;
    install.mutate({
      user,
      password,
      site: { title, base_url: baseURL, language },
    });
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
    <div className="flex min-h-svh flex-col items-center justify-center gap-6 bg-muted p-6 md:p-10">
      <div className="flex w-full max-w-md flex-col gap-6">
        <div className="flex items-center justify-center gap-2">
          <KiteMark className="size-8" />
          <span className="text-2xl font-bold tracking-tight">Kite</span>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>
              {t(step === "site" ? "setup.site.title" : "setup.account.title")}
            </CardTitle>
            <CardDescription>
              {t(step === "site" ? "setup.site.description" : "setup.account.description")}
            </CardDescription>
            <Steps step={step} />
          </CardHeader>

          <CardContent>
            <form onSubmit={submit}>
              {step === "site" ? (
                <FieldGroup>
                  <Field>
                    <FieldLabel htmlFor="title">{t("setup.siteTitle")}</FieldLabel>
                    <Input
                      id="title"
                      value={title}
                      onChange={(e) => setTitle(e.target.value)}
                      autoFocus
                      required
                    />
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="base-url">{t("setup.baseURL")}</FieldLabel>
                    <Input
                      id="base-url"
                      type="url"
                      value={baseURL}
                      onChange={(e) => setBaseURL(e.target.value)}
                      placeholder="https://example.com"
                      required
                    />
                    <FieldDescription>{t("setup.baseURLHelp")}</FieldDescription>
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="language">{t("setup.language")}</FieldLabel>
                    <LanguageSelect id="language" value={language} onChange={setLanguage} />
                  </Field>

                  <Field>
                    <Button type="submit">
                      {t("setup.next")}
                      <ArrowRight data-icon="inline-end" />
                    </Button>
                  </Field>
                </FieldGroup>
              ) : (
                <FieldGroup>
                  <Field>
                    <FieldLabel htmlFor="user">{t("setup.user")}</FieldLabel>
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
                    <FieldLabel htmlFor="password">{t("setup.password")}</FieldLabel>
                    <InputGroup>
                      <InputGroupInput
                        id="password"
                        type={shown ? "text" : "password"}
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        autoComplete="new-password"
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
                    <FieldDescription>
                      {tooShort ? t("setup.passwordShort", { n: minimum }) : t("setup.passwordHelp", { n: minimum })}
                    </FieldDescription>
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="again">{t("setup.again")}</FieldLabel>
                    <Input
                      id="again"
                      type={shown ? "text" : "password"}
                      value={again}
                      onChange={(e) => setAgain(e.target.value)}
                      autoComplete="new-password"
                      required
                    />
                    {mismatch && (
                      <FieldDescription className="text-destructive">
                        {t("setup.mismatch")}
                      </FieldDescription>
                    )}
                  </Field>

                  {refusal && (
                    <Alert variant="destructive">
                      <XCircle />
                      <AlertTitle>{refusal.title}</AlertTitle>
                      {refusal.detail && <AlertDescription>{refusal.detail}</AlertDescription>}
                    </Alert>
                  )}

                  <Field orientation="horizontal">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => setStep("site")}
                      disabled={install.isPending}
                    >
                      <ArrowLeft data-icon="inline-start" />
                      {t("setup.back")}
                    </Button>
                    <Button
                      type="submit"
                      className="flex-1"
                      disabled={install.isPending || tooShort || mismatch}
                    >
                      {install.isPending ? (
                        <Spinner data-icon="inline-start" />
                      ) : (
                        <Check data-icon="inline-start" />
                      )}
                      {t(install.isPending ? "setup.installing" : "setup.install")}
                    </Button>
                  </Field>
                </FieldGroup>
              )}
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
              className={[
                "flex size-5 items-center justify-center rounded-full border text-[0.625rem] font-medium",
                here || done
                  ? "border-primary bg-primary text-primary-foreground"
                  : "border-border",
              ].join(" ")}
            >
              {done ? <Check className="size-3" /> : i + 1}
            </span>
            <span className={here ? "text-foreground" : undefined}>{s.label}</span>
            {i < steps.length - 1 && <span aria-hidden className="text-border">—</span>}
          </li>
        );
      })}
    </ol>
  );
}
