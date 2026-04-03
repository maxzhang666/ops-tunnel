import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  DndContext, closestCenter, KeyboardSensor, PointerSensor, useSensor, useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove, SortableContext, sortableKeyboardCoordinates,
  useSortable, verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical, X } from 'lucide-react'
import { Label } from '@/components/ui/label'
import { useSSHConnections } from '@/hooks/use-ssh-connections'
import type { SSHConnection } from '@/types/api'

interface SortableItemProps {
  id: string
  conn: SSHConnection
  onRemove: () => void
}

function SortableItem({ id, conn, onRemove }: SortableItemProps) {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex items-center gap-2 rounded-md border bg-card px-3 py-2"
    >
      <button type="button" className="cursor-grab text-muted-foreground" {...attributes} {...listeners}>
        <GripVertical className="h-4 w-4" />
      </button>
      <div className="min-w-0 flex-1">
        <span className="text-sm font-medium">{conn.name}</span>
        <span className="ml-2 text-xs text-muted-foreground">
          {conn.endpoint.host}:{conn.endpoint.port}
        </span>
      </div>
      <button type="button" className="text-muted-foreground hover:text-destructive" onClick={onRemove}>
        <X className="h-4 w-4" />
      </button>
    </div>
  )
}

interface ChainSelectorProps {
  value: string[]
  onChange: (ids: string[]) => void
}

export function ChainSelector({ value, onChange }: ChainSelectorProps) {
  const { t } = useTranslation()
  const { data: allConns } = useSSHConnections()
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  const connMap = useMemo(() => {
    const m = new Map<string, SSHConnection>()
    allConns?.forEach((c) => m.set(c.id, c))
    return m
  }, [allConns])

  const available = useMemo(
    () => (allConns ?? []).filter((c) => !value.includes(c.id)),
    [allConns, value]
  )

  const handleAdd = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const id = e.target.value
    if (id && !value.includes(id)) {
      onChange([...value, id])
    }
    e.target.value = ''
  }

  const handleRemove = (id: string) => {
    onChange(value.filter((v) => v !== id))
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    if (over && active.id !== over.id) {
      const oldIndex = value.indexOf(active.id as string)
      const newIndex = value.indexOf(over.id as string)
      onChange(arrayMove(value, oldIndex, newIndex))
    }
  }

  return (
    <div className="space-y-3">
      <Label>{t('tunnel.sshChain')}</Label>

      {value.length > 0 && (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={value} strategy={verticalListSortingStrategy}>
            <div className="space-y-2">
              {value.map((id) => {
                const conn = connMap.get(id)
                if (!conn) return null
                return (
                  <SortableItem key={id} id={id} conn={conn} onRemove={() => handleRemove(id)} />
                )
              })}
            </div>
          </SortableContext>
        </DndContext>
      )}

      {value.length === 0 && (
        <p className="text-sm text-muted-foreground">{t('tunnel.chainEmpty')}</p>
      )}

      <select
        className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs"
        onChange={handleAdd}
        defaultValue=""
      >
        <option value="" disabled>{t('tunnel.chainAddPlaceholder')}</option>
        {available.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name} ({c.endpoint.host}:{c.endpoint.port})
          </option>
        ))}
      </select>
    </div>
  )
}
