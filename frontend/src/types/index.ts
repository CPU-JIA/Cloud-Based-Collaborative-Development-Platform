// 用户相关类型
export interface User {
  id: number;
  email: string;
  name: string;
  display_name: string;
  username: string;
  avatar: string;
  created_at: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  display_name: string;
  username: string;
}

// 认证相关类型
export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  success: boolean;
  user: User;
  access_token: string;
  refresh_token?: string;
  expires_in?: number;
  message: string;
}

// 项目相关类型
export interface Project {
  id: number;
  name: string;
  key: string;
  description: string;
  status: string;
  created_at: string;
  updated_at: string;
  team_size: number;
  owner_id: number;
  tasks_count: number;
}

export interface CreateProjectRequest {
  name: string;
  description: string;
  key?: string;
}

export interface UpdateProjectRequest {
  name: string;
  description: string;
  status?: string;
}

// 任务相关类型
export interface Task {
  id: number;
  project_id: number;
  title: string;
  description: string;
  task_number: string;
  status_id: string;
  priority: 'low' | 'medium' | 'high' | 'urgent';
  assignee_id?: string;
  due_date?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTaskRequest {
  title: string;
  description: string;
  priority: string;
  status_id: string;
  assignee_id?: string;
  due_date?: string;
}

export interface UpdateTaskRequest {
  title: string;
  description: string;
  priority: string;
  status_id: string;
  assignee_id?: string;
  due_date?: string;
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