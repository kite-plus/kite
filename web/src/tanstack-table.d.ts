import '@tanstack/react-table'

declare module '@tanstack/react-table' {
  // TData and TValue must match the library's declaration to merge with it.
  interface ColumnMeta<TData, TValue> {
    className?: string // apply to both th and td
    tdClassName?: string
    thClassName?: string
    /** Kite: the column's name in the view menu, in the operator's language. */
    title?: string
  }
}
