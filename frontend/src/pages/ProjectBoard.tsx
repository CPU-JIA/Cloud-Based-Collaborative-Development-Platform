import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { Task, TaskStatus, Project, CreateTaskRequest, UpdateTaskRequest } from '../types';
import { taskApi, projectApi } from '../utils/api';
import TaskModal from '../components/TaskModal';
import ChatWidget from '../components/ChatWidget';
import FileManager from '../components/FileManager';
import TeamModal from '../components/TeamModal';
import { useWebSocket, useTaskUpdates, useOnlineUsers } from '../hooks/useWebSocket';
import { ConnectionStatus, MessageType } from '../utils/websocket';
import '../styles/modern-enterprise.css';
import '../styles/premium-tasks.css';
import '../styles/websocket.css';

const ProjectBoard: React.FC = () => {
  const { projectId } = useParams<{ projectId: string }>();
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  
  const [project, setProject] = useState<Project | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [taskStatuses] = useState<TaskStatus[]>([
    { id: '1', tenant_id: '', name: '待办', category: 'todo', display_order: 1 },
    { id: '2', tenant_id: '', name: '进行中', category: 'in_progress', display_order: 2 },
    { id: '3', tenant_id: '', name: '已完成', category: 'done', display_order: 3 },
  ]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  
  // 任务管理状态
  const [isTaskModalOpen, setIsTaskModalOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<Task | null>(null);
  const [currentStatusId, setCurrentStatusId] = useState<string>('1');
  
  // 聊天组件状态
  const [isChatOpen, setIsChatOpen] = useState(false);
  
  // 文件管理器状态
  const [isFileManagerOpen, setIsFileManagerOpen] = useState(false);
  
  // 团队管理状态
  const [isTeamModalOpen, setIsTeamModalOpen] = useState(false);
  
  // WebSocket消息处理
  const handleWebSocketMessage = useCallback((message: any) => {
    console.log('📨 收到WebSocket消息:', message);
    
    switch (message.type) {
      case MessageType.TASK_UPDATE:
        // 实时更新任务状态
        setTasks(prevTasks => 
          prevTasks.map(task => 
            task.id === message.data.task_id 
              ? { ...task, ...message.data }
              : task
          )
        );
        break;
        
      case MessageType.TASK_CREATE:
        // 实时添加新任务
        if (message.data && message.user_id !== user?.id) {
          loadProjectData(); // 重新加载数据以获取完整的任务信息
        }
        break;
        
      case MessageType.TASK_DELETE:
        // 实时删除任务
        setTasks(prevTasks => 
          prevTasks.filter(task => task.id !== message.data.task_id)
        );
        break;
        
      default:
        break;
    }
  }, [user?.id]);

  // WebSocket集成
  const projectIdNum = parseInt(projectId || '0');
  const { 
    isConnected, 
    status: wsStatus, 
    sendTaskUpdate, 
    sendTaskCreate, 
    sendTaskDelete,
    sendChatMessage 
  } = useWebSocket({
    projectId: projectIdNum,
    onMessage: handleWebSocketMessage
  });
  
  const onlineUsers = useOnlineUsers(projectIdNum);

  useEffect(() => {
    if (projectId) {
      loadProjectData();
    }
  }, [projectId]);

  const loadProjectData = async () => {
    try {
      setLoading(true);
      const [projectData, taskList] = await Promise.all([
        projectApi.getById(projectId!),
        taskApi.list(projectId!)
      ]);
      setProject(projectData);
      setTasks(taskList);
    } catch (err: any) {
      setError(err.message || '加载项目数据失败');
    } finally {
      setLoading(false);
    }
  };

  const getTasksByStatus = (statusId: string) => {
    return tasks.filter(task => task.status_id === statusId);
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'urgent': return 'bg-red-100 text-red-800';
      case 'high': return 'bg-orange-100 text-orange-800';
      case 'medium': return 'bg-yellow-100 text-yellow-800';
      case 'low': return 'bg-green-100 text-green-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getPriorityLabel = (priority: string) => {
    switch (priority) {
      case 'urgent': return '紧急';
      case 'high': return '高';
      case 'medium': return '中';
      case 'low': return '低';
      default: return '未知';
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
  
  // 任务管理函数
  const handleCreateTask = async (data: CreateTaskRequest) => {
    try {
      const newTask = await taskApi.create(projectId!, data);
      setTasks(prev => [...prev, newTask]);
      
      // 发送WebSocket通知
      if (isConnected) {
        sendTaskCreate({
          task_id: newTask.id,
          title: newTask.title,
          description: newTask.description,
          status_id: newTask.status_id,
          priority: newTask.priority,
          assignee_id: newTask.assignee_id,
          due_date: newTask.due_date
        });
      }
      
      console.log('任务创建成功:', newTask);
    } catch (error: any) {
      console.error('创建任务失败:', error);
      throw error;
    }
  };
  
  const handleEditTask = (task: Task) => {
    setEditingTask(task);
    setIsTaskModalOpen(true);
  };
  
  const handleUpdateTask = async (data: UpdateTaskRequest) => {
    if (!editingTask) return;
    
    try {
      const updatedTask = await taskApi.update(editingTask.id.toString(), data);
      setTasks(prev => prev.map(t => t.id === editingTask.id ? updatedTask : t));
      
      // 发送WebSocket通知
      if (isConnected) {
        sendTaskUpdate({
          task_id: updatedTask.id,
          title: updatedTask.title,
          description: updatedTask.description,
          status_id: updatedTask.status_id,
          priority: updatedTask.priority,
          assignee_id: updatedTask.assignee_id,
          due_date: updatedTask.due_date
        });
      }
      
      setEditingTask(null);
      console.log('任务更新成功:', updatedTask);
    } catch (error: any) {
      console.error('更新任务失败:', error);
      throw error;
    }
  };
  
  const handleDeleteTask = async (task: Task) => {
    if (!confirm(`确定要删除任务"${task.title}"？`)) {
      return;
    }
    
    try {
      await taskApi.delete(task.id.toString());
      setTasks(prev => prev.filter(t => t.id !== task.id));
      
      // 发送WebSocket通知
      if (isConnected) {
        sendTaskDelete(task.id);
      }
      
      console.log('任务删除成功:', task.id);
    } catch (error: any) {
      console.error('删除任务失败:', error);
      alert('删除失败：' + (error.message || '网络错误'));
    }
  };
  
  const handleAddTaskClick = (statusId: string) => {
    setCurrentStatusId(statusId);
    setEditingTask(null);
    setIsTaskModalOpen(true);
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="loading"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="text-red-600 mb-4">{error}</div>
          <button 
            onClick={() => navigate('/dashboard')}
            className="btn btn-primary"
          >
            返回主页
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="premium-task-board">
      {/* 顶级导航栏 */}
      <header className="premium-navbar">
        <div className="premium-navbar-content">
          <div className="premium-nav-brand">
            <button
              onClick={() => navigate('/dashboard')}
              className="premium-action-btn edit"
              style={{ marginRight: '1rem' }}
            >
              ←
            </button>
            <div className="premium-nav-logo">
              📋
            </div>
            <div>
              <div style={{ fontWeight: 800 }}>{project?.name || '项目看板'}</div>
              <div style={{ fontSize: '0.9rem', opacity: 0.7 }}>{project?.key}</div>
            </div>
          </div>
          
          <div className="premium-nav-actions">
            {/* WebSocket连接状态 */}
            <div className="premium-connection-status">
              <div className={`premium-status-indicator ${isConnected ? 'connected' : 'disconnected'}`}>
                <div className="premium-status-dot"></div>
                <span className="premium-status-text">
                  {wsStatus === ConnectionStatus.CONNECTED ? '已连接' : 
                   wsStatus === ConnectionStatus.CONNECTING ? '连接中...' : 
                   wsStatus === ConnectionStatus.RECONNECTING ? '重连中...' : '未连接'}
                </span>
              </div>
            </div>
            
            {/* 在线用户显示 */}
            {onlineUsers.length > 0 && (
              <div className="premium-online-users">
                <div className="premium-users-avatars">
                  {onlineUsers.slice(0, 3).map((user, index) => (
                    <div 
                      key={user.user_id} 
                      className="premium-user-avatar online"
                      style={{ marginLeft: index > 0 ? '-8px' : '0' }}
                      title={`${user.username} - ${user.status}`}
                    >
                      {user.avatar ? (
                        <img src={user.avatar} alt={user.username} />
                      ) : (
                        user.username.charAt(0).toUpperCase()
                      )}
                    </div>
                  ))}
                  {onlineUsers.length > 3 && (
                    <div className="premium-user-avatar more" style={{ marginLeft: '-8px' }}>
                      +{onlineUsers.length - 3}
                    </div>
                  )}
                </div>
                <span className="premium-online-count">{onlineUsers.length} 人在线</span>
              </div>
            )}
            
            <div className="premium-user-profile" onClick={handleLogout}>
              <div className="premium-user-avatar">
                {(user?.display_name || user?.username || 'U').charAt(0).toUpperCase()}
              </div>
              <span>{user?.display_name || user?.username}</span>
            </div>
          </div>
        </div>
      </header>

      {/* 主要内容区 */}
      <div className="premium-main-content">
        {/* 项目头部 */}
        <div className="premium-project-header">
          <h1 className="premium-project-title">{project?.name || '项目看板'}</h1>
          <div className="premium-project-meta">
            <div className="premium-status-badge active">进行中</div>
            <div className="premium-meta-item">
              <span>🔑</span>
              <span>{project?.key}</span>
            </div>
            <div className="premium-meta-item">
              <span>📅</span>
              <span>{new Date().toLocaleDateString()}</span>
            </div>
          </div>
          {project?.description && (
            <p className="premium-project-description">{project.description}</p>
          )}
        </div>

        {/* 任务操作区 */}
        <div className="premium-task-actions">
          <div className="premium-task-stats">
            {taskStatuses.map((status) => {
              const statusTasks = getTasksByStatus(status.id);
              return (
                <div key={status.id} className="premium-stat-item">
                  <div className={`premium-stat-number ${status.category === 'done' ? 'completed' : status.category === 'in_progress' ? 'in-progress' : 'pending'}`}>
                    {statusTasks.length}
                  </div>
                  <div className="premium-stat-label">{status.name}</div>
                </div>
              );
            })}
            <div className="premium-stat-item">
              <div className="premium-stat-number total">{tasks.length}</div>
              <div className="premium-stat-label">总任务数</div>
            </div>
          </div>
          
          <div className="flex gap-3">
            <button 
              className="premium-add-task-btn"
              onClick={() => handleAddTaskClick('1')}
            >
              <span>+</span>
              <span>创建新任务</span>
            </button>
            
            <button 
              className="premium-add-task-btn"
              style={{ background: 'linear-gradient(135deg, #10b981, #059669)' }}
              onClick={() => setIsFileManagerOpen(true)}
            >
              <span>📁</span>
              <span>文件管理</span>
            </button>
            
            <button 
              className="premium-add-task-btn"
              style={{ background: 'linear-gradient(135deg, #8b5cf6, #7c3aed)' }}
              onClick={() => setIsTeamModalOpen(true)}
            >
              <span>👥</span>
              <span>团队管理</span>
            </button>
          </div>
        </div>

        {/* 任务看板区域 */}
        <div className="premium-task-columns">
          {taskStatuses.map((status) => {
            const statusTasks = getTasksByStatus(status.id);
            const columnClass = status.category === 'todo' ? 'todo' : status.category === 'in_progress' ? 'in-progress' : 'completed';
            
            return (
              <div key={status.id} className="premium-task-column">
                {/* 列标题 */}
                <div className={`premium-column-header ${columnClass}`}>
                  <div className="premium-column-title">
                    <div className={`premium-column-icon ${columnClass}`}>
                      {status.category === 'todo' ? '📋' : status.category === 'in_progress' ? '⚡' : '✅'}
                    </div>
                    <span>{status.name}</span>
                  </div>
                  <div className="premium-task-count">{statusTasks.length}</div>
                </div>

                {/* 任务列表 */}
                <div className="premium-task-list">
                  {statusTasks.length === 0 ? (
                    <div className="premium-empty-state">
                      <div className="premium-empty-icon">
                        {status.category === 'todo' ? '📝' : status.category === 'in_progress' ? '⚡' : '🎉'}
                      </div>
                      <p className="premium-empty-text">暂无任务</p>
                    </div>
                  ) : (
                    statusTasks.map((task) => (
                      <div
                        key={task.id}
                        className="premium-task-card"
                        onClick={() => handleEditTask(task)}
                      >
                        <h4 className="premium-task-title">{task.title}</h4>
                        
                        {task.description && (
                          <p className="premium-task-description">{task.description}</p>
                        )}
                        
                        <div className="premium-task-meta">
                          <div className={`premium-task-priority ${task.priority}`}>
                            {getPriorityLabel(task.priority)}
                          </div>
                          
                          {task.due_date && (
                            <div className="premium-task-date">
                              <span>📅</span>
                              <span>{new Date(task.due_date).toLocaleDateString()}</span>
                            </div>
                          )}
                        </div>
                        
                        <div className="premium-task-actions-mini">
                          <button
                            className="premium-action-btn edit"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleEditTask(task);
                            }}
                          >
                            ✏️
                          </button>
                          <button
                            className="premium-action-btn delete"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteTask(task);
                            }}
                          >
                            🗑️
                          </button>
                        </div>
                      </div>
                    ))
                  )}
                  
                  {/* 添加任务按钮 */}
                  <button 
                    className="premium-add-task-btn"
                    style={{ width: '100%', padding: 'var(--space-3)', fontSize: '0.9rem' }}
                    onClick={() => handleAddTaskClick(status.id)}
                  >
                    <span>+</span>
                    <span>添加任务</span>
                  </button>
                </div>
              </div>
            );
          })}
        </div>
        
        {/* 任务创建/编辑对话框 */}
        <TaskModal
          isOpen={isTaskModalOpen}
          onClose={() => {
            setIsTaskModalOpen(false);
            setEditingTask(null);
          }}
          onSubmit={editingTask ? handleUpdateTask : (data) => {
            const taskData = { ...data, status_id: currentStatusId };
            return handleCreateTask(taskData);
          }}
          task={editingTask}
          title={editingTask ? '编辑任务' : '创建任务'}
          projectId={projectId!}
        />
        
        {/* 实时协作聊天 */}
        <ChatWidget
          projectId={projectIdNum}
          isOpen={isChatOpen}
          onToggle={() => setIsChatOpen(!isChatOpen)}
          sendChatMessage={sendChatMessage}
        />
        
        {/* 文件管理器 */}
        <FileManager
          projectId={projectIdNum}
          isOpen={isFileManagerOpen}
          onClose={() => setIsFileManagerOpen(false)}
        />
        
        {/* 团队管理 */}
        <TeamModal
          projectId={projectIdNum}
          isOpen={isTeamModalOpen}
          onClose={() => setIsTeamModalOpen(false)}
        />
      </div>
    </div>
  );
};

export default ProjectBoard;