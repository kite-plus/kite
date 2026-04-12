import { LogOut, User, Languages } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { useI18n, localeLabels, type Locale } from "@/i18n";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export function Header() {
  const { user, logout } = useAuth();
  const { t, locale, setLocale } = useI18n();

  return (
    <header className="flex h-14 items-center justify-end gap-2 border-b px-6">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon-sm">
            <Languages className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {(Object.keys(localeLabels) as Locale[]).map((l) => (
            <DropdownMenuItem
              key={l}
              onClick={() => setLocale(l)}
              className={locale === l ? "font-medium" : ""}
            >
              {localeLabels[l]}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      <DropdownMenu>
        <DropdownMenuTrigger className="flex items-center gap-2 rounded-md px-2 py-1 text-sm outline-none hover:bg-accent">
          <Avatar>
            <AvatarFallback>
              {user?.username?.charAt(0).toUpperCase() ?? "U"}
            </AvatarFallback>
          </Avatar>
          <span className="hidden sm:inline">{user?.username}</span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuLabel className="font-normal">
            <div className="text-sm font-medium">{user?.username}</div>
            <div className="text-xs text-muted-foreground">{user?.role}</div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem>
            <User className="size-4" />
            {t("auth.profile")}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" onClick={logout}>
            <LogOut className="size-4" />
            {t("auth.logout")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  );
}
