import type { ReactNode } from "react";

import { SidebarTrigger } from "@/components/ui/sidebar";

interface Props {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  /** toolbar stays put above the body while the body scrolls. */
  toolbar?: ReactNode;
  children: ReactNode;
}

// Past this width a card row stops reading as a row, so the content is held
// at it and centered; below it nothing changes.
const measure = "mx-auto w-full max-w-[1440px]";

/** Page is one screen inside the shell: a fixed header over a scrolling body. */
export function Page({ title, description, actions, toolbar, children }: Props) {
  return (
    <div className="flex h-svh min-w-0 flex-col">
      <header className="px-4 pt-5 pb-3.5 sm:px-7">
        {/* The actions drop under a title they would otherwise squeeze. */}
        <div className={`${measure} flex flex-wrap items-start justify-between gap-x-4 gap-y-3`}>
          <div className="flex min-w-0 flex-1 basis-48 items-start gap-2">
            <SidebarTrigger className="md:hidden" />
            <div className="min-w-0">
              <h1 className="truncate text-xl font-semibold tracking-tight">{title}</h1>
              {description && (
                <div className="mt-1 text-sm text-muted-foreground">{description}</div>
              )}
            </div>
          </div>
          {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
        </div>
      </header>
      {toolbar && (
        <div className="px-4 pb-3.5 sm:px-7">
          <div className={measure}>{toolbar}</div>
        </div>
      )}
      <div className="min-h-0 flex-1 overflow-auto px-4 pb-6 sm:px-7">
        <div className={measure}>{children}</div>
      </div>
    </div>
  );
}
