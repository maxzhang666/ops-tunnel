import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent } from '@/components/ui/card'
import type { Mapping, TunnelMode, Socks5Auth } from '@/types/api'

interface MappingEditorProps {
  mode: TunnelMode
  value: Mapping[]
  onChange: (mappings: Mapping[]) => void
}

function emptyMapping(mode: TunnelMode): Mapping {
  const base: Mapping = { id: '', listen: { host: '127.0.0.1', port: 0 } }
  if (mode === 'local' || mode === 'remote') {
    base.connect = { host: '', port: 0 }
  }
  if (mode === 'dynamic') {
    base.socks5 = { auth: 'none' }
  }
  return base
}

function updateMapping(mappings: Mapping[], index: number, partial: Partial<Mapping>): Mapping[] {
  return mappings.map((m, i) => (i === index ? { ...m, ...partial } : m))
}

export function MappingEditor({ mode, value, onChange }: MappingEditorProps) {
  const { t } = useTranslation()
  const addMapping = () => onChange([...value, emptyMapping(mode)])
  const removeMapping = (index: number) => onChange(value.filter((_, i) => i !== index))

  return (
    <div className="space-y-3">
      {value.map((mapping, idx) => (
        <Card key={idx} className='pt-1'>
          <CardContent className="space-y-3 pt-0">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-muted-foreground">{t('mapping.label', { index: idx + 1 })}</span>
              {value.length > 1 && (
                <Button type="button" variant="ghost" size="icon" className="h-7 w-7 text-destructive hover:text-destructive" onClick={() => removeMapping(idx)}>
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              )}
            </div>

            {(mode === 'local' || mode === 'remote') ? (
              <>
                {/* Listen + Connect on same row */}
                <div className="grid grid-cols-6 gap-2">
                  <div className="col-span-2 space-y-1">
                    <Label className="text-xs">{t('mapping.listenHost')}</Label>
                    <Input value={mapping.listen.host} onChange={(e) => onChange(updateMapping(value, idx, { listen: { ...mapping.listen, host: e.target.value } }))} placeholder="127.0.0.1" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">{t('mapping.listenPort')}</Label>
                    <Input type="number" value={mapping.listen.port || ''} onChange={(e) => onChange(updateMapping(value, idx, { listen: { ...mapping.listen, port: Number(e.target.value) } }))} />
                  </div>
                  <div className="col-span-2 space-y-1">
                    <Label className="text-xs">{t('mapping.connectHost')}</Label>
                    <Input value={mapping.connect?.host ?? ''} onChange={(e) => onChange(updateMapping(value, idx, { connect: { host: e.target.value, port: mapping.connect?.port ?? 0 } }))} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">{t('mapping.connectPort')}</Label>
                    <Input type="number" value={mapping.connect?.port || ''} onChange={(e) => onChange(updateMapping(value, idx, { connect: { host: mapping.connect?.host ?? '', port: Number(e.target.value) } }))} />
                  </div>
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">{t('mapping.notes')}</Label>
                  <Input value={mapping.notes ?? ''} onChange={(e) => onChange(updateMapping(value, idx, { notes: e.target.value }))} placeholder={t('mapping.notesPlaceholder')} />
                </div>
              </>
            ) : (
              <>
                {/* Dynamic: listen only */}
                <div className="grid grid-cols-3 gap-3">
                  <div className="col-span-2 space-y-1">
                    <Label className="text-xs">{t('mapping.listenHost')}</Label>
                    <Input value={mapping.listen.host} onChange={(e) => onChange(updateMapping(value, idx, { listen: { ...mapping.listen, host: e.target.value } }))} placeholder="127.0.0.1" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">{t('mapping.listenPort')}</Label>
                    <Input type="number" value={mapping.listen.port || ''} onChange={(e) => onChange(updateMapping(value, idx, { listen: { ...mapping.listen, port: Number(e.target.value) } }))} />
                  </div>
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">{t('mapping.socks5Auth')}</Label>
                  <select className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs" value={mapping.socks5?.auth ?? 'none'} onChange={(e) => onChange(updateMapping(value, idx, { socks5: { ...mapping.socks5!, auth: e.target.value as Socks5Auth } }))}>
                    <option value="none">{t('mapping.socks5AuthNone')}</option>
                    <option value="userpass">{t('mapping.socks5AuthUserpass')}</option>
                  </select>
                </div>
                {mapping.socks5?.auth === 'userpass' && (
                  <div className="grid grid-cols-2 gap-3">
                    <div className="space-y-1">
                      <Label className="text-xs">{t('common.username')}</Label>
                      <Input value={mapping.socks5?.username ?? ''} onChange={(e) => onChange(updateMapping(value, idx, { socks5: { ...mapping.socks5!, username: e.target.value } }))} />
                    </div>
                    <div className="space-y-1">
                      <Label className="text-xs">{t('common.password')}</Label>
                      <Input type="password" value={mapping.socks5?.password ?? ''} onChange={(e) => onChange(updateMapping(value, idx, { socks5: { ...mapping.socks5!, password: e.target.value } }))} />
                    </div>
                  </div>
                )}
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label className="text-xs">{t('mapping.allowCidrs')}</Label>
                    <Input value={mapping.socks5?.allowCIDRs?.join(', ') ?? ''} onChange={(e) => onChange(updateMapping(value, idx, { socks5: { ...mapping.socks5!, allowCIDRs: e.target.value ? e.target.value.split(',').map((s) => s.trim()) : [] } }))} placeholder={t('mapping.allowCidrsPlaceholder')} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">{t('mapping.denyCidrs')}</Label>
                    <Input value={mapping.socks5?.denyCIDRs?.join(', ') ?? ''} onChange={(e) => onChange(updateMapping(value, idx, { socks5: { ...mapping.socks5!, denyCIDRs: e.target.value ? e.target.value.split(',').map((s) => s.trim()) : [] } }))} />
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      ))}

      <Button type="button" variant="outline" size="sm" onClick={addMapping}>
        <Plus className="mr-1 h-3.5 w-3.5" /> {t('mapping.addMapping')}
      </Button>
    </div>
  )
}
