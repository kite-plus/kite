export { cn } from "cn";

/**
 * Initials from a display name: first character of the first word + first
 * character of the last word. One word only: first two characters. Empty: `?`.
 * A name in Chinese characters shows its last two characters instead: the
 * given name, the way chat apps in China draw an avatar.
 */
export function getDisplayNameInitials(displayName: string): string {
  const parts = displayName.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) {
    const chars = [...parts[0]];
    if (/\p{Script=Han}/u.test(parts[0])) return chars.slice(-2).join("");
    return chars.slice(0, 2).join("").toUpperCase();
  }
  const first = parts[0][0] ?? "";
  const last = parts[parts.length - 1]?.[0] ?? "";
  return (first + last).toUpperCase();
}
