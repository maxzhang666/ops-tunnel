import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'

interface StatsResponse {
  memAlloc: number
  memSys: number
  goroutines: number
}

export function useStats() {
  return useQuery({
    queryKey: ['stats'],
    queryFn: () => api.get<StatsResponse>('/stats'),
    refetchInterval: 5000,
  })
}
