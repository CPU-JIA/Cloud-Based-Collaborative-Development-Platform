import React, { useState, useEffect } from 'react';
import { Navigate, useLocation, useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { ApiError } from '../types';
import '../styles/modern-enterprise.css';
import '../styles/premium-auth.css';

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

  const handleDemoLogin = (demoEmail: string, demoPassword: string) => {
    setEmail(demoEmail);
    setPassword(demoPassword);
  };

  return (
    <div className="premium-auth-page">
      {/* 浮动几何图形 */}
      <div className="floating-shapes">
        <div className="shape shape-1"></div>
        <div className="shape shape-2"></div>
        <div className="shape shape-3"></div>
      </div>

      <div className="premium-auth-container">
        {/* 左侧品牌展示 */}
        <div className="premium-brand-section">
          <div className="premium-brand-header">
            <div className="premium-logo">
              <div className="premium-logo-icon">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <div>
                <h1 className="premium-brand-title">CloudDev</h1>
                <p className="premium-brand-subtitle">现代化企业协作开发平台</p>
              </div>
            </div>
          </div>

          <div className="premium-features-grid">
            <div className="premium-feature-item">
              <div className="premium-feature-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <h3 className="premium-feature-title">⚡ 闪电协作</h3>
              <p className="premium-feature-desc">实时同步，多人协作无延迟</p>
            </div>

            <div className="premium-feature-item">
              <div className="premium-feature-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4z" />
                </svg>
              </div>
              <h3 className="premium-feature-title">🛡️ 银行级安全</h3>
              <p className="premium-feature-desc">端到端加密，保护代码资产</p>
            </div>

            <div className="premium-feature-item">
              <div className="premium-feature-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <h3 className="premium-feature-title">🤖 AI增强</h3>
              <p className="premium-feature-desc">智能代码分析和项目洞察</p>
            </div>

            <div className="premium-feature-item">
              <div className="premium-feature-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
                </svg>
              </div>
              <h3 className="premium-feature-title">📈 数据驱动</h3>
              <p className="premium-feature-desc">可视化项目进度和团队效率</p>
            </div>
          </div>

          <div className="premium-stats">
            <div className="premium-stat">
              <div className="premium-stat-number">50K+</div>
              <p className="premium-stat-label">活跃开发者</p>
            </div>
            <div className="premium-stat">
              <div className="premium-stat-number">99.9%</div>
              <p className="premium-stat-label">服务可用性</p>
            </div>
            <div className="premium-stat">
              <div className="premium-stat-number">1000+</div>
              <p className="premium-stat-label">企业客户</p>
            </div>
          </div>
        </div>

        {/* 右侧登录表单 */}
        <div className="premium-form-section">
          <div className="premium-form-header">
            <h2 className="premium-form-title">欢迎回来</h2>
            <p className="premium-form-subtitle">登录您的企业账户，继续高效协作开发</p>
          </div>

          <form onSubmit={handleSubmit} className="premium-form">
            <div className="premium-field-group">
              <label htmlFor="email" className="premium-field-label">
                <svg width="18" height="18" viewBox="0 0 20 20" fill="currentColor" className="premium-field-icon">
                  <path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z"/>
                  <path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z"/>
                </svg>
                企业邮箱地址
              </label>
              <input
                type="email"
                id="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                className="premium-input"
                placeholder="输入您的企业邮箱地址"
                disabled={isLoading}
                autoComplete="email"
                autoFocus
              />
            </div>

            <div className="premium-field-group">
              <label htmlFor="password" className="premium-field-label">
                <svg width="18" height="18" viewBox="0 0 20 20" fill="currentColor" className="premium-field-icon">
                  <path fillRule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 616 0z" clipRule="evenodd"/>
                </svg>
                登录密码
              </label>
              <input
                type="password"
                id="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                className="premium-input"
                placeholder="输入您的登录密码"
                disabled={isLoading}
                autoComplete="current-password"
              />
            </div>

            {error && (
              <div className="premium-error-alert">
                <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd"/>
                </svg>
                <div>
                  <strong>登录失败</strong>
                  <div>{error}</div>
                </div>
              </div>
            )}

            <button
              type="submit"
              disabled={isLoading || !email || !password}
              className="premium-submit-btn"
            >
              {isLoading ? (
                <>
                  <div className="premium-loading-spinner"></div>
                  <span>正在登录...</span>
                </>
              ) : (
                <>
                  <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clipRule="evenodd"/>
                  </svg>
                  <span>立即登录</span>
                </>
              )}
            </button>
          </form>

          {/* 演示账户信息 */}
          <div className="premium-demo-section">
            <div className="premium-demo-header">
              <div className="premium-demo-icon">
                <svg width="18" height="18" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 011-1h.01a1 1 0 110 2H12a1 1 0 01-1-1zm1 4a1 1 0 011 1v4a1 1 0 11-2 0v-4a1 1 0 011-1z" clipRule="evenodd"/>
                </svg>
              </div>
              <h4 className="premium-demo-title">体验演示账户</h4>
            </div>
            <div className="premium-demo-credentials">
              <div className="premium-demo-item">
                <div className="premium-demo-label">演示邮箱</div>
                <div 
                  className="premium-demo-value"
                  onClick={() => handleDemoLogin('demo@clouddev.com', 'demo123')}
                >
                  demo@clouddev.com
                </div>
              </div>
              <div className="premium-demo-item">
                <div className="premium-demo-label">演示密码</div>
                <div 
                  className="premium-demo-value"
                  onClick={() => handleDemoLogin('demo@clouddev.com', 'demo123')}
                >
                  demo123
                </div>
              </div>
            </div>
          </div>

          <div className="premium-divider">
            <div className="premium-divider-line"></div>
            <span className="premium-divider-text">或</span>
            <div className="premium-divider-line"></div>
          </div>

          {/* 底部链接 */}
          <div className="premium-auth-footer">
            <p className="footer-text">
              还没有账户？
              <Link to="/register" className="premium-auth-link">
                立即注册
              </Link>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Login;