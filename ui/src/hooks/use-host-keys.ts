import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { TUNNEL_KEYS } from '@/hooks/use-tunnels'
import type { KnownHost } from '@/types/api'

const KEYS = {
  all: ['host-keys'] as const,
}

// Trust changes invalidate any TunnelStatus.hostKey a card is still rendering.
// allStatuses is the prefix of every per-tunnel status key, so this refreshes
// them all without needing a tunnel id — which neither mutation has in scope.
function invalidateAffected(qc: QueryClient) {
  qc.invalidateQueries({ queryKey: KEYS.all })
  qc.invalidateQueries({ queryKey: TUNNEL_KEYS.all })
  qc.invalidateQueries({ queryKey: TUNNEL_KEYS.allStatuses })
}

export function useKnownHosts() {
  return useQuery({
    queryKey: KEYS.all,
    queryFn: () => api.get<{ hosts: KnownHost[] }>('/host-keys'),
  })
}

export function useTrustHostKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: { hostPort: string; fingerprint: string }) =>
      api.put<{ ok: boolean; fingerprint: string; keyType: string }>('/host-keys', data),
    onSuccess: () => invalidateAffected(qc),
  })
}

export function useRevokeHostKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (hostPort: string) =>
      api.del(`/host-keys?hostPort=${encodeURIComponent(hostPort)}`),
    onSuccess: () => invalidateAffected(qc),
  })
}
