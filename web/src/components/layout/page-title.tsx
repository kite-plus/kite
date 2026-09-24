/** PageTitle is a page's heading row, as each shadcn-admin page writes it. */
export function PageTitle({
  title,
  description,
  children,
}: {
  title: React.ReactNode
  description?: React.ReactNode
  /** The page's own buttons, on the right. */
  children?: React.ReactNode
}) {
  return (
    <div className='flex flex-wrap items-end justify-between gap-2'>
      <div>
        <h2 className='text-2xl font-bold tracking-tight'>{title}</h2>
        {description && <p className='text-muted-foreground'>{description}</p>}
      </div>
      {children && <div className='flex flex-wrap gap-2'>{children}</div>}
    </div>
  )
}
