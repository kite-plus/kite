import { memo } from "react"

type SvgProps = React.ComponentPropsWithoutRef<"svg">

// Kite: the admin design's glyph, drawn as a stroke like the rest of its icons.
export const CodeBlockIcon = memo(({ className, ...props }: SvgProps) => {
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
      <path d="m8 8-4 4 4 4" />
      <path d="m16 8 4 4-4 4" />
    </svg>
  )
})

CodeBlockIcon.displayName = "CodeBlockIcon"
