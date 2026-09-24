import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { toast } from 'sonner'
import { ApiError } from '@/api/client'
import { useI18n, useProblem } from '@/i18n'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

/** RefreshButton asks again for everything on the page and says how it went. */
export function RefreshButton() {
  const { t } = useI18n()
  const problem = useProblem()
  const queryClient = useQueryClient()
  const [refreshing, setRefreshing] = useState(false)

  const refresh = async () => {
    if (refreshing) return
    setRefreshing(true)
    const started = performance.now()
    let failure: unknown = null
    try {
      await queryClient.invalidateQueries(undefined, { throwOnError: true })
    } catch (cause) {
      failure = cause
    }
    // animate-spin turns once a second. Finishing the turn keeps the icon
    // from jumping back, and shows a spin even when the answer is instant.
    const turn = 1000 - ((performance.now() - started) % 1000)
    await new Promise((resolve) => setTimeout(resolve, turn))
    setRefreshing(false)
    if (!failure) toast.success(t('dashboard.refreshed'))
    else if (failure instanceof ApiError) toast.error(problem(failure.code, failure.message).title)
    else toast.error(t('dashboard.refreshFailed'))
  }

  return (
    <Button variant='outline' aria-busy={refreshing} onClick={() => void refresh()}>
      <RefreshCw className={cn(refreshing && 'animate-spin')} />
      {t('dashboard.refresh')}
    </Button>
  )
}
