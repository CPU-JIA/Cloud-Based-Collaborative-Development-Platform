import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, act, renderHook } from '@testing-library/react'
import { AuthProvider, useAuth } from '../contexts/AuthContext'
import { ReactNode } from 'react'

// Mock API
const mockAuthApi = {
  login: vi.fn(),
  getCurrentUser: vi.fn(),
  logout: vi.fn(),
  refreshToken: vi.fn(),
}

vi.mock('../utils/api', () => ({
  authApi: mockAuthApi,
}))

// Mock localStorage
const mockLocalStorage = (() => {
  let store: Record<string, string> = {}
  
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key]
    }),
    clear: vi.fn(() => {
      store = {}
    })
  }
})()

Object.defineProperty(window, 'localStorage', {
  value: mockLocalStorage
})

const mockUser = {
  id: '1',
  email: 'test@example.com',
  name: '测试用户',
  role: 'developer'
}

const mockTokens = {
  accessToken: 'mock-access-token',
  refreshToken: 'mock-refresh-token'
}

describe('AuthContext', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockLocalStorage.clear()
  })

  it('provides initial auth state', () => {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <AuthProvider>{children}</AuthProvider>
    )

    const { result } = renderHook(() => useAuth(), { wrapper })

    expect(result.current.user).toBeNull()
    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.isLoading).toBe(true)
  })

  it('handles successful login', async () => {
    mockAuthApi.login.mockResolvedValue({
      data: {
        user: mockUser,
        tokens: mockTokens
      }
    })

    const wrapper = ({ children }: { children: ReactNode }) => (
      <AuthProvider>{children}</AuthProvider>
    )

    const { result } = renderHook(() => useAuth(), { wrapper })

    await act(async () => {
      await result.current.login('test@example.com', 'password123')
    })

    expect(mockAuthApi.login).toHaveBeenCalledWith('test@example.com', 'password123')
    expect(result.current.user).toEqual(mockUser)
    expect(result.current.isAuthenticated).toBe(true)
    expect(mockLocalStorage.setItem).toHaveBeenCalledWith('accessToken', mockTokens.accessToken)
    expect(mockLocalStorage.setItem).toHaveBeenCalledWith('refreshToken', mockTokens.refreshToken)
  })

  it('handles login failure', async () => {
    mockAuthApi.login.mockRejectedValue(new Error('Invalid credentials'))

    const wrapper = ({ children }: { children: ReactNode }) => (
      <AuthProvider>{children}</AuthProvider>
    )

    const { result } = renderHook(() => useAuth(), { wrapper })

    await act(async () => {
      try {
        await result.current.login('test@example.com', 'wrongpassword')
      } catch (error) {
        expect(error).toBeInstanceOf(Error)
        expect((error as Error).message).toBe('Invalid credentials')
      }
    })

    expect(result.current.user).toBeNull()
    expect(result.current.isAuthenticated).toBe(false)
  })

  it('handles logout', async () => {
    // 先设置已登录状态
    mockLocalStorage.setItem('accessToken', mockTokens.accessToken)
    mockAuthApi.logout.mockResolvedValue({})

    const wrapper = ({ children }: { children: ReactNode }) => (
      <AuthProvider>{children}</AuthProvider>
    )

    const { result } = renderHook(() => useAuth(), { wrapper })

    // 设置初始用户状态
    await act(async () => {
      result.current.setUser(mockUser)
    })

    await act(async () => {
      await result.current.logout()
    })

    expect(mockAuthApi.logout).toHaveBeenCalled()
    expect(result.current.user).toBeNull()
    expect(result.current.isAuthenticated).toBe(false)
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('accessToken')
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('refreshToken')
  })

  it('restores user session from token', async () => {
    mockLocalStorage.setItem('accessToken', mockTokens.accessToken)
    mockAuthApi.getCurrentUser.mockResolvedValue({ data: mockUser })

    const wrapper = ({ children }: { children: ReactNode }) => (
      <AuthProvider>{children}</AuthProvider>
    )

    const { result } = renderHook(() => useAuth(), { wrapper })

    // 等待初始化完成
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 100))
    })

    expect(mockAuthApi.getCurrentUser).toHaveBeenCalled()
  })

  it('handles token refresh', async () => {
    mockLocalStorage.setItem('refreshToken', mockTokens.refreshToken)
    mockAuthApi.refreshToken.mockResolvedValue({
      data: {
        accessToken: 'new-access-token',
        refreshToken: 'new-refresh-token'
      }
    })

    const wrapper = ({ children }: { children: ReactNode }) => (
      <AuthProvider>{children}</AuthProvider>
    )

    const { result } = renderHook(() => useAuth(), { wrapper })

    await act(async () => {
      await result.current.refreshAccessToken()
    })

    expect(mockAuthApi.refreshToken).toHaveBeenCalledWith(mockTokens.refreshToken)
    expect(mockLocalStorage.setItem).toHaveBeenCalledWith('accessToken', 'new-access-token')
    expect(mockLocalStorage.setItem).toHaveBeenCalledWith('refreshToken', 'new-refresh-token')
  })

  it('handles invalid token during session restore', async () => {
    mockLocalStorage.setItem('accessToken', 'invalid-token')
    mockAuthApi.getCurrentUser.mockRejectedValue(new Error('Unauthorized'))

    const wrapper = ({ children }: { children: ReactNode }) => (
      <AuthProvider>{children}</AuthProvider>
    )

    const { result } = renderHook(() => useAuth(), { wrapper })

    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 100))
    })

    expect(result.current.user).toBeNull()
    expect(result.current.isAuthenticated).toBe(false)
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('accessToken')
  })
})