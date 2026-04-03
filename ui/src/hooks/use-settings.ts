import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Settings, VersionInfo } from '@/types/api'

const KEYS = {
  settings: ['settings'] as const,
  version: ['version'] as const,
}

export function useSettings() {
  return useQuery({
    queryKey: KEYS.settings,
    queryFn: () => api.get<Settings>('/settings'),
  })
}

export function useUpdateSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Record<string, unknown>) =>
      api.patch<Settings>('/settings', data),
    onSuccess: (data) => {
      qc.setQueryData(KEYS.settings, data)
    },
  })
}

export function useVersion() {
  return useQuery({
    queryKey: KEYS.version,
    queryFn: () => api.get<VersionInfo>('/version'),
    staleTime: 60 * 60 * 1000,
  })
}

export { KEYS as SETTINGS_KEYS }
