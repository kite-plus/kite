import type { ReactNode } from "react";

/**
 * The icons the design draws, path for path.
 *
 * They are close to lucide's but not the same: thinner (1.8), and several
 * glyphs differ. Chrome the design covers uses these so it matches it; the
 * rest of the admin keeps lucide.
 */
interface Props {
  className?: string;
  strokeWidth?: number;
}

function icon(paths: ReactNode, defaultStroke = 1.8) {
  return function Icon({ className = "size-4", strokeWidth = defaultStroke }: Props) {
    return (
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeLinejoin="round"
        className={`shrink-0 ${className}`}
        aria-hidden
      >
        {paths}
      </svg>
    );
  };
}

export const IconDashboard = icon(
  <>
    <rect x="3" y="3" width="7" height="9" rx="1.5" />
    <rect x="14" y="3" width="7" height="5" rx="1.5" />
    <rect x="14" y="12" width="7" height="9" rx="1.5" />
    <rect x="3" y="16" width="7" height="5" rx="1.5" />
  </>,
);

export const IconPosts = icon(
  <>
    <path d="M14.5 2.5H6.5a2 2 0 0 0-2 2v15a2 2 0 0 0 2 2h11a2 2 0 0 0 2-2V7.5Z" />
    <path d="M14.5 2.5v5h5" />
    <path d="M9 13h6" />
    <path d="M9 17h6" />
  </>,
);

export const IconPages = icon(
  <>
    <rect x="3" y="3" width="18" height="18" rx="2" />
    <path d="M3 9h18" />
    <path d="M9 9v12" />
  </>,
);

export const IconComments = icon(
  <path d="M21 15a2 2 0 0 1-2 2H7.5L3 21V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2Z" />,
);

export const IconAttachments = icon(
  <path d="M3.5 6.5a2 2 0 0 1 2-2h3.6a2 2 0 0 1 1.6.8l1 1.4a2 2 0 0 0 1.6.8h5.2a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-13a2 2 0 0 1-2-2Z" />,
);

// Not in the design, which has no taxonomy screen; drawn in its hand.
export const IconTags = icon(
  <>
    <path d="M3.5 4.5a1 1 0 0 1 1-1h6.3a2 2 0 0 1 1.4.6l8.2 8.2a2 2 0 0 1 0 2.8l-5.6 5.6a2 2 0 0 1-2.8 0l-8.2-8.2a2 2 0 0 1-.6-1.4Z" />
    <circle cx="8.25" cy="8.25" r="1.25" />
  </>,
);

export const IconTheme = icon(
  <>
    <rect x="3" y="3" width="18" height="8" rx="2" />
    <rect x="3" y="14.5" width="10" height="6.5" rx="2" />
    <rect x="16.5" y="14.5" width="4.5" height="6.5" rx="1.5" />
  </>,
);

export const IconMenus = icon(
  <>
    <path d="M4 7h16" />
    <path d="M4 12h16" />
    <path d="M4 17h10" />
  </>,
);

export const IconPlugins = icon(
  <>
    <rect x="3" y="3" width="7.5" height="7.5" rx="1.5" />
    <rect x="13.5" y="3" width="7.5" height="7.5" rx="1.5" />
    <rect x="3" y="13.5" width="7.5" height="7.5" rx="1.5" />
    <rect x="13.5" y="13.5" width="7.5" height="7.5" rx="1.5" />
  </>,
);

export const IconUsers = icon(
  <>
    <circle cx="9" cy="8" r="3.5" />
    <path d="M3 20c0-3.3 2.7-6 6-6s6 2.7 6 6" />
    <path d="M16 4.6a3.5 3.5 0 0 1 0 6.8" />
    <path d="M17.5 14.6c2 .8 3.5 2.9 3.5 5.4" />
  </>,
);

export const IconSettings = icon(
  <>
    <path d="M20 7h-9" />
    <path d="M14 17H5" />
    <circle cx="17" cy="17" r="3" />
    <circle cx="7" cy="7" r="3" />
  </>,
);

export const IconBackups = icon(
  <>
    <ellipse cx="12" cy="5.5" rx="8" ry="3" />
    <path d="M4 5.5v13c0 1.7 3.6 3 8 3s8-1.3 8-3v-13" />
    <path d="M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3" />
  </>,
);

export const IconSearch = icon(
  <>
    <circle cx="11" cy="11" r="7" />
    <path d="m20.5 20.5-4-4" />
  </>,
  2,
);

export const IconSignOut = icon(
  <>
    <path d="M9 21H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3" />
    <path d="m15.5 16.5 4.5-4.5-4.5-4.5" />
    <path d="M20 12H9.5" />
  </>,
);

export const IconExternal = icon(
  <>
    <path d="M14 4h6v6" />
    <path d="M20 4 11 13" />
    <path d="M19 14v5a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h5" />
  </>,
);

export const IconPlus = icon(
  <>
    <path d="M12 5.5v13" />
    <path d="M5.5 12h13" />
  </>,
);

export const IconUpload = icon(
  <>
    <path d="M12 16V4" />
    <path d="m6 10 6-6 6 6" />
    <path d="M4 20h16" />
  </>,
);

export const IconChevronRight = icon(<path d="m9.5 6 6 6-6 6" />, 2);
export const IconChevronDown = icon(<path d="m6 9.5 6 6 6-6" />, 2);
export const IconChevronLeft = icon(<path d="m14.5 6-6 6 6 6" />, 2);

export function IconDots({ className = "size-4" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={`shrink-0 ${className}`} aria-hidden>
      <circle cx="5.5" cy="12" r="1.3" />
      <circle cx="12" cy="12" r="1.3" />
      <circle cx="18.5" cy="12" r="1.3" />
    </svg>
  );
}

export const IconBold = icon(
  <>
    <path d="M7 5h6a3.5 3.5 0 0 1 0 7H7Z" />
    <path d="M7 12h7a3.5 3.5 0 0 1 0 7H7Z" />
  </>,
  2,
);

export const IconItalic = icon(
  <>
    <path d="M19 5h-8" />
    <path d="M13 19H5" />
    <path d="m15 5-6 14" />
  </>,
  2,
);

export const IconLink = icon(
  <>
    <path d="m9 15 6-6" />
    <path d="m11 6 1.5-1.5a4 4 0 0 1 5.7 5.7L16.5 12" />
    <path d="m13 18-1.5 1.5a4 4 0 0 1-5.7-5.7L7.5 12" />
  </>,
);

export const IconImage = icon(
  <>
    <rect x="3" y="4" width="18" height="16" rx="2" />
    <circle cx="9" cy="10" r="1.5" />
    <path d="m4.5 18 5-5 3 3 3.5-3.5 3.5 3.5" />
  </>,
);

export const IconCode = icon(
  <>
    <path d="m8 8-4 4 4 4" />
    <path d="m16 8 4 4-4 4" />
  </>,
);

export function IconList({ className = "size-4" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={`shrink-0 ${className}`}
      aria-hidden
    >
      <path d="M9 6h12" />
      <path d="M9 12h12" />
      <path d="M9 18h12" />
      <circle cx="4.5" cy="6" r="1" fill="currentColor" stroke="none" />
      <circle cx="4.5" cy="12" r="1" fill="currentColor" stroke="none" />
      <circle cx="4.5" cy="18" r="1" fill="currentColor" stroke="none" />
    </svg>
  );
}
