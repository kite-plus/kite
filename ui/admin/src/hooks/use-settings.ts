import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { mockSettings } from '@/mocks/settings'
import type { AllSettings } from '@/types/settings'

/** 模拟延迟 */
function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/** 内存中的设置副本 */
let settingsStore: AllSettings = structuredClone(mockSettings)

/**
 * Mock 获取全部设置
 */
async function fetchSettings(): Promise<AllSettings> {
  await delay(200)
  return structuredClone(settingsStore)
}

/**
 * Mock 保存设置
 */
async function saveSettings(data: AllSettings): Promise<AllSettings> {
  await delay(500)
  settingsStore = structuredClone(data)
  return settingsStore
}

/**
 * 获取设置 Hook
 */
export function useSettings() {
  return useQuery({
    queryKey: ['settings'],
    queryFn: fetchSettings,
  })
}

/**
 * 保存设置 Hook
 */
export function useSaveSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: saveSettings,
    onSuccess: (data) => {
      queryClient.setQueryData(['settings'], data)
    },
  })
}
