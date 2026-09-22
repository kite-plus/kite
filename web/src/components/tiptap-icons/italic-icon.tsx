import { memo } from "react"

type SvgProps = React.ComponentPropsWithoutRef<"svg">

// Kite: the admin design's glyph, drawn as a stroke like the rest of its icons.
export const ItalicIcon = memo(({ className, ...props }: SvgProps) => {
  return (
    <svg
      width="24"
      height="24"
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path d="M19 5h-8" />
      <path d="M13 19H5" />
      <path d="m15 5-6 14" />
    </svg>
  )
})

ItalicIcon.displayName = "ItalicIcon"
