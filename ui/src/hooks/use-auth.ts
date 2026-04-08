import { createContext, useContext, useCallback } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { AuthCheckResponse } from '@/types/api'

interface AuthState {
  required: boolean
  authenticated: boolean
  checking: boolean
}

const AuthContext = createContext<AuthState>({
  required: false,
  authenticated: false,
  checking: true,
})

export function useAuthQuery() {
  return useQuery<AuthCheckResponse>({
    queryKey: ['auth', 'check'],
    queryFn: () => api.authCheck(),
    retry: false,
    staleTime: 30_000,
  })
}

export function useAuth(): AuthState {
  return useContext(AuthContext)
}

export function useLogout() {
  const queryClient = useQueryClient()
  return useCallback(async () => {
    await api.logout()
    queryClient.setQueryData(['auth', 'check'], { required: true, authenticated: false })
    window.location.href = '/login'
  }, [queryClient])
}

export { AuthContext }
