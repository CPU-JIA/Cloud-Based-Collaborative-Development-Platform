import React, { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate } from 'react-router-dom';
import { Project, CreateProjectRequest, UpdateProjectRequest } from '../types';
import { projectApi } from '../utils/api';
import ProjectModal from '../components/ProjectModal';
import '../styles/modern-enterprise.css';
import '../styles/premium-dashboard.css';

const Dashboard: React.FC = () => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  
  // 项目管理状态
  const [isProjectModalOpen, setIsProjectModalOpen] = useState(false);
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');

  useEffect(() => {
    loadProjects();
  }, []);

  const loadProjects = async () => {
    try {
      setLoading(true);
      const projectList = await projectApi.list();
      setProjects(projectList);
    } catch (err: any) {
      setError(err.message || '加载项目失败');
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = async () => {
    try {
      await logout();
      navigate('/login');
    } catch (error) {
      console.error('登出失败:', error);
    }
  };
  
  // 项目管理函数
  const handleCreateProject = async (data: CreateProjectRequest) => {
    try {
      const newProject = await projectApi.create(data);
      setProjects(prev => [...prev, newProject]);
      setIsProjectModalOpen(false);
    } catch (error: any) {
      console.error('创建项目失败:', error);
      throw error;
    }
  };

  const handleEditProject = (project: Project) => {
    setEditingProject(project);
    setIsProjectModalOpen(true);
  };

  const handleUpdateProject = async (data: UpdateProjectRequest) => {
    if (!editingProject) return;
    
    try {
      const updatedProject = await projectApi.update(editingProject.id.toString(), data);
      setProjects(prev => 
        prev.map(p => p.id === editingProject.id ? updatedProject : p)
      );
      setIsProjectModalOpen(false);
      setEditingProject(null);
    } catch (error: any) {
      console.error('更新项目失败:', error);
      throw error;
    }
  };

  const handleDeleteProject = async (project: Project) => {
    if (!confirm(`确定要删除项目「${project.name}」吗？此操作不可撤销。`)) {
      return;
    }

    try {
      await projectApi.delete(project.id.toString());
      setProjects(prev => prev.filter(p => p.id !== project.id));
    } catch (error: any) {
      console.error('删除项目失败:', error);
      alert('删除项目失败：' + error.message);
    }
  };

  const handleProjectClick = (project: Project) => {
    navigate(`/board/${project.id}`);
  };

  const getProjectStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'success';
      case 'planning': return 'warning';
      case 'completed': return 'primary';
      case 'paused': return 'secondary';
      default: return 'secondary';
    }
  };

  const getProjectStatusText = (status: string) => {
    switch (status) {
      case 'active': return '进行中';
      case 'planning': return '规划中';
      case 'completed': return '已完成';
      case 'paused': return '已暂停';
      default: return '未知';
    }
  };

  if (loading) {
    return (
      <div className="premium-dashboard-container">
        <div className="premium-loading-state">
          <div className="premium-loading-spinner-large"></div>
          <p>正在加载您的项目...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="premium-dashboard-container">
      {/* 顶级导航栏 */}
      <header className="premium-dashboard-header">
        <div className="premium-header-content">
          <div className="premium-header-left">
            <div className="premium-logo-section">
              <div className="premium-logo-icon">📋</div>
              <h1 className="premium-logo-text">CloudDev</h1>
            </div>
            
            <nav className="premium-nav-tabs">
              <button className="premium-nav-tab active">
                <span>🏠</span>
                <span>仪表板</span>
              </button>
              <button className="premium-nav-tab">
                <span>👥</span>
                <span>团队</span>
              </button>
              <button className="premium-nav-tab">
                <span>📊</span>
                <span>分析</span>
              </button>
              <button className="premium-nav-tab">
                <span>⚙️</span>
                <span>设置</span>
              </button>
            </nav>
          </div>
          
          <div className="premium-header-right">
            <div className="premium-search-section">
              <div className="premium-search-bar">
                <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z" clipRule="evenodd"/>
                </svg>
                <input type="text" placeholder="搜索项目、任务..." />
              </div>
            </div>
            
            <div className="premium-header-actions">
              <button className="premium-notification-btn">
                <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                  <path d="M10 2a6 6 0 00-6 6c0 1.887-.454 3.665-1.257 5.234a.75.75 0 00.515 1.076 32.91 32.91 0 003.256.508 3.5 3.5 0 006.972 0 32.91 32.91 0 003.256-.508.75.75 0 00.515-1.076A11.448 11.448 0 0116 8a6 6 0 00-6-6zM8.05 14.943a33.54 33.54 0 003.9 0 2 2 0 01-3.9 0z"/>
                </svg>
                <span className="premium-notification-badge">3</span>
              </button>
              
              <div className="premium-user-menu">
                <div className="premium-user-profile" onClick={handleLogout}>
                  <div className="premium-user-avatar">
                    {(user?.display_name || user?.username || 'U').charAt(0).toUpperCase()}
                  </div>
                  <div className="premium-user-info">
                    <span className="premium-user-name">{user?.display_name || user?.username}</span>
                    <span className="premium-user-role">管理员</span>
                  </div>
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                    <path fillRule="evenodd" d="M4.22 6.22a.75.75 0 011.06 0L8 8.94l2.72-2.72a.75.75 0 111.06 1.06l-3.25 3.25a.75.75 0 01-1.06 0L4.22 7.28a.75.75 0 010-1.06z" clipRule="evenodd"/>
                  </svg>
                </div>
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* 主要内容区域 */}
      <main className="premium-dashboard-main">
        {/* 欢迎横幅 */}
        <section className="premium-welcome-section">
          <div className="premium-welcome-content">
            <div className="premium-welcome-text">
              <h2 className="premium-welcome-title">
                欢迎回来，{user?.display_name || user?.username}! 👋
              </h2>
              <p className="premium-welcome-subtitle">
                您有 {projects.filter(p => p.status === 'active').length} 个活跃项目正在进行中
              </p>
            </div>
            
            <div className="premium-quick-stats">
              <div className="premium-stat-card">
                <div className="premium-stat-icon success">📊</div>
                <div className="premium-stat-content">
                  <span className="premium-stat-number">{projects.length}</span>
                  <span className="premium-stat-label">总项目数</span>
                </div>
              </div>
              
              <div className="premium-stat-card">
                <div className="premium-stat-icon warning">⚡</div>
                <div className="premium-stat-content">
                  <span className="premium-stat-number">
                    {projects.filter(p => p.status === 'active').length}
                  </span>
                  <span className="premium-stat-label">进行中</span>
                </div>
              </div>
              
              <div className="premium-stat-card">
                <div className="premium-stat-icon primary">✅</div>
                <div className="premium-stat-content">
                  <span className="premium-stat-number">
                    {projects.filter(p => p.status === 'completed').length}
                  </span>
                  <span className="premium-stat-label">已完成</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* 项目区域 */}
        <section className="premium-projects-section">
          <div className="premium-section-header">
            <div className="premium-section-title">
              <h3>我的项目</h3>
              <span className="premium-section-count">{projects.length} 个项目</span>
            </div>
            
            <div className="premium-section-actions">
              <div className="premium-view-toggle">
                <button 
                  className={`premium-view-btn ${viewMode === 'grid' ? 'active' : ''}`}
                  onClick={() => setViewMode('grid')}
                >
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                    <path d="M1 2.5A1.5 1.5 0 012.5 1h3A1.5 1.5 0 017 2.5v3A1.5 1.5 0 015.5 7h-3A1.5 1.5 0 011 5.5v-3zM2.5 2a.5.5 0 00-.5.5v3a.5.5 0 00.5.5h3a.5.5 0 00.5-.5v-3a.5.5 0 00-.5-.5h-3zm6.5.5A1.5 1.5 0 0110.5 1h3A1.5 1.5 0 0115 2.5v3A1.5 1.5 0 0113.5 7h-3A1.5 1.5 0 019 5.5v-3zm1.5-.5a.5.5 0 00-.5.5v3a.5.5 0 00.5.5h3a.5.5 0 00.5-.5v-3a.5.5 0 00-.5-.5h-3zM1 10.5A1.5 1.5 0 012.5 9h3A1.5 1.5 0 017 10.5v3A1.5 1.5 0 015.5 15h-3A1.5 1.5 0 011 13.5v-3zm1.5-.5a.5.5 0 00-.5.5v3a.5.5 0 00.5.5h3a.5.5 0 00.5-.5v-3a.5.5 0 00-.5-.5h-3zm6.5.5A1.5 1.5 0 0110.5 9h3a1.5 1.5 0 011.5 1.5v3a1.5 1.5 0 01-1.5 1.5h-3A1.5 1.5 0 019 13.5v-3zm1.5-.5a.5.5 0 00-.5.5v3a.5.5 0 00.5.5h3a.5.5 0 00.5-.5v-3a.5.5 0 00-.5-.5h-3z"/>
                  </svg>
                </button>
                <button 
                  className={`premium-view-btn ${viewMode === 'list' ? 'active' : ''}`}
                  onClick={() => setViewMode('list')}
                >
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                    <path d="M2.5 12a.5.5 0 01.5-.5h10a.5.5 0 010 1H3a.5.5 0 01-.5-.5zm0-4a.5.5 0 01.5-.5h10a.5.5 0 010 1H3a.5.5 0 01-.5-.5zm0-4a.5.5 0 01.5-.5h10a.5.5 0 010 1H3a.5.5 0 01-.5-.5z"/>
                  </svg>
                </button>
              </div>
              
              <button 
                className="premium-create-project-btn"
                onClick={() => setIsProjectModalOpen(true)}
              >
                <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                  <path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z"/>
                </svg>
                <span>创建项目</span>
              </button>
            </div>
          </div>

          {/* 错误状态 */}
          {error && (
            <div className="premium-error-banner">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd"/>
              </svg>
              <span>{error}</span>
              <button onClick={loadProjects} className="premium-retry-btn">重试</button>
            </div>
          )}

          {/* 项目网格/列表 */}
          {projects.length === 0 ? (
            <div className="premium-empty-state">
              <div className="premium-empty-icon">📋</div>
              <h3 className="premium-empty-title">还没有项目</h3>
              <p className="premium-empty-description">
                创建您的第一个项目，开始协作开发之旅
              </p>
              <button 
                className="premium-create-project-btn primary"
                onClick={() => setIsProjectModalOpen(true)}
              >
                <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                  <path d="M10.75 4.75a.75.75 0 00-1.5 0v4.5h-4.5a.75.75 0 000 1.5h4.5v4.5a.75.75 0 001.5 0v-4.5h4.5a.75.75 0 000-1.5h-4.5v-4.5z"/>
                </svg>
                <span>创建第一个项目</span>
              </button>
            </div>
          ) : (
            <div className={`premium-projects-container ${viewMode}`}>
              {projects.map((project) => (
                <div 
                  key={project.id} 
                  className="premium-project-card"
                  onClick={() => handleProjectClick(project)}
                >
                  <div className="premium-card-header">
                    <div className="premium-project-icon">📁</div>
                    <div className="premium-card-actions">
                      <button 
                        className="premium-card-action-btn"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleEditProject(project);
                        }}
                      >
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                          <path d="m13.498.795.149-.149a1.207 1.207 0 1 1 1.707 1.708l-.149.148a1.5 1.5 0 0 1-.059 2.059L4.854 14.854a.5.5 0 0 1-.233.131l-4 1a.5.5 0 0 1-.606-.606l1-4a.5.5 0 0 1 .131-.232l9.642-9.642a.5.5 0 0 0-.642.056L6.854 4.854a.5.5 0 1 1-.708-.708L9.44.854A1.5 1.5 0 0 1 11.5.796a1.5 1.5 0 0 1 1.998-.001z"/>
                        </svg>
                      </button>
                      <button 
                        className="premium-card-action-btn danger"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteProject(project);
                        }}
                      >
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                          <path d="M6.5 1h3a.5.5 0 0 1 .5.5v1H6v-1a.5.5 0 0 1 .5-.5ZM11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3A1.5 1.5 0 0 0 5 1.5v1H2.506a.58.58 0 0 0-.01 0H1.5a.5.5 0 0 0 0 1h.538l.853 10.66A2 2 0 0 0 4.885 16h6.23a2 2 0 0 0 1.994-1.84L13.962 3.5H14.5a.5.5 0 0 0 0-1h-1.004a.58.58 0 0 0-.01 0H11Zm1.958 1-.846 10.58a1 1 0 0 1-.997.92h-6.23a1 1 0 0 1-.997-.92L3.042 3.5h9.916Zm-7.487 1a.5.5 0 0 1 .528.47l.5 8.5a.5.5 0 0 1-.998.06L5 5.03a.5.5 0 0 1 .47-.53Zm5.058 0a.5.5 0 0 1 .47.53l-.5 8.5a.5.5 0 1 1-.998-.06l.5-8.5a.5.5 0 0 1 .528-.47ZM8 4.5a.5.5 0 0 1 .5.5v8.5a.5.5 0 0 1-1 0V5a.5.5 0 0 1 .5-.5Z"/>
                        </svg>
                      </button>
                    </div>
                  </div>
                  
                  <div className="premium-card-content">
                    <h4 className="premium-project-title">{project.name}</h4>
                    <p className="premium-project-key">{project.key}</p>
                    <p className="premium-project-description">
                      {project.description || '暂无项目描述'}
                    </p>
                  </div>
                  
                  <div className="premium-card-footer">
                    <div className="premium-project-meta">
                      <span className={`premium-status-badge ${getProjectStatusColor(project.status)}`}>
                        {getProjectStatusText(project.status)}
                      </span>
                      <span className="premium-meta-item">
                        <svg width="14" height="14" viewBox="0 0 14 14" fill="currentColor">
                          <path d="M7 7a3 3 0 100-6 3 3 0 000 6zM14 12a7 7 0 10-14 0h14z"/>
                        </svg>
                        {project.team_size || 1}人
                      </span>
                      <span className="premium-meta-item">
                        <svg width="14" height="14" viewBox="0 0 14 14" fill="currentColor">
                          <path d="M8 0a1 1 0 0 1 1 1v5.268l4.562 2.634a1 1 0 1 1-1 1.732L8 8.732V1a1 1 0 0 1 1-1z"/>
                        </svg>
                        {project.tasks_count || 0}个任务
                      </span>
                    </div>
                    
                    <div className="premium-project-progress">
                      <div className="premium-progress-bar">
                        <div 
                          className="premium-progress-fill" 
                          style={{ width: `${Math.random() * 100}%` }}
                        ></div>
                      </div>
                      <span className="premium-progress-text">
                        {Math.floor(Math.random() * 100)}%
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>
      </main>

      {/* 项目创建/编辑对话框 */}
      <ProjectModal
        isOpen={isProjectModalOpen}
        onClose={() => {
          setIsProjectModalOpen(false);
          setEditingProject(null);
        }}
        onSubmit={editingProject ? handleUpdateProject : handleCreateProject}
        project={editingProject}
        title={editingProject ? '编辑项目' : '创建新项目'}
      />
    </div>
  );
};

export default Dashboard;