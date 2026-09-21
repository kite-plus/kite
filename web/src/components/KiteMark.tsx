/**
 * The mark, drawn rather than fetched so it paints with whatever holds it.
 *
 * The stroke repeats the fill deliberately: that is what rounds the corners of
 * the three shapes. It is the same geometry as docs/assets/logo.svg, kept in
 * step by hand because the alternative -- fetching the file -- would make the
 * rail wait on a request to draw its own header.
 */
export function KiteMark({ className = "size-6" }: { className?: string }) {
  return (
    <svg viewBox="0 0 64 64" className={`shrink-0 ${className}`} aria-hidden>
      <g
        fill="#4A77D6"
        stroke="#4A77D6"
        strokeWidth="5"
        strokeLinejoin="round"
        transform="translate(32 32) scale(0.82) translate(-32 -32)"
      >
        <path d="M10 14.5 L27 21 L27 30 L10 23.5 Z" />
        <path d="M10 32 L27 38.5 L27 49 L10 42.5 Z" />
        <path d="M37 21 L54 14.5 L54 42.5 L37 49 Z" />
      </g>
    </svg>
  );
}
