import { Moon, Sun } from "lucide-react";
import { useTheme } from "@/components/theme-provider";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type Props = {
  className?: string;
};

export function ThemeToggle({ className }: Props) {
  const { theme, setTheme } = useTheme();
  const isDark =
    theme === "dark" ||
    (theme === "system" &&
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);

  return (
    <Button
      variant="ghost"
      size="icon-sm"
      onClick={() => setTheme(isDark ? "light" : "dark")}
      aria-label={isDark ? "切换为浅色模式" : "切换为深色模式"}
      className={cn("rounded-full", className)}
    >
      <Sun className="size-4 scale-100 rotate-0 transition-all dark:scale-0 dark:-rotate-90" />
      <Moon className="absolute size-4 scale-0 rotate-90 transition-all dark:scale-100 dark:rotate-0" />
    </Button>
  );
}
