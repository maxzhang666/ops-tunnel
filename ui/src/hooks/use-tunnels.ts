import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Tunnel, TunnelStatus } from '@/types/api'

const KEYS = {
  all: ['tunnels'] as const,
  one: (id: string) => ['tunnels', id] as const,
  status: (id: string) => ['tunnel-status', id] as const,
  allStatuses: ['tunnel-status'] as const,
}

export function useTunnels() {
  return useQuery({
    queryKey: KEYS.all,
    queryFn: () => api.get<Tunnel[]>('/tunnels'),
  })
}

export function useTunnel(id: string) {
  return useQuery({
    queryKey: KEYS.one(id),
    queryFn: () => api.get<Tunnel>(`/tunnels/${id}`),
    enabled: !!id,
  })
}

export function useTunnelStatus(id: string, enabled = true) {
  return useQuery({
    queryKey: KEYS.status(id),
    queryFn: () => api.get<TunnelStatus>(`/tunnels/${id}/status`),
    enabled: !!id && enabled,
    refetchInterval: (query) => {
      const state = query.state.data?.state
      if (state === 'running' || state === 'starting' || state === 'degraded') {
        return 3000
      }
      return false
    },
  })
}

export function useCreateTunnel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Tunnel>) =>
      api.post<Tunnel>('/tunnels', data),
    onSuccess: () => qc.invalidateQueries({ queryKey: KEYS.all }),
  })
}

export function useUpdateTunnel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Tunnel> }) =>
      api.put<Tunnel>(`/tunnels/${id}`, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: KEYS.all }),
  })
}

export function useDeleteTunnel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.del(`/tunnels/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: KEYS.all }),
  })
}

export function useStartTunnel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.post<void>(`/tunnels/${id}/start`, {}),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: KEYS.all })
      qc.invalidateQueries({ queryKey: KEYS.status(id) })
    },
  })
}

export function useStopTunnel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.post<void>(`/tunnels/${id}/stop`, {}),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: KEYS.all })
      qc.invalidateQueries({ queryKey: KEYS.status(id) })
    },
  })
}

export function useRestartTunnel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.post<void>(`/tunnels/${id}/restart`, {}),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: KEYS.all })
      qc.invalidateQueries({ queryKey: KEYS.status(id) })
    },
  })
}

export { KEYS as TUNNEL_KEYS }
