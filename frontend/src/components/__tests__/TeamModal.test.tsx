import React from 'react';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { jest } from '@jest/globals';

import TeamModal from '../TeamModal';
import { useAuth } from '../../contexts/AuthContext';

// Mock dependencies
jest.mock('../../contexts/AuthContext');
const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

// Mock fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

// Mock data
const mockUser = {
  id: 1,
  username: 'testuser',
  email: 'test@example.com',
  display_name: '测试用户'
};

const mockTeams = [
  {
    id: 1,
    project_id: 1,
    name: '开发团队',
    description: '负责产品开发',
    members: [
      {
        id: 1,
        user_id: 1,
        role_id: 1,
        status: 'active',
        joined_at: '2024-01-01T00:00:00Z',
        invited_by: 1,
        user: mockUser,
        role: { id: 1, name: 'owner', description: '团队所有者', permissions: ['read', 'write', 'delete', 'share', 'admin'], is_system: true }
      }
    ],
    is_active: true,
    created_by: 1,
    created_at: '2024-01-01T00:00:00Z'
  }
];

const mockUsers = [mockUser];

const mockRoles = [
  { id: 1, name: 'owner', description: '团队所有者', permissions: ['read', 'write', 'delete', 'share', 'admin'], is_system: true },
  { id: 2, name: 'admin', description: '团队管理员', permissions: ['read', 'write', 'delete', 'share'], is_system: true },
  { id: 3, name: 'member', description: '团队成员', permissions: ['read', 'write', 'share'], is_system: true },
  { id: 4, name: 'viewer', description: '团队观察者', permissions: ['read'], is_system: true }
];

const mockInvitations = [
  {
    id: 1,
    team_id: 1,
    email: 'newuser@example.com',
    role_id: 3,
    token: 'test-token',
    status: 'pending',
    expires_at: '2024-12-31T23:59:59Z',
    message: '欢迎加入我们的团队',
    invited_by: 1,
    created_at: '2024-01-01T00:00:00Z'
  }
];

const mockPermissionRequests = [
  {
    id: 1,
    project_id: 1,
    user_id: 2,
    request_type: 'role',
    permission: 'admin',
    reason: '需要管理权限处理项目',
    status: 'pending',
    created_at: '2024-01-01T00:00:00Z'
  }
];

// Test suite
describe('TeamModal', () => {
  const defaultProps = {
    projectId: 1,
    isOpen: true,
    onClose: jest.fn()
  };

  beforeEach(() => {
    mockUseAuth.mockReturnValue({
      user: mockUser,
      login: jest.fn(),
      logout: jest.fn(),
      register: jest.fn(),
      isLoading: false,
      error: null
    });

    // Reset fetch mock
    mockFetch.mockReset();
    
    // Default successful responses
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ teams: mockTeams })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ users: mockUsers })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ roles: mockRoles })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ invitations: mockInvitations })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ requests: mockPermissionRequests })
      });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('不渲染当isOpen为false时', () => {
    render(<TeamModal {...defaultProps} isOpen={false} />);
    expect(screen.queryByText('👥 团队管理')).not.toBeInTheDocument();
  });

  it('渲染团队管理模态框', async () => {
    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('👥 团队管理')).toBeInTheDocument();
    });
    
    expect(screen.getByText('团队 (1)')).toBeInTheDocument();
    expect(screen.getByText('邀请 (1)')).toBeInTheDocument();
    expect(screen.getByText('权限申请 (1)')).toBeInTheDocument();
  });

  it('显示团队列表和详情', async () => {
    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('开发团队')).toBeInTheDocument();
    });
    
    expect(screen.getByText('负责产品开发')).toBeInTheDocument();
    expect(screen.getByText('👥 1 成员')).toBeInTheDocument();
  });

  it('切换到邀请标签页', async () => {
    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('团队管理')).toBeInTheDocument();
    });
    
    const invitationTab = screen.getByText('邀请 (1)');
    fireEvent.click(invitationTab);
    
    await waitFor(() => {
      expect(screen.getByText('团队邀请管理')).toBeInTheDocument();
      expect(screen.getByText('newuser@example.com')).toBeInTheDocument();
      expect(screen.getByText('欢迎加入我们的团队')).toBeInTheDocument();
    });
  });

  it('切换到权限申请标签页', async () => {
    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('团队管理')).toBeInTheDocument();
    });
    
    const requestTab = screen.getByText('权限申请 (1)');
    fireEvent.click(requestTab);
    
    await waitFor(() => {
      expect(screen.getByText('权限申请管理')).toBeInTheDocument();
      expect(screen.getByText('申请权限')).toBeInTheDocument();
      expect(screen.getByText('需要管理权限处理项目')).toBeInTheDocument();
    });
  });

  it('创建新团队', async () => {
    const user = userEvent.setup();
    
    // Mock successful team creation
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ 
        id: 2, 
        name: '新团队', 
        description: '新团队描述',
        project_id: 1,
        members: [],
        is_active: true,
        created_by: 1,
        created_at: '2024-01-01T00:00:00Z'
      })
    });

    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('团队管理')).toBeInTheDocument();
    });
    
    // 点击创建团队按钮
    const createButton = screen.getByText('➕ 创建团队');
    await user.click(createButton);
    
    // 填写团队信息
    const nameInput = screen.getByPlaceholderText('输入团队名称...');
    const descInput = screen.getByPlaceholderText('输入团队描述...');
    
    await user.type(nameInput, '新团队');
    await user.type(descInput, '新团队描述');
    
    // 提交创建
    const submitButton = screen.getByText('创建');
    await user.click(submitButton);
    
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/teams', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify({
          project_id: 1,
          name: '新团队',
          description: '新团队描述',
        }),
      });
    });
  });

  it('选择团队并显示成员详情', async () => {
    const user = userEvent.setup();
    
    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('开发团队')).toBeInTheDocument();
    });
    
    // 点击团队
    const teamItem = screen.getByText('开发团队');
    await user.click(teamItem);
    
    await waitFor(() => {
      expect(screen.getByText('团队成员')).toBeInTheDocument();
      expect(screen.getByText('测试用户')).toBeInTheDocument();
      expect(screen.getByText('test@example.com')).toBeInTheDocument();
    });
    
    // 验证邀请成员按钮存在
    expect(screen.getByText('📧 邀请成员')).toBeInTheDocument();
  });

  it('邀请新用户', async () => {
    const user = userEvent.setup();
    
    // Mock successful invitation creation
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ 
        id: 2,
        team_id: 1,
        email: 'invite@example.com',
        role_id: 3,
        token: 'new-token',
        status: 'pending',
        expires_at: '2024-12-31T23:59:59Z',
        message: '欢迎加入',
        invited_by: 1,
        created_at: '2024-01-01T00:00:00Z'
      })
    });

    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('开发团队')).toBeInTheDocument();
    });
    
    // 选择团队
    const teamItem = screen.getByText('开发团队');
    await user.click(teamItem);
    
    await waitFor(() => {
      expect(screen.getByText('📧 邀请成员')).toBeInTheDocument();
    });
    
    // 点击邀请成员
    const inviteButton = screen.getByText('📧 邀请成员');
    await user.click(inviteButton);
    
    // 填写邀请信息
    const emailInput = screen.getByPlaceholderText('输入用户邮箱...');
    const messageInput = screen.getByPlaceholderText('输入邀请消息...');
    
    await user.type(emailInput, 'invite@example.com');
    await user.type(messageInput, '欢迎加入');
    
    // 提交邀请
    const sendButton = screen.getByText('发送邀请');
    await user.click(sendButton);
    
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/invitations', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify({
          team_id: 1,
          email: 'invite@example.com',
          role_id: 3,
          message: '欢迎加入',
        }),
      });
    });
  });

  it('创建权限申请', async () => {
    const user = userEvent.setup();
    
    // Mock successful request creation
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ 
        id: 2,
        project_id: 1,
        user_id: 1,
        request_type: 'role',
        permission: 'write',
        reason: '需要写入权限',
        status: 'pending',
        created_at: '2024-01-01T00:00:00Z'
      })
    });

    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('团队管理')).toBeInTheDocument();
    });
    
    // 切换到权限申请标签页
    const requestTab = screen.getByText('权限申请 (1)');
    await user.click(requestTab);
    
    await waitFor(() => {
      expect(screen.getByText('➕ 申请权限')).toBeInTheDocument();
    });
    
    // 点击申请权限
    const requestButton = screen.getByText('➕ 申请权限');
    await user.click(requestButton);
    
    // 填写申请信息
    const reasonInput = screen.getByPlaceholderText('请详细说明申请理由...');
    await user.type(reasonInput, '需要写入权限');
    
    // 选择权限类型
    const permissionSelect = screen.getByDisplayValue('读取权限');
    await user.selectOptions(permissionSelect, '写入权限');
    
    // 提交申请
    const submitButton = screen.getByText('提交申请');
    await user.click(submitButton);
    
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/permission-requests', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify({
          project_id: 1,
          user_id: 1,
          request_type: 'role',
          permission: 'write',
          reason: '需要写入权限',
          target_id: undefined,
        }),
      });
    });
  });

  it('审批权限申请', async () => {
    const user = userEvent.setup();
    
    // Mock successful review
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ success: true })
    });

    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('团队管理')).toBeInTheDocument();
    });
    
    // 切换到权限申请标签页
    const requestTab = screen.getByText('权限申请 (1)');
    await user.click(requestTab);
    
    await waitFor(() => {
      expect(screen.getByText('批准')).toBeInTheDocument();
    });
    
    // 点击批准按钮
    const approveButton = screen.getByText('批准');
    await user.click(approveButton);
    
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/permission-requests/1/review', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify({
          approved: true,
          review_reason: '申请已批准',
        }),
      });
    });
  });

  it('更新成员角色', async () => {
    const user = userEvent.setup();
    
    // Mock successful role update
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ success: true })
    });

    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('开发团队')).toBeInTheDocument();
    });
    
    // 选择团队
    const teamItem = screen.getByText('开发团队');
    await user.click(teamItem);
    
    await waitFor(() => {
      expect(screen.getByText('团队成员')).toBeInTheDocument();
    });
    
    // 查找角色选择下拉框
    const roleSelect = screen.getByDisplayValue('owner');
    await user.selectOptions(roleSelect, '2'); // 选择管理员角色
    
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/teams/1/members/1', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify({ role_id: 2 }),
      });
    });
  });

  it('关闭模态框', async () => {
    const user = userEvent.setup();
    const onCloseMock = jest.fn();
    
    render(<TeamModal {...defaultProps} onClose={onCloseMock} />);
    
    await waitFor(() => {
      expect(screen.getByText('团队管理')).toBeInTheDocument();
    });
    
    // 点击关闭按钮
    const closeButton = screen.getByText('✕');
    await user.click(closeButton);
    
    expect(onCloseMock).toHaveBeenCalled();
  });

  it('处理加载状态', () => {
    // Mock pending fetch
    mockFetch.mockImplementation(() => new Promise(() => {}));
    
    render(<TeamModal {...defaultProps} />);
    
    expect(screen.getByText('加载中...')).toBeInTheDocument();
  });

  it('显示空状态', async () => {
    // Mock empty responses
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ teams: [] })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ users: [] })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ roles: [] })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ invitations: [] })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ requests: [] })
      });

    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('选择一个团队查看详情')).toBeInTheDocument();
    });
    
    // 切换到邀请标签页
    const invitationTab = screen.getByText('邀请 (0)');
    fireEvent.click(invitationTab);
    
    await waitFor(() => {
      expect(screen.getByText('暂无邀请记录')).toBeInTheDocument();
    });
    
    // 切换到权限申请标签页
    const requestTab = screen.getByText('权限申请 (0)');
    fireEvent.click(requestTab);
    
    await waitFor(() => {
      expect(screen.getByText('暂无权限申请')).toBeInTheDocument();
    });
  });

  it('验证表单输入', async () => {
    const user = userEvent.setup();
    
    render(<TeamModal {...defaultProps} />);
    
    await waitFor(() => {
      expect(screen.getByText('➕ 创建团队')).toBeInTheDocument();
    });
    
    // 点击创建团队但不填写信息
    const createButton = screen.getByText('➕ 创建团队');
    await user.click(createButton);
    
    const submitButton = screen.getByText('创建');
    expect(submitButton).toBeDisabled(); // 应该禁用因为名称为空
    
    // 填写名称
    const nameInput = screen.getByPlaceholderText('输入团队名称...');
    await user.type(nameInput, '测试团队');
    
    expect(submitButton).toBeEnabled(); // 现在应该启用
  });
});