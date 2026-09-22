import type { ReactNode } from "react";
import { cn } from "cn";

import { SidebarTrigger } from "@/components/ui/sidebar";

interface Props {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  /** toolbar stays put above the body while the body scrolls. */
  toolbar?: ReactNode;
  /** hero is the dashboard's larger heading, its actions on the subtitle's line. */
  hero?: boolean;
  /**
   * flush drops the room kept above the body. It is there because a card's
   * edge is a ring drawn outside its box, which a scrolling container clips
   * when the card sits flush with its top; a body that opens on a real border
   * does not need it.
   */
  flush?: boolean;
  children: ReactNode;
}

/** Page is one screen inside the shell: a fixed header over a scrolling body. */
export function Page({ title, description, actions, toolbar, hero, flush, children }: Props) {
  return (
    <div className="flex h-svh min-w-0 flex-col">
      <header
        className={cn(
          "px-4 sm:px-7",
          hero ? "pt-[26px] pb-[18px]" : toolbar ? "pt-[22px] pb-[14px]" : "pt-[22px] pb-[18px]",
        )}
      >
        {/* The actions drop under a title they would otherwise squeeze. */}
        <div
          className={cn(
            "flex flex-wrap justify-between gap-x-4 gap-y-3",
            hero ? "items-end" : "items-start",
          )}
        >
          <div className="flex min-w-0 flex-1 basis-48 items-start gap-2">
            <SidebarTrigger className="md:hidden" />
            <div className="min-w-0">
              <h1
                className={cn(
                  "truncate",
                  hero
                    ? "text-[21px] font-[650] tracking-[-0.3px]"
                    : "text-[19px] font-semibold tracking-[-0.2px]",
                )}
              >
                {title}
              </h1>
              {description && (
                <div className={cn("text-[13px] text-muted-foreground", hero ? "mt-1.5" : "mt-1")}>
                  {description}
                </div>
              )}
            </div>
          </div>
          {actions && (
            <div className="flex max-w-full shrink-0 flex-wrap items-center gap-2">{actions}</div>
          )}
        </div>
      </header>
      {toolbar && <div className="px-4 pb-[14px] sm:px-7">{toolbar}</div>}
      <div
        className={cn(
          "min-h-0 flex-1 overflow-auto px-4 sm:px-7",
          flush ? "pt-0" : "pt-1",
          hero ? "pb-[26px]" : "pb-6",
        )}
      >
        {children}
      </div>
    </div>
  );
}
