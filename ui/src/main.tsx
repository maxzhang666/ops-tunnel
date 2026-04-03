import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ThemeProvider } from 'next-themes'
import { router } from './router'
import { eventSocket } from './lib/ws'
import './lib/i18n'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 10_000,
      retry: 1,
    },
  },
})

// In desktop mode, fetch the actual API port for WebSocket (Wails asset server doesn't support WS)
fetch('/api/v1/version')
  .then((r) => r.json())
  .then((info) => {
    if (info.wsPort) {
      eventSocket.setUrl(`ws://127.0.0.1:${info.wsPort}/ws`)
    }
    eventSocket.connect()
  })
  .catch(() => eventSocket.connect())

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider attribute="class" defaultTheme="light" enableSystem>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </ThemeProvider>
  </StrictMode>,
)
