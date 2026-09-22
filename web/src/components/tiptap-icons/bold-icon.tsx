import { memo } from "react"

type SvgProps = React.ComponentPropsWithoutRef<"svg">

// Kite: the admin design's glyph, drawn as a stroke like the rest of its icons.
export const BoldIcon = memo(({ className, ...props }: SvgProps) => {
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
      <path d="M7 5h6a3.5 3.5 0 0 1 0 7H7Z" />
      <path d="M7 12h7a3.5 3.5 0 0 1 0 7H7Z" />
    </svg>
  )
})

BoldIcon.displayName = "BoldIcon"
