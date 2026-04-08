import { useState } from 'react'
import { Eye, EyeOff } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useRevealSSHConnection } from '@/hooks/use-ssh-connections'
import type { SSHConnection } from '@/types/api'

type FieldPath = 'password' | 'passphrase' | 'keyPem'

interface PasswordRevealInputProps extends Omit<React.ComponentProps<'input'>, 'type'> {
  connectionId?: string
  fieldPath: FieldPath
  onRevealValue?: (value: string) => void
}

function extractField(conn: SSHConnection, field: FieldPath): string {
  switch (field) {
    case 'password':
      return conn.auth?.password ?? ''
    case 'passphrase':
      return conn.auth?.privateKey?.passphrase ?? ''
    case 'keyPem':
      return conn.auth?.privateKey?.keyPem ?? ''
  }
}

export function PasswordRevealInput({
  connectionId,
  fieldPath,
  onRevealValue,
  value,
  ...props
}: PasswordRevealInputProps) {
  const [visible, setVisible] = useState(false)
  const reveal = useRevealSSHConnection()

  const toggle = async () => {
    if (visible) {
      setVisible(false)
      return
    }
    // If editing an existing connection and field is empty, fetch real value
    if (connectionId && !value) {
      try {
        const conn = await reveal.mutateAsync(connectionId)
        const real = extractField(conn, fieldPath)
        if (real) onRevealValue?.(real)
      } catch {
        // Reveal failed — just toggle visibility of whatever is typed
      }
    }
    setVisible(true)
  }

  return (
    <div className="relative">
      <Input
        type={visible ? 'text' : 'password'}
        value={value}
        className="pr-9"
        {...props}
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="absolute right-0 top-0 h-8 w-8 text-muted-foreground hover:text-foreground"
        onClick={toggle}
        tabIndex={-1}
      >
        {visible ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
      </Button>
    </div>
  )
}
