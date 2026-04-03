import { Component } from 'react'
import type { ErrorInfo, ReactNode } from 'react'
import i18n from '@/lib/i18n'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error?: Error
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('ErrorBoundary caught:', error, info.componentStack)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex h-screen flex-col items-center justify-center gap-4">
          <h2 className="text-xl font-bold">{i18n.t('error.somethingWentWrong')}</h2>
          <p className="max-w-md text-center text-sm text-muted-foreground">
            {this.state.error?.message || i18n.t('error.unexpectedError')}
          </p>
          <button
            onClick={() => window.location.reload()}
            className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            {i18n.t('common.reload')}
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
