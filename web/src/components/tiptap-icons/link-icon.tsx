import { memo } from "react"

type SvgProps = React.ComponentPropsWithoutRef<"svg">

// Kite: the admin design's glyph, drawn as a stroke like the rest of its icons.
export const LinkIcon = memo(({ className, ...props }: SvgProps) => {
  return (
    <svg
      width="24"
      height="24"
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path d="m9 15 6-6" />
      <path d="m11 6 1.5-1.5a4 4 0 0 1 5.7 5.7L16.5 12" />
      <path d="m13 18-1.5 1.5a4 4 0 0 1-5.7-5.7L7.5 12" />
    </svg>
  )
})

LinkIcon.displayName = "LinkIcon"
