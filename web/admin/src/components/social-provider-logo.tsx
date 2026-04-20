import { useState } from "react";
import { Icon } from "@iconify/react";

import { cn } from "@/lib/utils";

type SocialProviderKey = "wechat" | "github" | "google";

type SvglRoute =
  | string
  | {
      light: string;
      dark: string;
    };

const SOCIAL_PROVIDER_META: Record<
  SocialProviderKey,
  {
    label: string;
    colorClassName?: string;
    source:
      | { kind: "svgl"; route: SvglRoute }
      | { kind: "iconify"; icon: string };
    fallback: string;
  }
> = {
  wechat: {
    label: "微信",
    colorClassName: "text-[#07C160]",
    source: { kind: "iconify", icon: "simple-icons:wechat" },
    fallback: "simple-icons:wechat",
  },
  github: {
    label: "GitHub",
    source: {
      kind: "svgl",
      route: {
        light: "https://svgl.app/library/github_light.svg",
        dark: "https://svgl.app/library/github_dark.svg",
      },
    },
    fallback: "simple-icons:github",
  },
  google: {
    label: "Google",
    source: {
      kind: "svgl",
      route: "https://svgl.app/library/google.svg",
    },
    fallback: "simple-icons:google",
  },
};

function SvglLogo({
  route,
  label,
  className,
  onError,
}: {
  route: SvglRoute;
  label: string;
  className?: string;
  onError: () => void;
}) {
  if (typeof route === "string") {
    return (
      <img
        src={route}
        alt={label}
        loading="lazy"
        draggable={false}
        onError={onError}
        className={cn("size-full object-contain", className)}
      />
    );
  }

  return (
    <>
      <img
        src={route.light}
        alt={label}
        loading="lazy"
        draggable={false}
        onError={onError}
        className={cn("size-full object-contain dark:hidden", className)}
      />
      <img
        src={route.dark}
        alt={label}
        loading="lazy"
        draggable={false}
        onError={onError}
        className={cn("hidden size-full object-contain dark:block", className)}
      />
    </>
  );
}

interface SocialProviderLogoProps {
  provider: string;
  size?: number;
  className?: string;
  rounded?: string;
  appearance?: "badge" | "plain";
}

export function SocialProviderLogo({
  provider,
  size = 20,
  className,
  rounded = "rounded-md",
  appearance = "badge",
}: SocialProviderLogoProps) {
  const key = (provider as SocialProviderKey) || "github";
  const meta = SOCIAL_PROVIDER_META[key] ?? SOCIAL_PROVIDER_META.github;
  const [failed, setFailed] = useState(false);

  const iconNode =
    !failed && meta.source.kind === "svgl" ? (
      <SvglLogo
        route={meta.source.route}
        label={meta.label}
        onError={() => setFailed(true)}
        className={cn(appearance === "badge" ? "p-1" : "p-0")}
      />
    ) : (
      <Icon
        icon={meta.source.kind === "iconify" ? meta.source.icon : meta.fallback}
        className={cn(
          appearance === "badge" ? "size-[70%]" : "size-full",
          meta.colorClassName
        )}
      />
    );

  if (appearance === "plain") {
    return (
      <div
        className={cn(
          "inline-flex shrink-0 items-center justify-center overflow-hidden",
          className
        )}
        style={{ width: size, height: size }}
        aria-label={meta.label}
        title={meta.label}
      >
        {iconNode}
      </div>
    );
  }

  return (
    <div
      className={cn(
        "inline-flex shrink-0 items-center justify-center overflow-hidden border bg-background/80 text-foreground",
        rounded,
        className
      )}
      style={{ width: size, height: size }}
      aria-label={meta.label}
      title={meta.label}
    >
      {iconNode}
    </div>
  );
}
