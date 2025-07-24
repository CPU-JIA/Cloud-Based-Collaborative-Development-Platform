// 用户相关类型
export interface User {
  id: string;
  username: string;
  email: string;
  display_name?: string;
  avatar_url?: string;
  status: 'active' | 'suspended' | 'deactivated';
  is_platform_admin: boolean;
  created_at: string;
}

// 认证相关类型
export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

// 项目相关类型
export interface Project {
  id: string;
  tenant_id: string;
  key: string;
  name: string;
  description?: string;
  manager_id?: string;
  status: 'active' | 'archived';
  created_at: string;
  updated_at: string;
}

// 任务相关类型
export interface Task {
  id: string;
  project_id: string;
  task_number: number;
  title: string;
  description?: string;
  status_id?: string;
  assignee_id?: string;
  creator_id: string;
  parent_task_id?: string;
  due_date?: string;
  priority: 'low' | 'medium' | 'high' | 'urgent';
  created_at: string;
  updated_at: string;
}

// 任务状态类型
export interface TaskStatus {
  id: string;
  tenant_id: string;
  name: string;
  category: 'todo' | 'in_progress' | 'done';
  display_order: number;
}

// API响应类型
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
}

// 分页响应类型
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

// 错误类型
export interface ApiError {
  error: string;
  message: string;
  details?: any;
}