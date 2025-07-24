import React, { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate } from 'react-router-dom';
import { Project } from '../types';
import { projectApi } from '../utils/api';

const Dashboard: React.FC = () => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

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

  return (
    <div className="min-h-screen bg-gray-50">
      {/* 顶部导航栏 */}
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="container mx-auto px-4 py-4">
          <div className="flex justify-between items-center">
            <div className="flex items-center space-x-4">
              <h1 className="text-2xl font-bold text-gray-900">
                🚀 协作开发平台
              </h1>
              <div className="text-sm text-gray-500">
                欢迎回来，{user?.display_name || user?.username}
              </div>
            </div>
            
            <div className="flex items-center space-x-4">
              <div className="flex items-center text-sm text-gray-600">
                <div className="w-2 h-2 bg-green-400 rounded-full mr-2"></div>
                系统正常
              </div>
              <button
                onClick={handleLogout}
                className="btn btn-secondary text-sm"
              >
                退出登录
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* 主要内容区域 */}
      <main className="container mx-auto px-4 py-8">
        {/* 统计概览 */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <div className="card">
            <div className="flex items-center">
              <div className="text-3xl text-blue-500 mr-4">📊</div>
              <div>
                <div className="text-2xl font-bold text-gray-900">
                  {projects.length}
                </div>
                <div className="text-sm text-gray-600">活跃项目</div>
              </div>
            </div>
          </div>
          
          <div className="card">
            <div className="flex items-center">
              <div className="text-3xl text-green-500 mr-4">✅</div>
              <div>
                <div className="text-2xl font-bold text-gray-900">24</div>
                <div className="text-sm text-gray-600">已完成任务</div>
              </div>
            </div>
          </div>
          
          <div className="card">
            <div className="flex items-center">
              <div className="text-3xl text-orange-500 mr-4">⏳</div>
              <div>
                <div className="text-2xl font-bold text-gray-900">12</div>
                <div className="text-sm text-gray-600">进行中任务</div>
              </div>
            </div>
          </div>
          
          <div className="card">
            <div className="flex items-center">
              <div className="text-3xl text-purple-500 mr-4">👥</div>
              <div>
                <div className="text-2xl font-bold text-gray-900">8</div>
                <div className="text-sm text-gray-600">团队成员</div>
              </div>
            </div>
          </div>
        </div>

        {/* 项目列表 */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="lg:col-span-2">
            <div className="card">
              <div className="flex justify-between items-center mb-6">
                <h2 className="text-xl font-bold text-gray-900">我的项目</h2>
                <button className="btn btn-primary">
                  + 新建项目
                </button>
              </div>

              {loading ? (
                <div className="flex justify-center py-8">
                  <div className="loading"></div>
                </div>
              ) : error ? (
                <div className="text-center py-8 text-red-600">
                  {error}
                </div>
              ) : projects.length === 0 ? (
                <div className="text-center py-8 text-gray-500">
                  <div className="text-4xl mb-4">📝</div>
                  <p>还没有项目，创建第一个项目开始协作吧！</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {projects.map((project) => (
                    <div
                      key={project.id}
                      className="border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow cursor-pointer"
                      onClick={() => navigate(`/board/${project.id}`)}
                    >
                      <div className="flex justify-between items-start">
                        <div className="flex-1">
                          <h3 className="font-semibold text-gray-900 mb-1">
                            {project.name}
                          </h3>
                          <p className="text-gray-600 text-sm mb-2">
                            {project.description || '暂无描述'}
                          </p>
                          <div className="flex items-center text-xs text-gray-500">
                            <span className="inline-flex items-center bg-blue-100 text-blue-800 px-2 py-1 rounded">
                              {project.key}
                            </span>
                            <span className="ml-4">
                              创建于 {new Date(project.created_at).toLocaleDateString()}
                            </span>
                          </div>
                        </div>
                        <div className="ml-4">
                          <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs ${
                            project.status === 'active' 
                              ? 'bg-green-100 text-green-800' 
                              : 'bg-gray-100 text-gray-800'
                          }`}>
                            {project.status === 'active' ? '活跃' : '已归档'}
                          </span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* 快速操作 */}
        <div className="mt-8">
          <div className="card">
            <h2 className="text-xl font-bold text-gray-900 mb-4">快速操作</h2>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <button className="flex flex-col items-center p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
                <div className="text-2xl mb-2">📋</div>
                <span className="text-sm font-medium">创建任务</span>
              </button>
              <button className="flex flex-col items-center p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
                <div className="text-2xl mb-2">📚</div>
                <span className="text-sm font-medium">知识库</span>
              </button>
              <button className="flex flex-col items-center p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
                <div className="text-2xl mb-2">📊</div>
                <span className="text-sm font-medium">数据报告</span>
              </button>
              <button className="flex flex-col items-center p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
                <div className="text-2xl mb-2">⚙️</div>
                <span className="text-sm font-medium">设置</span>
              </button>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
};

export default Dashboard;