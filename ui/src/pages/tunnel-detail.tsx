import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useTunnel, useTunnelStatus, TUNNEL_KEYS } from '@/hooks/use-tunnels'
import { useWsEvent } from '@/hooks/use-ws-events'
import { DetailHeader } from '@/components/tunnel/detail-header'
import { DetailOverview } from '@/components/tunnel/detail-overview'
import { DetailMappings } from '@/components/tunnel/detail-mappings'
import { DetailConfig } from '@/components/tunnel/detail-config'

export default function TunnelDetailPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { data: tunnel, isLoading } = useTunnel(id!)
  const { data: status } = useTunnelStatus(id!)
  const queryClient = useQueryClient()

  useWsEvent(useCallback((event) => {
    if (event.tunnelId === id && event.type === 'tunnel.stateChanged') {
      queryClient.invalidateQueries({ queryKey: TUNNEL_KEYS.status(id!) })
      queryClient.invalidateQueries({ queryKey: TUNNEL_KEYS.one(id!) })
    }
  }, [id, queryClient]))

  if (isLoading) {
    return <div className="py-8 text-center text-muted-foreground">{t('common.loading')}</div>
  }

  if (!tunnel) {
    return <div className="py-8 text-center text-muted-foreground">{t('tunnel.tunnelNotFound')}</div>
  }

  return (
    <div className="flex h-full flex-col">
      <DetailHeader tunnel={tunnel} status={status} />

      <Tabs defaultValue="overview" className="flex min-h-0 flex-1 flex-col">
        <TabsList>
          <TabsTrigger value="overview">{t('tunnel.tabOverview')}</TabsTrigger>
          <TabsTrigger value="mappings">{t('tunnel.tabMappings')}</TabsTrigger>
          <TabsTrigger value="config">{t('tunnel.tabConfig')}</TabsTrigger>
        </TabsList>
        <TabsContent value="overview" className="mt-4 min-h-0 flex-1">
          <DetailOverview tunnel={tunnel} status={status} />
        </TabsContent>
        <TabsContent value="mappings" className="mt-4">
          <DetailMappings tunnel={tunnel} status={status} />
        </TabsContent>
        <TabsContent value="config" className="mt-4">
          <DetailConfig tunnel={tunnel} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
