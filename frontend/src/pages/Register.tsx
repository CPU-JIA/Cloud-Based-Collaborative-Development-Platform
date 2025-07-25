import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { authApi } from '../utils/api';
import '../styles/modern-enterprise.css';
import '../styles/premium-auth.css';

const Register: React.FC = () => {
  const navigate = useNavigate();
  const { login } = useAuth();
  
  const [formData, setFormData] = useState({
    email: '',
    password: '',
    confirmPassword: '',
    display_name: '',
    username: '',
  });
  
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  const validateForm = () => {
    const newErrors: { [key: string]: string } = {};

    // 邮箱验证
    if (!formData.email.trim()) {
      newErrors.email = '邮箱不能为空';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = '邮箱格式不正确';
    }

    // 用户名验证
    if (!formData.username.trim()) {
      newErrors.username = '用户名不能为空';
    } else if (!/^[a-zA-Z0-9_]{3,20}$/.test(formData.username)) {
      newErrors.username = '用户名只能包含字母、数字和下划线，长度3-20位';
    }

    // 显示名称验证
    if (!formData.display_name.trim()) {
      newErrors.display_name = '显示名称不能为空';
    } else if (formData.display_name.length > 50) {
      newErrors.display_name = '显示名称不能超过50个字符';
    }

    // 密码验证
    if (!formData.password) {
      newErrors.password = '密码不能为空';
    } else if (formData.password.length < 6) {
      newErrors.password = '密码至少需要6个字符';
    }

    // 确认密码验证
    if (!formData.confirmPassword) {
      newErrors.confirmPassword = '请确认密码';
    } else if (formData.password !== formData.confirmPassword) {
      newErrors.confirmPassword = '两次输入的密码不一致';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));

    // 清除字段错误
    if (errors[name]) {
      setErrors(prev => ({
        ...prev,
        [name]: ''
      }));
    }

    // 自动生成用户名
    if (name === 'display_name' && !formData.username) {
      const autoUsername = value
        .toLowerCase()
        .replace(/[^a-z0-9]/g, '')
        .substring(0, 15);
      
      setFormData(prev => ({
        ...prev,
        username: autoUsername
      }));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);

    try {
      // 注册用户
      const registerResponse = await authApi.register({
        email: formData.email,
        password: formData.password,
        display_name: formData.display_name,
        username: formData.username,
      });

      if (registerResponse.success) {
        // 注册成功后自动登录
        const loginResponse = await authApi.login(formData.email, formData.password);
        
        if (loginResponse.success) {
          await login(loginResponse.user, loginResponse.access_token);
          navigate('/dashboard');
        } else {
          // 注册成功但登录失败，跳转到登录页
          navigate('/login?message=注册成功，请登录');
        }
      }
    } catch (error: any) {
      console.error('注册失败:', error);
      setErrors({ 
        submit: error.message || '注册失败，请重试' 
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="premium-auth-page">
      {/* 浮动几何图形背景 */}
      <div className="floating-shapes">
        <div className="shape shape-1"></div>
        <div className="shape shape-2"></div>
        <div className="shape shape-3"></div>
        <div className="shape shape-4"></div>
        <div className="shape shape-5"></div>
        <div className="shape shape-6"></div>
      </div>
      
      <div className="premium-auth-container">
        {/* 左侧品牌展示区域 */}
        <div className="premium-brand-section">
          <div className="premium-brand-content">
            <div className="premium-logo-section">
              <div className="premium-logo-icon">📋</div>
              <h1 className="premium-brand-title">CloudDev</h1>
              <p className="premium-brand-tagline">Next-Gen Enterprise Platform</p>
            </div>
            
            <div className="premium-brand-description">
              <h2>加入数千家企业的选择</h2>
              <p>CloudDev 为现代企业提供最先进的协作开发解决方案，助力团队实现数字化转型和业务增长。</p>
            </div>
            
            <div className="premium-feature-grid">
              <div className="premium-feature-item">
                <div className="premium-feature-icon">⚡</div>
                <div className="premium-feature-content">
                  <h4>闪电般快速</h4>
                  <p>毫秒级响应，零延迟协作</p>
                </div>
              </div>
              
              <div className="premium-feature-item">
                <div className="premium-feature-icon">🛡️</div>
                <div className="premium-feature-content">
                  <h4>企业级安全</h4>
                  <p>SOC2 Type II 合规认证</p>
                </div>
              </div>
              
              <div className="premium-feature-item">
                <div className="premium-feature-icon">🚀</div>
                <div className="premium-feature-content">
                  <h4>无限扩展</h4>
                  <p>支持数万人同时在线</p>
                </div>
              </div>
              
              <div className="premium-feature-item">
                <div className="premium-feature-icon">📊</div>
                <div className="premium-feature-content">
                  <h4>智能洞察</h4>
                  <p>AI驱动的项目分析</p>
                </div>
              </div>
            </div>
            
            <div className="premium-social-proof">
              <p className="premium-social-text">受到全球顶尖企业信赖</p>
              <div className="premium-logo-strip">
                <div className="premium-company-logo">🏢</div>
                <div className="premium-company-logo">🏭</div>
                <div className="premium-company-logo">🏪</div>
                <div className="premium-company-logo">🏫</div>
              </div>
            </div>
          </div>
        </div>
        
        {/* 右侧注册表单区域 */}
        <div className="premium-form-section">
          <div className="premium-form-container">
            <div className="premium-form-header">
              <h2 className="premium-form-title">创建您的账户</h2>
              <p className="premium-form-subtitle">开启您的企业级协作之旅</p>
            </div>

            <form onSubmit={handleSubmit} className="premium-auth-form">
              {/* 显示名称 */}
              <div className="premium-form-group">
                <label htmlFor="display_name" className="premium-form-label">
                  显示名称
                </label>
                <div className="premium-input-wrapper">
                  <div className="premium-input-icon">
                    <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd"/>
                    </svg>
                  </div>
                  <input
                    type="text"
                    id="display_name"
                    name="display_name"
                    value={formData.display_name}
                    onChange={handleInputChange}
                    className={`premium-form-input ${errors.display_name ? 'premium-input-error' : ''}`}
                    placeholder="您的姓名"
                    disabled={isSubmitting}
                  />
                </div>
                {errors.display_name && (
                  <p className="premium-error-message">{errors.display_name}</p>
                )}
              </div>

              {/* 用户名 */}
              <div className="premium-form-group">
                <label htmlFor="username" className="premium-form-label">
                  用户名
                </label>
                <div className="premium-input-wrapper">
                  <div className="premium-input-icon">
                    <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-6-3a2 2 0 11-4 0 2 2 0 014 0zm-2 4a5 5 0 00-4.546 2.916A5.986 5.986 0 0010 16a5.986 5.986 0 004.546-2.084A5 5 0 0010 11z" clipRule="evenodd"/>
                    </svg>
                  </div>
                  <input
                    type="text"
                    id="username"
                    name="username"
                    value={formData.username}
                    onChange={handleInputChange}
                    className={`premium-form-input ${errors.username ? 'premium-input-error' : ''}`}
                    placeholder="3-20位字母数字下划线"
                    disabled={isSubmitting}
                  />
                </div>
                {errors.username && (
                  <p className="premium-error-message">{errors.username}</p>
                )}
              </div>

              {/* 邮箱 */}
              <div className="premium-form-group">
                <label htmlFor="email" className="premium-form-label">
                  企业邮箱
                </label>
                <div className="premium-input-wrapper">
                  <div className="premium-input-icon">
                    <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                      <path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z"/>
                      <path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z"/>
                    </svg>
                  </div>
                  <input
                    type="email"
                    id="email"
                    name="email"
                    value={formData.email}
                    onChange={handleInputChange}
                    className={`premium-form-input ${errors.email ? 'premium-input-error' : ''}`}
                    placeholder="name@company.com"
                    disabled={isSubmitting}
                  />
                </div>
                {errors.email && (
                  <p className="premium-error-message">{errors.email}</p>
                )}
              </div>

              {/* 密码组合 */}
              <div className="premium-form-row">
                <div className="premium-form-group">
                  <label htmlFor="password" className="premium-form-label">
                    密码
                  </label>
                  <div className="premium-input-wrapper">
                    <div className="premium-input-icon">
                      <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clipRule="evenodd"/>
                      </svg>
                    </div>
                    <input
                      type="password"
                      id="password"
                      name="password"
                      value={formData.password}
                      onChange={handleInputChange}
                      className={`premium-form-input ${errors.password ? 'premium-input-error' : ''}`}
                      placeholder="至少6位字符"
                      disabled={isSubmitting}
                    />
                  </div>
                  {errors.password && (
                    <p className="premium-error-message">{errors.password}</p>
                  )}
                </div>

                <div className="premium-form-group">
                  <label htmlFor="confirmPassword" className="premium-form-label">
                    确认密码
                  </label>
                  <div className="premium-input-wrapper">
                    <div className="premium-input-icon">
                      <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M2.166 4.999A11.954 11.954 0 0010 1.944 11.954 11.954 0 0017.834 5c.11.65.166 1.32.166 2.001 0 5.225-3.34 9.67-8 11.317C5.34 16.67 2 12.225 2 7c0-.682.057-1.35.166-2.001zm11.541 3.708a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd"/>
                      </svg>
                    </div>
                    <input
                      type="password"
                      id="confirmPassword"
                      name="confirmPassword"
                      value={formData.confirmPassword}
                      onChange={handleInputChange}
                      className={`premium-form-input ${errors.confirmPassword ? 'premium-input-error' : ''}`}
                      placeholder="再次输入密码"
                      disabled={isSubmitting}
                    />
                  </div>
                  {errors.confirmPassword && (
                    <p className="premium-error-message">{errors.confirmPassword}</p>
                  )}
                </div>
              </div>

              {/* 错误信息 */}
              {errors.submit && (
                <div className="premium-error-banner">
                  <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd"/>
                  </svg>
                  <span>{errors.submit}</span>
                </div>
              )}
              
              {/* 服务条款 */}
              <div className="premium-terms-section">
                <p className="premium-terms-text">
                  注册即表示您同意我们的
                  <a href="#" className="premium-terms-link">服务条款</a>
                  和
                  <a href="#" className="premium-terms-link">隐私政策</a>
                </p>
              </div>

              {/* 提交按钮 */}
              <button
                type="submit"
                className="premium-auth-button premium-primary-button"
                disabled={isSubmitting}
              >
                {isSubmitting ? (
                  <>
                    <div className="premium-button-spinner"></div>
                    <span>创建账户中...</span>
                  </>
                ) : (
                  <>
                    <span>创建企业账户</span>
                    <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clipRule="evenodd"/>
                    </svg>
                  </>
                )}
              </button>

              {/* 登录链接 */}
              <div className="premium-form-footer">
                <p className="premium-footer-text">
                  已有账户？
                  <Link to="/login" className="premium-footer-link">
                    立即登录 →
                  </Link>
                </p>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Register;