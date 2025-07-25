import React, { useState, useEffect } from 'react';
import { Project, CreateProjectRequest, UpdateProjectRequest } from '../types';
import '../styles/premium-modal.css';

interface ProjectModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: CreateProjectRequest | UpdateProjectRequest) => Promise<void>;
  project?: Project | null;
  title: string;
}

const ProjectModal: React.FC<ProjectModalProps> = ({
  isOpen,
  onClose,
  onSubmit,
  project,
  title
}) => {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    key: '',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  useEffect(() => {
    if (project) {
      setFormData({
        name: project.name,
        description: project.description,
        key: project.key,
      });
    } else {
      setFormData({
        name: '',
        description: '',
        key: '',
      });
    }
    setErrors({});
  }, [project, isOpen]);

  const validateForm = () => {
    const newErrors: { [key: string]: string } = {};

    if (!formData.name.trim()) {
      newErrors.name = '项目名称不能为空';
    } else if (formData.name.length > 100) {
      newErrors.name = '项目名称不能超过100个字符';
    }

    if (!formData.key.trim()) {
      newErrors.key = '项目键不能为空';
    } else if (!/^[A-Z0-9-]+$/.test(formData.key)) {
      newErrors.key = '项目键只能包含大写字母、数字和连字符';
    } else if (formData.key.length > 20) {
      newErrors.key = '项目键不能超过20个字符';
    }

    if (formData.description && formData.description.length > 500) {
      newErrors.description = '项目描述不能超过500个字符';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
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

    // 自动生成项目键
    if (name === 'name' && !project) {
      const autoKey = value
        .toUpperCase()
        .replace(/[^A-Z0-9\s]/g, '')
        .replace(/\s+/g, '-')
        .substring(0, 15);
      
      setFormData(prev => ({
        ...prev,
        key: autoKey
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
      await onSubmit(formData);
      onClose();
    } catch (error: any) {
      console.error('项目操作失败:', error);
      setErrors({ 
        submit: error.message || '操作失败，请重试' 
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  if (!isOpen) return null;

  return (
    <div className="premium-modal-overlay" onClick={handleBackdropClick}>
      <div className="premium-modal-container">
        <div className="premium-modal-content">
          {/* Modal Header */}
          <div className="premium-modal-header">
            <div className="premium-modal-icon">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                <path d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"/>
              </svg>
            </div>
            <div className="premium-modal-title-section">
              <h2 className="premium-modal-title">{title}</h2>
              <p className="premium-modal-subtitle">
                {project ? '更新项目信息以优化团队协作' : '创建新项目，开启高效协作之旅'}
              </p>
            </div>
            <button 
              className="premium-modal-close-btn"
              onClick={onClose}
              disabled={isSubmitting}
            >
              <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd"/>
              </svg>
            </button>
          </div>

          {/* Modal Body */}
          <form onSubmit={handleSubmit} className="premium-modal-form">
            <div className="premium-modal-body">
              {/* 项目名称 */}
              <div className="premium-form-group">
                <label htmlFor="name" className="premium-form-label">
                  项目名称 *
                </label>
                <div className="premium-input-wrapper">
                  <div className="premium-input-icon">
                    <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z" clipRule="evenodd"/>
                    </svg>
                  </div>
                  <input
                    type="text"
                    id="name"
                    name="name"
                    value={formData.name}
                    onChange={handleInputChange}
                    className={`premium-form-input ${errors.name ? 'premium-input-error' : ''}`}
                    placeholder="例如：企业管理系统"
                    disabled={isSubmitting}
                    maxLength={100}
                  />
                </div>
                {errors.name && (
                  <p className="premium-error-message">{errors.name}</p>
                )}
              </div>

              {/* 项目键 */}
              <div className="premium-form-group">
                <label htmlFor="key" className="premium-form-label">
                  项目键 *
                  <span className="premium-label-hint">用于URL和API，只能包含大写字母、数字和连字符</span>
                </label>
                <div className="premium-input-wrapper">
                  <div className="premium-input-icon">
                    <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M18 8a6 6 0 01-7.743 5.743L10 14l-1 1-1 1H6v2H2v-4l4.257-4.257A6 6 0 1118 8zm-6-4a1 1 0 100 2 2 2 0 012 2 1 1 0 102 0 4 4 0 00-4-4z" clipRule="evenodd"/>
                    </svg>
                  </div>
                  <input
                    type="text"
                    id="key"
                    name="key"
                    value={formData.key}
                    onChange={handleInputChange}
                    className={`premium-form-input ${errors.key ? 'premium-input-error' : ''}`}
                    placeholder="例如：EMS-2024"
                    disabled={isSubmitting}
                    maxLength={20}
                    style={{ textTransform: 'uppercase' }}
                  />
                </div>
                {errors.key && (
                  <p className="premium-error-message">{errors.key}</p>
                )}
              </div>

              {/* 项目描述 */}
              <div className="premium-form-group">
                <label htmlFor="description" className="premium-form-label">
                  项目描述
                  <span className="premium-label-hint">简要描述项目的目标和功能</span>
                </label>
                <div className="premium-textarea-wrapper">
                  <textarea
                    id="description"
                    name="description"
                    value={formData.description}
                    onChange={handleInputChange}
                    className={`premium-form-textarea ${errors.description ? 'premium-input-error' : ''}`}
                    placeholder="描述项目的主要功能、目标和特色..."
                    disabled={isSubmitting}
                    rows={4}
                    maxLength={500}
                  />
                  <div className="premium-textarea-counter">
                    {formData.description.length}/500
                  </div>
                </div>
                {errors.description && (
                  <p className="premium-error-message">{errors.description}</p>
                )}
              </div>

              {/* 项目模板选择 */}
              {!project && (
                <div className="premium-form-group">
                  <label className="premium-form-label">
                    项目模板
                    <span className="premium-label-hint">选择合适的项目模板快速开始</span>
                  </label>
                  <div className="premium-template-grid">
                    <div className="premium-template-card active">
                      <div className="premium-template-icon">🚀</div>
                      <div className="premium-template-content">
                        <h4>敏捷开发</h4>
                        <p>适合快速迭代的敏捷项目</p>
                      </div>
                    </div>
                    <div className="premium-template-card">
                      <div className="premium-template-icon">📊</div>
                      <div className="premium-template-content">
                        <h4>数据分析</h4>
                        <p>数据驱动的分析项目</p>
                      </div>
                    </div>
                    <div className="premium-template-card">
                      <div className="premium-template-icon">🏢</div>
                      <div className="premium-template-content">
                        <h4>企业应用</h4>
                        <p>大型企业级应用开发</p>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* 错误信息 */}
              {errors.submit && (
                <div className="premium-error-banner">
                  <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd"/>
                  </svg>
                  <span>{errors.submit}</span>
                </div>
              )}
            </div>

            {/* Modal Footer */}
            <div className="premium-modal-footer">
              <button
                type="button"
                className="premium-modal-btn premium-btn-secondary"
                onClick={onClose}
                disabled={isSubmitting}
              >
                取消
              </button>
              <button
                type="submit"
                className="premium-modal-btn premium-btn-primary"
                disabled={isSubmitting}
              >
                {isSubmitting ? (
                  <>
                    <div className="premium-button-spinner"></div>
                    <span>{project ? '更新中...' : '创建中...'}</span>
                  </>
                ) : (
                  <>
                    <span>{project ? '更新项目' : '创建项目'}</span>
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                      <path fillRule="evenodd" d="M1 8a.5.5 0 01.5-.5h11.793l-3.147-3.146a.5.5 0 01.708-.708l4 4a.5.5 0 010 .708l-4 4a.5.5 0 01-.708-.708L13.293 8.5H1.5A.5.5 0 011 8z"/>
                    </svg>
                  </>
                )}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default ProjectModal;