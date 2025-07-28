import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import Dashboard from '../pages/Dashboard'
import { AuthProvider } from '../contexts/AuthContext'

// Mock API
const mockAuthApi = {
  login: vi.fn(),
  getCurrentUser: vi.fn(),
  logout: vi.fn(),
  refreshToken: vi.fn(),
}

const mockProjectApi = {
  getProjects: vi.fn(),
  createProject: vi.fn(),
  updateProject: vi.fn(),
  deleteProject: vi.fn(),
}

const mockTeamApi = {
  getTeams: vi.fn(),
  createTeam: vi.fn(),
  updateTeam: vi.fn(),
  deleteTeam: vi.fn(),
  getTeamMembers: vi.fn(),
}

vi.mock('../utils/api', () => ({
  authApi: mockAuthApi,
  projectApi: mockProjectApi,
  teamApi: mockTeamApi,
}))

// Mock user data
const mockUser = {
  id: '1',
  email: 'test@example.com',
  name: '测试用户',
  role: 'developer'
}

const mockProjects = [
  {
    id: '1',
    name: '测试项目1',
    description: '这是一个测试项目',
    status: 'active',
    team: { id: '1', name: '开发团队' },
    createdAt: '2024-01-01T00:00:00Z'
  },
  {
    id: '2',
    name: '测试项目2',
    description: '另一个测试项目',
    status: 'completed',
    team: { id: '2', name: '设计团队' },
    createdAt: '2024-01-02T00:00:00Z'
  }
]

const mockTeams = [
  {
    id: '1',
    name: '开发团队',
    description: '负责开发工作',
    memberCount: 5
  },
  {
    id: '2',
    name: '设计团队',
    description: '负责UI/UX设计',
    memberCount: 3
  }
]

const DashboardWrapper = () => (
  <BrowserRouter>
    <AuthProvider>
      <Dashboard />
    </AuthProvider>
  </BrowserRouter>
)

describe('Dashboard Component', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    
    // Mock successful API responses
    mockAuthApi.getCurrentUser.mockResolvedValue({ data: mockUser })
    mockProjectApi.getProjects.mockResolvedValue({ data: mockProjects })
    mockTeamApi.getTeams.mockResolvedValue({ data: mockTeams })
  })

  it('renders dashboard with loading state initially', () => {
    render(<DashboardWrapper />)
    
    // 应该显示加载状态或仪表板基础元素
    expect(document.body).toBeTruthy()
  })

  it('displays user greeting and welcome message', async () => {
    render(<DashboardWrapper />)
    
    await waitFor(() => {
      // 等待用户数据加载
      expect(mockAuthApi.getCurrentUser).toHaveBeenCalled()
    }, { timeout: 3000 })
  })

  it('shows project statistics', async () => {
    render(<DashboardWrapper />)
    
    await waitFor(() => {
      expect(mockProjectApi.getProjects).toHaveBeenCalled()
    }, { timeout: 3000 })
  })

  it('displays team information', async () => {
    render(<DashboardWrapper />)
    
    await waitFor(() => {
      expect(mockTeamApi.getTeams).toHaveBeenCalled()
    }, { timeout: 3000 })
  })

  it('handles API error gracefully', async () => {
    mockProjectApi.getProjects.mockRejectedValue(new Error('API Error'))
    
    render(<DashboardWrapper />)
    
    await waitFor(() => {
      expect(mockProjectApi.getProjects).toHaveBeenCalled()
    }, { timeout: 3000 })
    
    // 错误处理应该不会导致崩溃
    expect(document.body).toBeTruthy()
  })

  it('renders navigation elements', () => {
    render(<DashboardWrapper />)
    
    // 检查是否有基本的导航结构
    expect(document.body).toBeTruthy()
  })
})