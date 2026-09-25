import { useEffect, useRef } from 'react'
import { useRouterState } from '@tanstack/react-router'
import LoadingBar, { type LoadingBarRef } from 'react-top-loading-bar'

export function NavigationProgress() {
  const ref = useRef<LoadingBarRef>(null)
  const state = useRouterState()

  useEffect(() => {
    if (state.status === 'pending') {
      ref.current?.continuousStart()
    } else {
      ref.current?.complete()
    }
  }, [state.status])

  return (
    <LoadingBar
      // Kite: the accent blue, like the rest of the studio's highlights.
      color='var(--primary)'
      ref={ref}
      shadow={true}
      height={2}
    />
  )
}
