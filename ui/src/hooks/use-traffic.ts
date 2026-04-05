import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'

interface TrafficSample {
  ts: string
  bytesIn: number
  bytesOut: number
}

interface RealtimeResponse {
  samples: TrafficSample[]
  interval: number
}

interface HistoryResponse {
  series: TrafficSample[]
}

export function useRealtimeTraffic() {
  return useQuery({
    queryKey: ['traffic', 'realtime'],
    queryFn: () => api.get<RealtimeResponse>('/traffic/realtime'),
    refetchInterval: 2000,
  })
}

export function useTrafficHistory(range_: string = '24h', step: string = '5m') {
  return useQuery({
    queryKey: ['traffic', 'history', range_, step],
    queryFn: () => api.get<HistoryResponse>(`/traffic/history?range=${range_}&step=${step}`),
    staleTime: 60_000,
  })
}
