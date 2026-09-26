import { useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { useLocation } from "@tanstack/react-router";
import { ImageUp, Info, KeyRound, LogOut, MonitorSmartphone, ShieldAlert, Trash2, XCircle } from "lucide-react";
import { toast } from "sonner";

import { ApiError } from "@/api/client";
import { useI18n, useProblem, type Key } from "@/i18n";
import {
  useAccountInfo,
  useEndOtherSessions,
  useRemoveAvatar,
  useRemoveCredentials,
  useSetCredentials,
  useUpdateProfile,
  useUploadAvatar,
  type Account,
} from "@/hooks/useAccount";
import { squarePicture } from "@/lib/avatar";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { UserAvatar } from "@/components/layout/account";
import { Hint, NewPassword, newPasswordReady } from "@/components/new-password";
import { PasswordInput } from "@/components/password-input";
import { QueryError } from "@/components/query-error";
import { SignOutDialog } from "@/components/sign-out-dialog";
import { ContentSection } from "../components/content-section";
import { Group, Row } from "../components/form-group";

/**
 * The account is the person's, not the site's: how they are shown in the
 * studio, and how they sign in. None of it is in kite.yaml, so nothing here
 * waits to be published.
 */
export function AccountSettings() {
  const { t } = useI18n();
  const account = useAccountInfo();
  const { hash } = useLocation();
  const loaded = Boolean(account.data);

  // The menu's "set a password" lands here; the group it means is below the
  // profile, so it is brought into view once there is something to show.
  useEffect(() => {
    if (loaded && hash === "sign-in") document.getElementById("sign-in")?.scrollIntoView({ block: "start" });
  }, [loaded, hash]);

  return (
    <ContentSection title={t("settings.account")} desc={t("settings.accountNote")}>
      {account.data ? (
        <div className="grid gap-10">
          <ProfileGroup account={account.data} />
          {account.data.protected ? (
            <SignInGroup account={account.data} />
          ) : (
            <SetPasswordGroup account={account.data} />
          )}
          {account.data.session && <SessionsGroup account={account.data} />}
          {account.data.removable && <RemoveGroup />}
        </div>
      ) : account.isError ? (
        <QueryError error={account.error} onRetry={() => void account.refetch()} />
      ) : (
        <Skeleton className="h-72 w-full" />
      )}
    </ContentSection>
  );
}

function ProfileGroup({ account }: { account: Account }) {
  const { t } = useI18n();
  const update = useUpdateProfile();
  const upload = useUploadAvatar();
  const removeAvatar = useRemoveAvatar();
  const picker = useRef<HTMLInputElement>(null);

  const [name, setName] = useState(account.profile.name);
  const [email, setEmail] = useState(account.profile.email);
  // What was stored is what the server trimmed, so the form follows it.
  useEffect(() => setName(account.profile.name), [account.profile.name]);
  useEffect(() => setEmail(account.profile.email), [account.profile.email]);

  const locked = !account.profile_editable;
  const dirty = name.trim() !== account.profile.name || email.trim() !== account.profile.email;
  // Judged once the field is left, not while the address is half typed.
  const [emailLeft, setEmailLeft] = useState(false);
  const emailOK = email.trim() === "" || looksLikeEmail(email.trim());

  const save = (e: FormEvent) => {
    e.preventDefault();
    update.mutate({ name, email }, { onSuccess: () => toast.success(t("account.profileSaved")) });
  };

  const choose = async (file: File | undefined) => {
    if (!file) return;
    let picture: Blob;
    try {
      picture = await squarePicture(file);
    } catch {
      toast.error(t("account.avatarUnreadable"));
      return;
    }
    upload.mutate(picture);
  };

  return (
    <Group title={t("account.groupProfile")}>
      <div className="flex items-center gap-4">
        <UserAvatar large />
        <div className="grid gap-2">
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={locked || upload.isPending}
              onClick={() => picker.current?.click()}
            >
              {upload.isPending ? <Spinner /> : <ImageUp />}
              {t(account.avatar ? "account.changeAvatar" : "account.uploadAvatar")}
            </Button>
            {account.avatar && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={locked || removeAvatar.isPending}
                onClick={() => removeAvatar.mutate()}
              >
                <Trash2 />
                {t("account.removeAvatar")}
              </Button>
            )}
          </div>
          <p className="text-sm text-muted-foreground">{t("account.avatarHelp")}</p>
        </div>
        <input
          ref={picker}
          type="file"
          accept="image/png,image/jpeg,image/gif,image/webp"
          className="hidden"
          onChange={(e) => {
            void choose(e.target.files?.[0]);
            // The same file chosen again is a change too.
            e.target.value = "";
          }}
        />
      </div>
      <Problem error={upload.error ?? removeAvatar.error} />

      {/* The server says what is wrong with an address, in the page rather
          than in the browser's own bubble. */}
      <form onSubmit={save} noValidate className="grid gap-6">
        <Row id="profile-name" label="account.name" help="account.nameHelp">
          <Input
            id="profile-name"
            value={name}
            placeholder={account.user}
            maxLength={64}
            autoComplete="name"
            disabled={locked}
            onChange={(e) => setName(e.target.value)}
          />
        </Row>
        <div className="grid gap-2">
          <Label htmlFor="profile-email">{t("account.email")}</Label>
          <Input
            id="profile-email"
            type="email"
            value={email}
            autoComplete="email"
            disabled={locked}
            aria-invalid={emailLeft && !emailOK}
            onChange={(e) => setEmail(e.target.value)}
            onBlur={() => setEmailLeft(true)}
          />
          {emailLeft && !emailOK ? (
            <Hint tone="wrong">{t("account.emailInvalid")}</Hint>
          ) : (
            <p className="text-sm text-muted-foreground">{t("account.emailHelp")}</p>
          )}
        </div>
        <Problem error={update.error} />
        <div>
          <Button type="submit" disabled={locked || !dirty || !emailOK || update.isPending}>
            {update.isPending && <Spinner />}
            {t("account.saveProfile")}
          </Button>
        </div>
      </form>
    </Group>
  );
}

/** SetPasswordGroup gives a studio with no password its first one. */
function SetPasswordGroup({ account }: { account: Account }) {
  const { t } = useI18n();
  const set = useSetCredentials();
  const [user, setUser] = useState("admin");
  const [password, setPassword] = useState("");
  const [again, setAgain] = useState("");
  const ready = user.trim() !== "" && newPasswordReady(password, again, account.min_password_length);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!ready || set.isPending) return;
    set.mutate(
      { user: user.trim(), password },
      { onSuccess: () => toast.success(t("account.passwordSet")) },
    );
  };

  return (
    <Group id="sign-in" title={t("account.groupSignIn")}>
      <Note icon={<ShieldAlert />}>{t("account.openNote")}</Note>
      {account.editable ? (
        <form onSubmit={submit} className="grid gap-6">
          <Row id="user" label="account.user" help="account.userHelp">
            <Input
              id="user"
              value={user}
              maxLength={64}
              autoComplete="username"
              onChange={(e) => setUser(e.target.value)}
            />
          </Row>
          <NewPassword
            password={password}
            again={again}
            onPassword={setPassword}
            onAgain={setAgain}
            minimum={account.min_password_length}
            label="account.password"
          />
          <Problem error={set.error} />
          <div>
            <Button type="submit" disabled={!ready || set.isPending}>
              {set.isPending ? <Spinner /> : <KeyRound />}
              {t("account.setPassword")}
            </Button>
          </div>
        </form>
      ) : (
        <Note>{t("account.readOnly")}</Note>
      )}
    </Group>
  );
}

/** SignInGroup changes the name and password of a guarded studio. */
function SignInGroup({ account }: { account: Account }) {
  const { t } = useI18n();
  const set = useSetCredentials();
  const current = account.user ?? "";
  const [user, setUser] = useState(current);
  const [password, setPassword] = useState("");
  const [again, setAgain] = useState("");
  const [confirm, setConfirm] = useState("");
  useEffect(() => setUser(current), [current]);

  if (!account.editable) {
    return (
      <Group id="sign-in" title={t("account.groupSignIn")}>
        <Row id="user" label="account.user">
          <Input id="user" value={current} disabled />
        </Row>
        <Note>{t(account.source === "environment" ? "account.fromEnvironment" : "account.readOnly")}</Note>
      </Group>
    );
  }

  const renamed = user.trim() !== "" && user.trim() !== current;
  const ready =
    (renamed || password !== "") &&
    user.trim() !== "" &&
    newPasswordReady(password, again, account.min_password_length, true) &&
    confirm !== "";

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!ready || set.isPending) return;
    set.mutate(
      {
        current_password: confirm,
        user: renamed ? user.trim() : undefined,
        password: password || undefined,
      },
      {
        onSuccess: () => {
          setPassword("");
          setAgain("");
          setConfirm("");
          toast.success(t("account.signInSaved"));
        },
      },
    );
  };

  return (
    <Group id="sign-in" title={t("account.groupSignIn")}>
      <form onSubmit={submit} className="grid gap-6">
        <Row id="user" label="account.user" help="account.userHelp">
          <Input
            id="user"
            value={user}
            maxLength={64}
            autoComplete="username"
            onChange={(e) => setUser(e.target.value)}
          />
        </Row>
        <NewPassword
          password={password}
          again={again}
          onPassword={setPassword}
          onAgain={setAgain}
          minimum={account.min_password_length}
          label="account.newPassword"
          help="account.newPasswordHelp"
        />
        <Row id="current-password" label="account.currentPassword" help="account.currentPasswordHelp">
          <PasswordInput
            id="current-password"
            value={confirm}
            autoComplete="current-password"
            onChange={(e) => setConfirm(e.target.value)}
          />
        </Row>
        <Problem error={set.error} />
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <Button type="submit" disabled={!ready || set.isPending}>
            {set.isPending && <Spinner />}
            {t("account.saveSignIn")}
          </Button>
          <p className="text-sm text-muted-foreground">{t("account.signInNote")}</p>
        </div>
      </form>
    </Group>
  );
}

/** SessionsGroup is this browser's session, and the way to end the others. */
function SessionsGroup({ account }: { account: Account }) {
  const { t, date } = useI18n();
  const problem = useProblem();
  const end = useEndOtherSessions();
  const [ending, setEnding] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const session = account.session;
  if (!session) return null;
  const fixed = account.source === "environment";

  return (
    <Group title={t("account.groupSessions")}>
      <div className="flex items-start gap-3 rounded-md border p-4">
        <MonitorSmartphone className="mt-0.5 size-5 shrink-0 text-muted-foreground" />
        <div className="grid gap-1 text-sm">
          <p>{t("account.sessionUntil", { time: date(session.expires_at, "long") })}</p>
          <p className="text-muted-foreground">
            {t(session.remembered ? "account.sessionRemembered" : "account.sessionWindow")}
          </p>
        </div>
      </div>
      <div className="grid gap-2">
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" disabled={fixed || !account.editable} onClick={() => setEnding(true)}>
            {t("account.endOthers")}
          </Button>
          <Button variant="outline" onClick={() => setLeaving(true)}>
            <LogOut />
            {t("session.signOut")}
          </Button>
        </div>
        {fixed && <p className="text-sm text-muted-foreground">{t("account.endOthersFixed")}</p>}
      </div>

      <ConfirmDialog
        open={ending}
        onOpenChange={setEnding}
        title={t("account.endOthers")}
        desc={t("account.endOthersNote")}
        confirmText={t("account.endOthers")}
        isLoading={end.isPending}
        className="sm:max-w-sm"
        handleConfirm={() =>
          end.mutate(undefined, {
            onSuccess: () => {
              setEnding(false);
              toast.success(t("account.endOthersDone"));
            },
            onError: (error) => {
              setEnding(false);
              toast.error(error instanceof ApiError ? problem(error.code, error.message).title : String(error));
            },
          })
        }
      />
      <SignOutDialog open={leaving} onOpenChange={setLeaving} />
    </Group>
  );
}

/** RemoveGroup takes the password away, which only a local server offers. */
function RemoveGroup() {
  const { t } = useI18n();
  const remove = useRemoveCredentials();
  const [open, setOpen] = useState(false);
  const [confirm, setConfirm] = useState("");

  const close = (next: boolean) => {
    setOpen(next);
    if (!next) {
      setConfirm("");
      remove.reset();
    }
  };

  return (
    <Group title={t("account.groupRemove")}>
      <div className="flex flex-wrap items-center justify-between gap-4 rounded-md border border-destructive/30 p-4">
        <p className="max-w-md text-sm text-muted-foreground">{t("account.removeNote")}</p>
        <Button variant="outline" className="text-destructive hover:text-destructive" onClick={() => setOpen(true)}>
          <Trash2 />
          {t("account.remove")}
        </Button>
      </div>

      <ConfirmDialog
        open={open}
        onOpenChange={close}
        title={t("account.remove")}
        desc={t("account.removeConfirm")}
        confirmText={t("account.remove")}
        destructive
        isLoading={remove.isPending}
        disabled={confirm === ""}
        className="sm:max-w-sm"
        form="remove-password"
      >
        <form
          id="remove-password"
          className="grid gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (confirm === "" || remove.isPending) return;
            remove.mutate(confirm, {
              onSuccess: () => {
                close(false);
                toast.success(t("account.removed"));
              },
            });
          }}
        >
          <Label htmlFor="remove-current">{t("account.currentPassword")}</Label>
          <PasswordInput
            id="remove-current"
            value={confirm}
            autoComplete="current-password"
            autoFocus
            onChange={(e) => setConfirm(e.target.value)}
          />
          <Problem error={remove.error} />
        </form>
      </ConfirmDialog>
    </Group>
  );
}

/** Note is a line the form has to say before anything can be typed. */
function Note({ icon, children }: { icon?: ReactNode; children: ReactNode }) {
  return (
    <Alert>
      {icon ?? <Info />}
      <AlertDescription>{children}</AlertDescription>
    </Alert>
  );
}

/** looksLikeEmail is the shape the server accepts: a bare address. */
function looksLikeEmail(value: string) {
  return /^[^\s@<>(),;:"]+@[^\s@<>(),;:"]+$/.test(value);
}

// A refused field of this page, said in the operator's language rather than
// the server's.
const fieldProblems: Record<string, Key> = {
  name: "account.nameInvalid",
  email: "account.emailInvalid",
  user: "account.userInvalid",
};

/**
 * Problem says why a change was refused. The server's own sentence stays
 * under the translated one, except where it would only repeat it.
 */
function Problem({ error }: { error: unknown }) {
  const { t } = useI18n();
  const problem = useProblem();
  if (!error) return null;
  const field = error instanceof ApiError && error.code === "invalid_request" ? error.field : undefined;
  const said =
    field && fieldProblems[field]
      ? { title: t(fieldProblems[field]), detail: undefined }
      : error instanceof ApiError
        ? problem(error.code, error.code === "wrong_password" ? undefined : error.message)
        : { title: String(error), detail: undefined };

  return (
    <Alert variant="destructive">
      <XCircle />
      <AlertTitle>{said.title}</AlertTitle>
      {said.detail && <AlertDescription>{said.detail}</AlertDescription>}
    </Alert>
  );
}
