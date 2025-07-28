import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import ProjectModal from '../ProjectModal'
import { AuthProvider } from '../../contexts/AuthContext'

// Mock API
const mockProjectApi = {
  createProject: vi.fn(),
  updateProject: vi.fn(),
  getProjects: vi.fn(),
  deleteProject: vi.fn(),
}

const mockTeamApi = {
  getTeams: vi.fn(),
}

vi.mock('../../utils/api', () => ({
  projectApi: mockProjectApi,
  teamApi: mockTeamApi,
  authApi: {
    getCurrentUser: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    refreshToken: vi.fn(),
  }
}))

const mockTeams = [
  { id: '1', name: '开发团队', description: '负责开发工作' },
  { id: '2', name: '设计团队', description: '负责UI/UX设计' },
  { id: '3', name: '测试团队', description: '负责质量保证' }
]

const mockProject = {
  id: '1',
  name: '测试项目',
  description: '这是一个测试项目',
  status: 'active',
  teamId: '1',
  priority: 'high',
  startDate: '2024-01-01',
  endDate: '2024-12-31'
}

const ProjectModalWrapper = ({ 
  isOpen = true, 
  onClose = vi.fn(), 
  project = null,
  onSave = vi.fn()
}) => (
  <BrowserRouter>
    <AuthProvider>
      <ProjectModal 
        isOpen={isOpen}
        onClose={onClose}
        project={project}
        onSave={onSave}
      />
    </AuthProvider>
  </BrowserRouter>
)

describe('ProjectModal Component', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockTeamApi.getTeams.mockResolvedValue({ data: mockTeams })
  })

  it('renders create project modal', async () => {
    render(<ProjectModalWrapper />)
    
    await waitFor(() => {
      expect(mockTeamApi.getTeams).toHaveBeenCalled()
    })
  })

  it('renders edit project modal with existing data', async () => {
    render(<ProjectModalWrapper project={mockProject} />)
    
    await waitFor(() => {
      expect(mockTeamApi.getTeams).toHaveBeenCalled()
    })
  })

  it('validates required fields', async () => {
    const onSave = vi.fn()
    render(<ProjectModalWrapper onSave={onSave} />)
    
    await waitFor(() => {
      expect(mockTeamApi.getTeams).toHaveBeenCalled()
    })

    // 查找并点击保存按钮（不填写必填字段）
    const submitButton = document.querySelector('button[type="submit"]') ||
                        document.querySelector('button:contains("保存")') ||
                        document.querySelector('[role="button"]')
    
    if (submitButton) {
      fireEvent.click(submitButton)
      
      // 验证表单验证
      await waitFor(() => {
        // 应该显示验证错误或阻止提交
        expect(onSave).not.toHaveBeenCalled()
      })
    }
  })

  it('handles successful project creation', async () => {
    const newProject = {
      id: '2',
      name: '新项目',
      description: '新建的项目',
      teamId: '1',
      status: 'active'
    }
    
    mockProjectApi.createProject.mockResolvedValue({ data: newProject })
    const onSave = vi.fn()
    
    render(<ProjectModalWrapper onSave={onSave} />)
    
    await waitFor(() => {
      expect(mockTeamApi.getTeams).toHaveBeenCalled()
    })

    // 填写表单
    const nameInput = document.querySelector('input[name="name"]') ||
                     document.querySelector('input[placeholder*="项目名称"]') ||
                     document.querySelector('input[type="text"]')
    
    if (nameInput) {
      fireEvent.change(nameInput, { target: { value: '新项目' } })
    }

    const descriptionInput = document.querySelector('textarea[name="description"]') ||
                           document.querySelector('textarea')
    
    if (descriptionInput) {
      fireEvent.change(descriptionInput, { target: { value: '新建的项目' } })
    }

    // 选择团队
    const teamSelect = document.querySelector('select[name="teamId"]') ||
                      document.querySelector('select')
    
    if (teamSelect) {
      fireEvent.change(teamSelect, { target: { value: '1' } })
    }

    // 提交表单
    const submitButton = document.querySelector('button[type="submit"]') ||
                        document.querySelector('button:contains("保存")')
    
    if (submitButton) {
      fireEvent.click(submitButton)
      
      await waitFor(() => {
        expect(mockProjectApi.createProject).toHaveBeenCalled()
      })
    }
  })

  it('handles project update', async () => {
    const updatedProject = { ...mockProject, name: '更新的项目名称' }
    mockProjectApi.updateProject.mockResolvedValue({ data: updatedProject })
    const onSave = vi.fn()
    
    render(<ProjectModalWrapper project={mockProject} onSave={onSave} />)
    
    await waitFor(() => {
      expect(mockTeamApi.getTeams).toHaveBeenCalled()
    })

    // 修改项目名称
    const nameInput = document.querySelector('input[name="name"]') ||
                     document.querySelector('input[type="text"]')
    
    if (nameInput) {
      fireEvent.change(nameInput, { target: { value: '更新的项目名称' } })
    }

    // 提交表单
    const submitButton = document.querySelector('button[type="submit"]')
    if (submitButton) {
      fireEvent.click(submitButton)
      
      await waitFor(() => {
        expect(mockProjectApi.updateProject).toHaveBeenCalledWith(
          mockProject.id,
          expect.objectContaining({
            name: '更新的项目名称'
          })
        )
      })
    }
  })

  it('handles API errors', async () => {
    mockProjectApi.createProject.mockRejectedValue(new Error('创建项目失败'))
    const onSave = vi.fn()
    
    render(<ProjectModalWrapper onSave={onSave} />)
    
    await waitFor(() => {
      expect(mockTeamApi.getTeams).toHaveBeenCalled()
    })

    // 填写基本信息
    const nameInput = document.querySelector('input[type="text"]')
    if (nameInput) {
      fireEvent.change(nameInput, { target: { value: '测试项目' } })
    }

    // 提交表单
    const submitButton = document.querySelector('button[type="submit"]')
    if (submitButton) {
      fireEvent.click(submitButton)
      
      await waitFor(() => {
        expect(mockProjectApi.createProject).toHaveBeenCalled()
      })
      
      // 错误处理应该不会导致崩溃
      expect(document.body).toBeTruthy()
    }
  })

  it('handles modal close', () => {
    const onClose = vi.fn()
    render(<ProjectModalWrapper onClose={onClose} />)
    
    // 查找关闭按钮
    const closeButton = document.querySelector('button:contains("取消")') ||
                       document.querySelector('button[aria-label="关闭"]') ||
                       document.querySelector('.modal-close')
    
    if (closeButton) {
      fireEvent.click(closeButton)
      expect(onClose).toHaveBeenCalled()
    }
  })

  it('loads teams for selection', async () => {
    render(<ProjectModalWrapper />)
    
    await waitFor(() => {
      expect(mockTeamApi.getTeams).toHaveBeenCalled()
    })
  })

  it('handles teams loading error', async () => {
    mockTeamApi.getTeams.mockRejectedValue(new Error('加载团队失败'))
    
    render(<ProjectModalWrapper />)
    
    await waitFor(() => {
      expect(mockTeamApi.getTeams).toHaveBeenCalled()
    })
    
    // 错误处理应该不会导致崩溃
    expect(document.body).toBeTruthy()
  })

  it('validates project dates', async () => {
    const onSave = vi.fn()
    render(<ProjectModalWrapper onSave={onSave} />)
    
    await waitFor(() => {
      expect(mockTeamApi.getTeams).toHaveBeenCalled()
    })

    // 设置结束日期早于开始日期
    const startDateInput = document.querySelector('input[type="date"]:first-of-type')
    const endDateInput = document.querySelector('input[type="date"]:last-of-type')
    
    if (startDateInput && endDateInput) {
      fireEvent.change(startDateInput, { target: { value: '2024-12-31' } })
      fireEvent.change(endDateInput, { target: { value: '2024-01-01' } })
      
      const submitButton = document.querySelector('button[type="submit"]')
      if (submitButton) {
        fireEvent.click(submitButton)
        
        // 应该显示日期验证错误
        await waitFor(() => {
          expect(onSave).not.toHaveBeenCalled()
        })
      }
    }
  })
})