import { Component, ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('ErrorBoundary caught error:', error, errorInfo)
  }

  handleReload = () => {
    window.location.reload()
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen bg-base-100 flex items-center justify-center">
          <div className="text-center max-w-md px-4" role="alert">
            <div className="text-error text-6xl mb-4" aria-hidden="true">!</div>
            <h1 className="text-xl font-semibold text-base-content mb-2">
              Something went wrong
            </h1>
            <p className="text-base-content/60 mb-4">
              An unexpected error occurred. You can try reloading the page.
            </p>
            {this.state.error && (
              <details className="mb-4 text-left">
                <summary className="cursor-pointer text-sm text-base-content/50">
                  Error details
                </summary>
                <pre className="mt-2 p-2 bg-base-200 rounded text-xs overflow-auto max-h-32">
                  {this.state.error.message}
                </pre>
              </details>
            )}
            <div className="flex gap-2 justify-center">
              <button onClick={this.handleReset} className="btn btn-outline">
                Try Again
              </button>
              <button onClick={this.handleReload} className="btn btn-primary">
                Reload Page
              </button>
            </div>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
