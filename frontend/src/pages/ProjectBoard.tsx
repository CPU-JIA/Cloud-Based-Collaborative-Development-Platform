import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { Task, TaskStatus, Project } from '../types';
import { taskApi, projectApi } from '../utils/api';

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
    <div className="min-h-screen bg-gray-50">
      {/* 顶部导航栏 */}
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="container mx-auto px-4 py-4">
          <div className="flex justify-between items-center">
            <div className="flex items-center space-x-4">
              <button
                onClick={() => navigate('/dashboard')}
                className="text-gray-500 hover:text-gray-700"
              >
                ← 返回
              </button>
              <h1 className="text-xl font-bold text-gray-900">
                📋 {project?.name || '项目看板'}
              </h1>
              <span className="inline-flex items-center bg-blue-100 text-blue-800 px-2 py-1 rounded text-sm">
                {project?.key}
              </span>
            </div>
            
            <div className="flex items-center space-x-4">
              <div className="text-sm text-gray-600">
                {user?.display_name || user?.username}
              </div>
              <button
                onClick={handleLogout}
                className="btn btn-secondary text-sm"
              >
                退出
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* 看板统计 */}
      <div className="container mx-auto px-4 py-4">
        <div className="grid grid-cols-3 gap-4 mb-6">
          {taskStatuses.map((status) => {
            const statusTasks = getTasksByStatus(status.id);
            return (
              <div key={status.id} className="bg-white rounded-lg p-4 shadow-sm">
                <div className="flex items-center justify-between">
                  <h3 className="font-medium text-gray-900">{status.name}</h3>
                  <span className="bg-gray-100 text-gray-800 px-2 py-1 rounded-full text-sm">
                    {statusTasks.length}
                  </span>
                </div>
              </div>
            );
          })}
        </div>

        {/* 看板主体 */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {taskStatuses.map((status) => {
            const statusTasks = getTasksByStatus(status.id);
            return (
              <div key={status.id} className="bg-white rounded-lg shadow-sm">
                {/* 列标题 */}
                <div className="p-4 border-b border-gray-200">
                  <div className="flex items-center justify-between">
                    <h3 className="font-semibold text-gray-900">{status.name}</h3>
                    <span className="bg-gray-100 text-gray-700 px-2 py-1 rounded-full text-sm">
                      {statusTasks.length}
                    </span>
                  </div>
                </div>

                {/* 任务列表 */}
                <div className="p-4 space-y-3 min-h-96">
                  {statusTasks.length === 0 ? (
                    <div className="text-center text-gray-500 py-8">
                      <div className="text-3xl mb-2">📝</div>
                      <p className="text-sm">暂无任务</p>
                    </div>
                  ) : (
                    statusTasks.map((task) => (
                      <div
                        key={task.id}
                        className="bg-gray-50 border border-gray-200 rounded-lg p-3 hover:shadow-md transition-shadow cursor-pointer"
                      >
                        <div className="flex items-start justify-between mb-2">
                          <h4 className="font-medium text-gray-900 text-sm leading-tight">
                            {task.title}
                          </h4>
                          <span className={`ml-2 px-2 py-1 rounded-full text-xs ${getPriorityColor(task.priority)}`}>
                            {getPriorityLabel(task.priority)}
                          </span>
                        </div>
                        
                        {task.description && (
                          <p className="text-gray-600 text-xs mb-2 line-clamp-2">
                            {task.description}
                          </p>
                        )}
                        
                        <div className="flex items-center justify-between text-xs text-gray-500">
                          <span>#{task.task_number}</span>
                          {task.due_date && (
                            <span>
                              📅 {new Date(task.due_date).toLocaleDateString()}
                            </span>
                          )}
                        </div>
                        
                        {task.assignee_id && (
                          <div className="mt-2 flex items-center">
                            <div className="w-6 h-6 bg-blue-500 rounded-full flex items-center justify-center text-white text-xs">
                              👤
                            </div>
                            <span className="ml-2 text-xs text-gray-600">已分配</span>
                          </div>
                        )}
                      </div>
                    ))
                  )}
                  
                  {/* 添加任务按钮 */}
                  <button className="w-full p-3 border-2 border-dashed border-gray-300 rounded-lg text-gray-500 hover:border-gray-400 hover:text-gray-600 transition-colors">
                    + 添加任务
                  </button>
                </div>
              </div>
            );
          })}
        </div>

        {/* 项目信息 */}
        {project?.description && (
          <div className="mt-6 bg-white rounded-lg p-4 shadow-sm">
            <h3 className="font-semibold text-gray-900 mb-2">项目描述</h3>
            <p className="text-gray-600 text-sm">{project.description}</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default ProjectBoard;