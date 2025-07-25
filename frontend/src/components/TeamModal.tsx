import React, { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';

// 团队和成员类型定义
interface Team {
  id: number;
  project_id: number;
  name: string;
  description: string;
  members: TeamMember[];
  is_active: boolean;
  created_by: number;
  created_at: string;
}

interface TeamMember {
  id: number;
  user_id: number;
  role_id: number;
  status: string;
  joined_at: string;
  invited_by: number;
  user: User;
  role: Role;
}

interface User {
  id: number;
  username: string;
  email: string;
  display_name: string;
  avatar: string;
  department: string;
  position: string;
  status: string;
}

interface Role {
  id: number;
  name: string;
  description: string;
  permissions: string[];
  is_system: boolean;
}

interface Invitation {
  id: number;
  team_id: number;
  email: string;
  role_id: number;
  token: string;
  status: string;
  expires_at: string;
  message: string;
  invited_by: number;
  created_at: string;
}

interface PermissionRequest {
  id: number;
  project_id: number;
  user_id: number;
  request_type: string;
  target_id?: number;
  permission: string;
  reason: string;
  status: string;
  reviewed_by?: number;
  reviewed_at?: string;
  created_at: string;
}

interface TeamModalProps {
  projectId: number;
  isOpen: boolean;
  onClose: () => void;
}

const TeamModal: React.FC<TeamModalProps> = ({ projectId, isOpen, onClose }) => {
  const { user } = useAuth();
  const [teams, setTeams] = useState<Team[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [permissionRequests, setPermissionRequests] = useState<PermissionRequest[]>([]);
  const [loading, setLoading] = useState(false);
  
  // UI状态
  const [activeTab, setActiveTab] = useState<'teams' | 'invitations' | 'requests'>('teams');
  const [selectedTeam, setSelectedTeam] = useState<Team | null>(null);
  const [showCreateTeam, setShowCreateTeam] = useState(false);
  const [showInviteUser, setShowInviteUser] = useState(false);
  const [showCreateRequest, setShowCreateRequest] = useState(false);
  
  // 表单状态
  const [newTeamData, setNewTeamData] = useState({ name: '', description: '' });
  const [inviteData, setInviteData] = useState({ email: '', role_id: 3, message: '' });
  const [requestData, setRequestData] = useState({ 
    request_type: 'role', 
    permission: 'read', 
    reason: '',
    target_id: undefined as number | undefined 
  });
  const [searchTerm, setSearchTerm] = useState('');

  // 加载数据
  const loadData = async () => {
    setLoading(true);
    try {
      // 并行加载所有数据
      const [teamsRes, usersRes, rolesRes, invitationsRes, requestsRes] = await Promise.all([
        fetch(`/api/v1/teams/project/${projectId}`, {
          headers: { 'X-Tenant-ID': 'default' }
        }),
        fetch('/api/v1/users?limit=50', {
          headers: { 'X-Tenant-ID': 'default' }
        }),
        fetch(`/api/v1/roles/project/${projectId}`, {
          headers: { 'X-Tenant-ID': 'default' }
        }),
        fetch(`/api/v1/invitations/team/1`, {
          headers: { 'X-Tenant-ID': 'default' }
        }),
        fetch(`/api/v1/permission-requests/project/${projectId}`, {
          headers: { 'X-Tenant-ID': 'default' }
        })
      ]);

      if (teamsRes.ok) {
        const data = await teamsRes.json();
        setTeams(data.teams || []);
      }

      if (usersRes.ok) {
        const data = await usersRes.json();
        setUsers(data.users || []);
      }

      if (rolesRes.ok) {
        const data = await rolesRes.json();
        setRoles(data.roles || []);
      }

      if (invitationsRes.ok) {
        const data = await invitationsRes.json();
        setInvitations(data.invitations || []);
      }

      if (requestsRes.ok) {
        const data = await requestsRes.json();
        setPermissionRequests(data.requests || []);
      }
    } catch (error) {
      console.error('加载团队数据失败:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen) {
      loadData();
    }
  }, [isOpen, projectId]);

  // 创建团队
  const handleCreateTeam = async () => {
    if (!newTeamData.name.trim()) return;

    try {
      const response = await fetch('/api/v1/teams', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify({
          project_id: projectId,
          name: newTeamData.name.trim(),
          description: newTeamData.description.trim(),
        }),
      });

      if (response.ok) {
        loadData();
        setShowCreateTeam(false);
        setNewTeamData({ name: '', description: '' });
      }
    } catch (error) {
      console.error('创建团队失败:', error);
    }
  };

  // 邀请用户
  const handleInviteUser = async () => {
    if (!selectedTeam || !inviteData.email.trim()) return;

    try {
      const response = await fetch('/api/v1/invitations', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify({
          team_id: selectedTeam.id,
          email: inviteData.email.trim(),
          role_id: inviteData.role_id,
          message: inviteData.message.trim(),
        }),
      });

      if (response.ok) {
        loadData();
        setShowInviteUser(false);
        setInviteData({ email: '', role_id: 3, message: '' });
      }
    } catch (error) {
      console.error('邀请用户失败:', error);
    }
  };

  // 创建权限申请
  const handleCreateRequest = async () => {
    if (!requestData.reason.trim()) return;

    try {
      const response = await fetch('/api/v1/permission-requests', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify({
          project_id: projectId,
          user_id: user?.id,
          request_type: requestData.request_type,
          permission: requestData.permission,
          reason: requestData.reason.trim(),
          target_id: requestData.target_id,
        }),
      });

      if (response.ok) {
        loadData();
        setShowCreateRequest(false);
        setRequestData({ 
          request_type: 'role', 
          permission: 'read', 
          reason: '', 
          target_id: undefined 
        });
      }
    } catch (error) {
      console.error('创建权限申请失败:', error);
    }
  };

  // 审批权限申请
  const handleReviewRequest = async (requestId: number, approved: boolean, reason: string) => {
    try {
      const response = await fetch(`/api/v1/permission-requests/${requestId}/review`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify({
          approved,
          review_reason: reason,
        }),
      });

      if (response.ok) {
        loadData();
      }
    } catch (error) {
      console.error('审批权限申请失败:', error);
    }
  };

  // 更新成员角色
  const handleUpdateMemberRole = async (teamId: number, userId: number, roleId: number) => {
    try {
      const response = await fetch(`/api/v1/teams/${teamId}/members/${userId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify({ role_id: roleId }),
      });

      if (response.ok) {
        loadData();
      }
    } catch (error) {
      console.error('更新成员角色失败:', error);
    }
  };

  // 移除团队成员
  const handleRemoveMember = async (teamId: number, userId: number) => {
    if (!confirm('确定要移除此成员吗？')) return;

    try {
      const response = await fetch(`/api/v1/teams/${teamId}/members/${userId}`, {
        method: 'DELETE',
        headers: { 'X-Tenant-ID': 'default' },
      });

      if (response.ok) {
        loadData();
      }
    } catch (error) {
      console.error('移除成员失败:', error);
    }
  };

  // 格式化时间
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  // 获取状态标签样式
  const getStatusBadge = (status: string) => {
    const styles: { [key: string]: string } = {
      pending: 'bg-yellow-100 text-yellow-800',
      approved: 'bg-green-100 text-green-800',
      rejected: 'bg-red-100 text-red-800',
      active: 'bg-green-100 text-green-800',
      inactive: 'bg-gray-100 text-gray-800',
      accepted: 'bg-blue-100 text-blue-800',
      expired: 'bg-red-100 text-red-800',
    };
    
    return `px-2 py-1 rounded-full text-xs font-medium ${styles[status] || 'bg-gray-100 text-gray-800'}`;
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 z-50" onClick={onClose}>
      <div 
        className="fixed inset-4 bg-white rounded-2xl shadow-2xl flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* 头部 */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <div className="flex items-center gap-4">
            <h2 className="text-2xl font-bold text-gray-900">👥 团队管理</h2>
            
            {/* 标签页导航 */}
            <nav className="flex bg-gray-100 rounded-lg p-1">
              <button
                onClick={() => setActiveTab('teams')}
                className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  activeTab === 'teams' 
                    ? 'bg-white text-gray-900 shadow-sm' 
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                团队 ({teams.length})
              </button>
              <button
                onClick={() => setActiveTab('invitations')}
                className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  activeTab === 'invitations' 
                    ? 'bg-white text-gray-900 shadow-sm' 
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                邀请 ({invitations.length})
              </button>
              <button
                onClick={() => setActiveTab('requests')}
                className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  activeTab === 'requests' 
                    ? 'bg-white text-gray-900 shadow-sm' 
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                权限申请 ({permissionRequests.length})
              </button>
            </nav>
          </div>
          
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
          >
            ✕
          </button>
        </div>

        {/* 内容区域 */}
        <div className="flex-1 overflow-hidden">
          {loading ? (
            <div className="flex items-center justify-center h-full">
              <div className="loading">加载中...</div>
            </div>
          ) : (
            <>
              {/* 团队管理 */}
              {activeTab === 'teams' && (
                <div className="h-full flex">
                  {/* 团队列表 */}
                  <div className="w-1/3 border-r border-gray-200 flex flex-col">
                    <div className="p-4 border-b border-gray-100">
                      <button
                        onClick={() => setShowCreateTeam(true)}
                        className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                      >
                        ➕ 创建团队
                      </button>
                    </div>
                    
                    <div className="flex-1 overflow-auto">
                      {teams.map((team) => (
                        <div
                          key={team.id}
                          className={`p-4 border-b border-gray-100 cursor-pointer hover:bg-gray-50 transition-colors ${
                            selectedTeam?.id === team.id ? 'bg-blue-50 border-l-4 border-l-blue-500' : ''
                          }`}
                          onClick={() => setSelectedTeam(team)}
                        >
                          <div className="font-medium text-gray-900">{team.name}</div>
                          <div className="text-sm text-gray-600 mt-1">{team.description}</div>
                          <div className="flex items-center gap-2 mt-2 text-xs text-gray-500">
                            <span>👥 {team.members.length} 成员</span>
                            <span>📅 {formatDate(team.created_at)}</span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* 团队详情 */}
                  <div className="flex-1 flex flex-col">
                    {selectedTeam ? (
                      <>
                        <div className="p-6 border-b border-gray-200">
                          <div className="flex items-center justify-between">
                            <div>
                              <h3 className="text-xl font-semibold text-gray-900">{selectedTeam.name}</h3>
                              <p className="text-gray-600 mt-1">{selectedTeam.description}</p>
                            </div>
                            <button
                              onClick={() => setShowInviteUser(true)}
                              className="flex items-center gap-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors"
                            >
                              📧 邀请成员
                            </button>
                          </div>
                        </div>

                        <div className="flex-1 overflow-auto p-6">
                          <h4 className="text-lg font-medium text-gray-900 mb-4">团队成员</h4>
                          <div className="space-y-4">
                            {selectedTeam.members.map((member) => (
                              <div key={member.id} className="flex items-center justify-between p-4 border border-gray-200 rounded-lg">
                                <div className="flex items-center gap-4">
                                  <div className="w-10 h-10 bg-gray-300 rounded-full flex items-center justify-center">
                                    {member.user.avatar ? (
                                      <img src={member.user.avatar} alt={member.user.display_name} className="w-10 h-10 rounded-full" />
                                    ) : (
                                      <span className="text-gray-600 font-medium">
                                        {member.user.display_name.charAt(0)}
                                      </span>
                                    )}
                                  </div>
                                  <div>
                                    <div className="font-medium text-gray-900">{member.user.display_name}</div>
                                    <div className="text-sm text-gray-600">{member.user.email}</div>
                                    <div className="text-xs text-gray-500 mt-1">
                                      {member.user.department} · {member.user.position}
                                    </div>
                                  </div>
                                </div>
                                
                                <div className="flex items-center gap-3">
                                  <select
                                    value={member.role_id}
                                    onChange={(e) => handleUpdateMemberRole(selectedTeam.id, member.user_id, parseInt(e.target.value))}
                                    className="px-3 py-1 border border-gray-300 rounded text-sm focus:ring-2 focus:ring-blue-500"
                                  >
                                    {roles.map((role) => (
                                      <option key={role.id} value={role.id}>
                                        {role.name}
                                      </option>
                                    ))}
                                  </select>
                                  
                                  <span className={getStatusBadge(member.status)}>
                                    {member.status}
                                  </span>
                                  
                                  <button
                                    onClick={() => handleRemoveMember(selectedTeam.id, member.user_id)}
                                    className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                                    title="移除成员"
                                  >
                                    🗑️
                                  </button>
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>
                      </>
                    ) : (
                      <div className="flex-1 flex items-center justify-center">
                        <div className="text-center text-gray-500">
                          <div className="text-4xl mb-4">👥</div>
                          <div className="text-lg">选择一个团队查看详情</div>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* 邀请管理 */}
              {activeTab === 'invitations' && (
                <div className="p-6 h-full overflow-auto">
                  <div className="flex items-center justify-between mb-6">
                    <h3 className="text-xl font-semibold text-gray-900">团队邀请管理</h3>
                  </div>

                  <div className="space-y-4">
                    {invitations.map((invitation) => (
                      <div key={invitation.id} className="p-4 border border-gray-200 rounded-lg">
                        <div className="flex items-center justify-between">
                          <div>
                            <div className="font-medium text-gray-900">{invitation.email}</div>
                            <div className="text-sm text-gray-600 mt-1">{invitation.message}</div>
                            <div className="flex items-center gap-4 mt-2 text-xs text-gray-500">
                              <span>📅 {formatDate(invitation.created_at)}</span>
                              <span>⏰ 过期: {formatDate(invitation.expires_at)}</span>
                            </div>
                          </div>
                          
                          <div className="flex items-center gap-3">
                            <span className={getStatusBadge(invitation.status)}>
                              {invitation.status}
                            </span>
                          </div>
                        </div>
                      </div>
                    ))}
                    
                    {invitations.length === 0 && (
                      <div className="text-center py-16 text-gray-500">
                        <div className="text-4xl mb-4">📧</div>
                        <div className="text-lg">暂无邀请记录</div>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* 权限申请 */}
              {activeTab === 'requests' && (
                <div className="p-6 h-full overflow-auto">
                  <div className="flex items-center justify-between mb-6">
                    <h3 className="text-xl font-semibold text-gray-900">权限申请管理</h3>
                    <button
                      onClick={() => setShowCreateRequest(true)}
                      className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                    >
                      ➕ 申请权限
                    </button>
                  </div>

                  <div className="space-y-4">
                    {permissionRequests.map((request) => {
                      const requestUser = users.find(u => u.id === request.user_id);
                      return (
                        <div key={request.id} className="p-4 border border-gray-200 rounded-lg">
                          <div className="flex items-start justify-between">
                            <div className="flex-1">
                              <div className="flex items-center gap-2 mb-2">
                                <span className="font-medium text-gray-900">
                                  {requestUser?.display_name || '未知用户'}
                                </span>
                                <span className="text-gray-600">申请</span>
                                <span className="font-medium text-blue-600">{request.permission}</span>
                                <span className="text-gray-600">权限</span>
                              </div>
                              
                              <div className="text-sm text-gray-600 mb-2">
                                <strong>类型:</strong> {request.request_type}
                              </div>
                              
                              <div className="text-sm text-gray-600 mb-2">
                                <strong>理由:</strong> {request.reason}
                              </div>
                              
                              <div className="text-xs text-gray-500">
                                📅 {formatDate(request.created_at)}
                              </div>
                            </div>
                            
                            <div className="flex items-center gap-3">
                              <span className={getStatusBadge(request.status)}>
                                {request.status}
                              </span>
                              
                              {request.status === 'pending' && (
                                <div className="flex gap-2">
                                  <button
                                    onClick={() => handleReviewRequest(request.id, true, '申请已批准')}
                                    className="px-3 py-1 bg-green-600 text-white text-sm rounded hover:bg-green-700 transition-colors"
                                  >
                                    批准
                                  </button>
                                  <button
                                    onClick={() => handleReviewRequest(request.id, false, '申请被拒绝')}
                                    className="px-3 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700 transition-colors"
                                  >
                                    拒绝
                                  </button>
                                </div>
                              )}
                            </div>
                          </div>
                        </div>
                      );
                    })}
                    
                    {permissionRequests.length === 0 && (
                      <div className="text-center py-16 text-gray-500">
                        <div className="text-4xl mb-4">🔔</div>
                        <div className="text-lg">暂无权限申请</div>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* 创建团队模态框 */}
      {showCreateTeam && (
        <div className="fixed inset-0 bg-black bg-opacity-50 z-60 flex items-center justify-center">
          <div className="bg-white rounded-lg p-6 w-96">
            <h3 className="text-lg font-semibold mb-4">创建新团队</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  团队名称
                </label>
                <input
                  type="text"
                  value={newTeamData.name}
                  onChange={(e) => setNewTeamData(prev => ({ ...prev, name: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  placeholder="输入团队名称..."
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  团队描述
                </label>
                <textarea
                  value={newTeamData.description}
                  onChange={(e) => setNewTeamData(prev => ({ ...prev, description: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  rows={3}
                  placeholder="输入团队描述..."
                />
              </div>
            </div>
            <div className="flex gap-3 mt-6">
              <button
                onClick={handleCreateTeam}
                disabled={!newTeamData.name.trim()}
                className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                创建
              </button>
              <button
                onClick={() => {
                  setShowCreateTeam(false);
                  setNewTeamData({ name: '', description: '' });
                }}
                className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
              >
                取消
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 邀请用户模态框 */}
      {showInviteUser && selectedTeam && (
        <div className="fixed inset-0 bg-black bg-opacity-50 z-60 flex items-center justify-center">
          <div className="bg-white rounded-lg p-6 w-96">
            <h3 className="text-lg font-semibold mb-4">邀请用户加入团队</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  邮箱地址
                </label>
                <input
                  type="email"
                  value={inviteData.email}
                  onChange={(e) => setInviteData(prev => ({ ...prev, email: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  placeholder="输入用户邮箱..."
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  分配角色
                </label>
                <select
                  value={inviteData.role_id}
                  onChange={(e) => setInviteData(prev => ({ ...prev, role_id: parseInt(e.target.value) }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                >
                  {roles.map((role) => (
                    <option key={role.id} value={role.id}>
                      {role.name} - {role.description}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  邀请消息 (可选)
                </label>
                <textarea
                  value={inviteData.message}
                  onChange={(e) => setInviteData(prev => ({ ...prev, message: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  rows={3}
                  placeholder="输入邀请消息..."
                />
              </div>
            </div>
            <div className="flex gap-3 mt-6">
              <button
                onClick={handleInviteUser}
                disabled={!inviteData.email.trim()}
                className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                发送邀请
              </button>
              <button
                onClick={() => {
                  setShowInviteUser(false);
                  setInviteData({ email: '', role_id: 3, message: '' });
                }}
                className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
              >
                取消
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 创建权限申请模态框 */}
      {showCreateRequest && (
        <div className="fixed inset-0 bg-black bg-opacity-50 z-60 flex items-center justify-center">
          <div className="bg-white rounded-lg p-6 w-96">
            <h3 className="text-lg font-semibold mb-4">申请权限</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  申请类型
                </label>
                <select
                  value={requestData.request_type}
                  onChange={(e) => setRequestData(prev => ({ ...prev, request_type: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                >
                  <option value="role">角色权限</option>
                  <option value="file_permission">文件权限</option>
                  <option value="folder_permission">文件夹权限</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  申请权限
                </label>
                <select
                  value={requestData.permission}
                  onChange={(e) => setRequestData(prev => ({ ...prev, permission: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                >
                  <option value="read">读取权限</option>
                  <option value="write">写入权限</option>
                  <option value="delete">删除权限</option>
                  <option value="share">分享权限</option>
                  <option value="admin">管理权限</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  申请理由
                </label>
                <textarea
                  value={requestData.reason}
                  onChange={(e) => setRequestData(prev => ({ ...prev, reason: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  rows={3}
                  placeholder="请详细说明申请理由..."
                />
              </div>
            </div>
            <div className="flex gap-3 mt-6">
              <button
                onClick={handleCreateRequest}
                disabled={!requestData.reason.trim()}
                className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                提交申请
              </button>
              <button
                onClick={() => {
                  setShowCreateRequest(false);
                  setRequestData({ 
                    request_type: 'role', 
                    permission: 'read', 
                    reason: '', 
                    target_id: undefined 
                  });
                }}
                className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
              >
                取消
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default TeamModal;