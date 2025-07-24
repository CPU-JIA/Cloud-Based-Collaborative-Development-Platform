import React, { useState, useEffect } from 'react';
import { Navigate, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { ApiError } from '../types';

const Login: React.FC = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  
  const { login, isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  
  // 获取用户原本要访问的路径
  const from = (location.state as any)?.from?.pathname || '/dashboard';

  // 如果已经登录，直接重定向
  if (isAuthenticated) {
    return <Navigate to={from} replace />;
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!email || !password) {
      setError('请输入邮箱和密码');
      return;
    }

    try {
      setIsLoading(true);
      setError('');
      
      await login({ email, password });
      
      // 登录成功，重定向到目标页面
      navigate(from, { replace: true });
    } catch (err) {
      const apiError = err as ApiError;
      setError(apiError.message || '登录失败，请检查邮箱和密码');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-500 to-purple-600">
      <div className="max-w-md w-full mx-4">
        <div className="card">
          {/* Logo和标题 */}
          <div className="text-center mb-8">
            <div className="text-4xl mb-4">🚀</div>
            <h1 className="text-2xl font-bold text-gray-900 mb-2">
              企业协作开发平台
            </h1>
            <p className="text-gray-600">
              欢迎回来，请登录您的账户
            </p>
          </div>

          {/* 错误提示 */}
          {error && (
            <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
              <div className="flex">
                <div className="text-red-400">⚠️</div>
                <div className="ml-2 text-red-700 text-sm">{error}</div>
              </div>
            </div>
          )}

          {/* 登录表单 */}
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="form-group">
              <label htmlFor="email" className="form-label">
                邮箱地址
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="form-input"
                placeholder="请输入您的邮箱"
                disabled={isLoading}
                autoComplete="email"
                autoFocus
              />
            </div>

            <div className="form-group">
              <label htmlFor="password" className="form-label">
                密码
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="form-input"
                placeholder="请输入您的密码"
                disabled={isLoading}
                autoComplete="current-password"
              />
            </div>

            <button
              type="submit"
              disabled={isLoading || !email || !password}
              className="w-full btn btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? (
                <>
                  <div className="loading mr-2"></div>
                  登录中...
                </>
              ) : (
                '登录'
              )}
            </button>
          </form>

          {/* 演示账户提示 */}
          <div className="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
            <h3 className="font-medium text-blue-900 mb-2">演示账户</h3>
            <p className="text-blue-700 text-sm mb-2">
              您可以使用以下账户进行体验：
            </p>
            <div className="text-blue-800 text-sm font-mono bg-blue-100 p-2 rounded">
              <div>邮箱: demo@example.com</div>
              <div>密码: demo123</div>
            </div>
          </div>

          {/* 系统状态 */}
          <div className="mt-6 text-center">
            <div className="inline-flex items-center text-sm text-gray-500">
              <div className="w-2 h-2 bg-green-400 rounded-full mr-2"></div>
              系统运行正常
            </div>
          </div>
        </div>

        {/* 版权信息 */}
        <div className="text-center mt-6 text-white text-sm opacity-80">
          <p>🤖 Generated with Claude Code</p>
          <p>© 2024 企业协作开发平台. All rights reserved.</p>
        </div>
      </div>
    </div>
  );
};

export default Login;