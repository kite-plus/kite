import type { ReactNode } from "react";
import { Check } from "lucide-react";

import { useI18n, type Key } from "@/i18n";
import { cn } from "@/lib/utils";
import { Label } from "@/components/ui/label";
import { PasswordInput } from "@/components/password-input";

/** passwordLength counts characters as the server does, not UTF-16 units. */
export function passwordLength(password: string): number {
  return [...password].length;
}

/**
 * newPasswordReady reports whether a new password, typed twice, can be sent.
 * An optional one may also be left empty in both boxes.
 */
export function newPasswordReady(password: string, again: string, minimum: number, optional = false) {
  if (optional && password === "" && again === "") return true;
  return passwordLength(password) >= minimum && again === password;
}

/**
 * NewPassword is a new password typed twice, each box with a line under it
 * saying how far the answer is from being usable.
 */
export function NewPassword({
  password,
  again,
  onPassword,
  onAgain,
  minimum,
  label = "setup.password",
  help,
  autoFocus,
}: {
  password: string;
  again: string;
  onPassword: (value: string) => void;
  onAgain: (value: string) => void;
  minimum: number;
  label?: Key;
  // Shown while the box is empty, in place of the length rule.
  help?: Key;
  autoFocus?: boolean;
}) {
  const { t } = useI18n();
  const length = passwordLength(password);
  const longEnough = length >= minimum;
  const matches = again.length > 0 && again === password;
  const mismatch = again.length > 0 && again !== password;

  return (
    <>
      <div className="grid gap-2">
        <Label htmlFor="new-password">{t(label)}</Label>
        <PasswordInput
          id="new-password"
          value={password}
          onChange={(e) => onPassword(e.target.value)}
          autoComplete="new-password"
          autoFocus={autoFocus}
        />
        {length > 0 && !longEnough ? (
          <Hint tone="wrong">{t("setup.passwordNeeds", { count: minimum - length })}</Hint>
        ) : longEnough ? (
          <Hint tone="right">{t("setup.passwordOK")}</Hint>
        ) : (
          <Hint>{help ? t(help, { n: minimum }) : t("setup.passwordHelp", { n: minimum })}</Hint>
        )}
      </div>
      <div className="grid gap-2">
        <Label htmlFor="new-password-again">{t("setup.again")}</Label>
        <PasswordInput
          id="new-password-again"
          value={again}
          onChange={(e) => onAgain(e.target.value)}
          autoComplete="new-password"
        />
        {mismatch ? (
          <Hint tone="wrong">{t("setup.mismatch")}</Hint>
        ) : matches ? (
          <Hint tone="right">{t("setup.match")}</Hint>
        ) : null}
      </div>
    </>
  );
}

/**
 * One line of feedback under a field.
 *
 * It says when an answer is right as well as when it is wrong. A form that
 * only speaks up to complain leaves somebody staring at a password they
 * cannot see, wondering whether the two boxes agree.
 */
export function Hint({ tone, children }: { tone?: "right" | "wrong"; children: ReactNode }) {
  return (
    // Polite rather than assertive: this changes on every keystroke, and a
    // screen reader interrupting each one would be unusable.
    <p
      aria-live="polite"
      className={cn(
        "text-sm text-muted-foreground",
        tone === "wrong" && "text-destructive",
        tone === "right" && "text-success",
      )}
    >
      {tone === "right" && <Check aria-hidden className="me-1 inline size-3.5 align-[-0.15em]" />}
      {children}
    </p>
  );
}
