import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { SSHConnection, TestResult } from '@/types/api'

const KEYS = {
  all: ['ssh-connections'] as const,
  one: (id: string) => ['ssh-connections', id] as const,
}

export function useSSHConnections() {
  return useQuery({
    queryKey: KEYS.all,
    queryFn: () => api.get<SSHConnection[]>('/ssh-connections'),
  })
}

export function useSSHConnection(id: string) {
  return useQuery({
    queryKey: KEYS.one(id),
    queryFn: () => api.get<SSHConnection>(`/ssh-connections/${id}`),
    enabled: !!id,
  })
}

export function useCreateSSHConnection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<SSHConnection>) =>
      api.post<SSHConnection>('/ssh-connections', data),
    onSuccess: () => qc.invalidateQueries({ queryKey: KEYS.all }),
  })
}

export function useUpdateSSHConnection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<SSHConnection> }) =>
      api.put<SSHConnection>(`/ssh-connections/${id}`, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: KEYS.all }),
  })
}

export function useDeleteSSHConnection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.del(`/ssh-connections/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: KEYS.all }),
  })
}

export function useTestSSHConnection() {
  return useMutation({
    mutationFn: (id: string) =>
      api.post<TestResult>(`/ssh-connections/${id}/test`, {}),
  })
}

export function useTestSSHConnectionDirect() {
  return useMutation({
    mutationFn: (data: Partial<SSHConnection>) =>
      api.post<TestResult>('/ssh-connections/test', data),
  })
}
