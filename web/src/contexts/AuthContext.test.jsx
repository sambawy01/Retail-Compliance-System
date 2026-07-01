import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AuthProvider, useAuth } from '../contexts/AuthContext'

// Mock the api module
vi.mock('../services/api', () => {
  const api = {
    get: vi.fn(),
    post: vi.fn(),
  }
  return { default: api, TOKEN_KEY: 'watchdog_token' }
})

import api from '../services/api'

function TestConsumer() {
  const { user, isAuthenticated, login, logout, error, loading } = useAuth()
  return (
    <div>
      <span data-testid="auth-state">{isAuthenticated ? 'authenticated' : 'unauthenticated'}</span>
      <span data-testid="user-name">{user?.display_name || user?.email || 'none'}</span>
      <span data-testid="error">{error || 'no-error'}</span>
      <span data-testid="loading">{loading ? 'loading' : 'not-loading'}</span>
      <button onClick={() => login('test@example.com', 'password123')}>Login</button>
      <button onClick={() => logout()}>Logout</button>
    </div>
  )
}

describe('AuthContext', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('starts unauthenticated with no token', async () => {
    render(<AuthProvider><TestConsumer /></AuthProvider>)
    await waitFor(() => {
      expect(screen.getByTestId('auth-state')).toHaveTextContent('unauthenticated')
    })
  })

  it('starts loading when a token exists in localStorage', async () => {
    localStorage.setItem('watchdog_token', 'fake-token')
    api.get.mockResolvedValueOnce({
      data: { user: { user_id: '1', email: 'test@example.com', display_name: 'Test User', role: 'admin' } },
    })
    render(<AuthProvider><TestConsumer /></AuthProvider>)
    // Eventually resolves to authenticated
    await waitFor(() => {
      expect(screen.getByTestId('auth-state')).toHaveTextContent('authenticated')
    })
    expect(screen.getByTestId('user-name')).toHaveTextContent('Test User')
  })

  it('removes token and sets unauthenticated when /me fails', async () => {
    localStorage.setItem('watchdog_token', 'expired-token')
    api.get.mockRejectedValueOnce(new Error('401'))
    render(<AuthProvider><TestConsumer /></AuthProvider>)
    await waitFor(() => {
      expect(screen.getByTestId('auth-state')).toHaveTextContent('unauthenticated')
    })
    expect(localStorage.getItem('watchdog_token')).toBeNull()
  })

  it('login stores token and sets user', async () => {
    api.post.mockResolvedValueOnce({
      data: {
        token: 'new-jwt-token',
        user: { user_id: '1', email: 'test@example.com', display_name: 'Test User', role: 'admin' },
      },
    })
    render(<AuthProvider><TestConsumer /></AuthProvider>)
    await waitFor(() => {
      expect(screen.getByTestId('auth-state')).toHaveTextContent('unauthenticated')
    })
    await act(async () => {
      await userEvent.click(screen.getByText('Login'))
    })
    await waitFor(() => {
      expect(screen.getByTestId('auth-state')).toHaveTextContent('authenticated')
    })
    expect(screen.getByTestId('user-name')).toHaveTextContent('Test User')
    expect(localStorage.getItem('watchdog_token')).toBe('new-jwt-token')
  })

  it('login sets error on failure', async () => {
    api.post.mockRejectedValueOnce({
      response: { data: { error: 'invalid credentials' } },
    })
    render(<AuthProvider><TestConsumer /></AuthProvider>)
    await waitFor(() => {
      expect(screen.getByTestId('auth-state')).toHaveTextContent('unauthenticated')
    })
    await act(async () => {
      await userEvent.click(screen.getByText('Login'))
    })
    await waitFor(() => {
      expect(screen.getByTestId('error')).toHaveTextContent('invalid credentials')
    })
    expect(screen.getByTestId('auth-state')).toHaveTextContent('unauthenticated')
  })

  it('logout clears token and user', async () => {
    localStorage.setItem('watchdog_token', 'fake-token')
    api.get.mockResolvedValueOnce({
      data: { user: { user_id: '1', email: 'test@example.com', display_name: 'Test User' } },
    })
    api.post.mockResolvedValueOnce({})
    render(<AuthProvider><TestConsumer /></AuthProvider>)
    await waitFor(() => {
      expect(screen.getByTestId('auth-state')).toHaveTextContent('authenticated')
    })
    await act(async () => {
      await userEvent.click(screen.getByText('Logout'))
    })
    await waitFor(() => {
      expect(screen.getByTestId('auth-state')).toHaveTextContent('unauthenticated')
    })
    expect(localStorage.getItem('watchdog_token')).toBeNull()
  })
})