import { Component } from 'react'

export class ErrorBoundary extends Component {
  constructor(props) {
    super(props)
    this.state = { hasError: false, error: null, errorInfo: null }
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error }
  }

  componentDidCatch(error, errorInfo) {
    this.setState({ errorInfo })
    console.error('ErrorBoundary caught:', error, errorInfo)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{ padding: '20px', color: '#e5e7eb', background: '#0f1115', minHeight: '100vh', fontFamily: 'monospace' }}>
          <h1 style={{ color: '#ef4444', marginBottom: '16px' }}>React Error</h1>
          <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
            {this.state.error?.toString()}
          </pre>
          <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all', marginTop: '10px', color: '#9ca3af' }}>
            {this.state.errorInfo?.componentStack}
          </pre>
        </div>
      )
    }
    return this.props.children
  }
}
