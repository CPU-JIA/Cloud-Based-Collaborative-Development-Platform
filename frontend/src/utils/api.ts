import axios, { AxiosResponse } from 'axios';
import { ApiResponse, ApiError } from '../types';

// 创建API客户端
const api = axios.create({
  baseURL: 'http://localhost:8082', // 连接Mock API服务
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器 - 添加认证token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('access_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器 - 处理通用错误
api.interceptors.response.use(
  (response: AxiosResponse) => {
    return response;
  },
  (error) => {
    if (error.response?.status === 401) {
      // Token过期，清除本地存储并跳转到登录页
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// API工具函数
export const apiCall = async <T>(
  method: 'GET' | 'POST' | 'PUT' | 'DELETE',
  url: string,
  data?: any
): Promise<T> => {
  try {
    const response = await api.request<ApiResponse<T>>({
      method,
      url,
      data,
    });
    
    if (response.data.success) {
      return response.data.data as T;
    } else {
      throw new Error(response.data.message || 'API调用失败');
    }
  } catch (error: any) {
    const apiError: ApiError = {
      error: error.response?.data?.error || 'NETWORK_ERROR',
      message: error.response?.data?.message || error.message || '网络错误',
      details: error.response?.data,
    };
    throw apiError;
  }
};

// 认证相关API
export const authApi = {
  login: async (email: string, password: string) => {
    const response = await api.post('/auth/login', { email, password });
    return response.data; // 直接返回Mock API的响应格式
  },

  register: async (userData: {
    email: string;
    password: string;
    display_name: string;
    username: string;
  }) => {
    const response = await api.post('/auth/register', userData);
    return response.data;
  },
    
  logout: () =>
    apiCall('POST', '/auth/logout', {}),
    
  getCurrentUser: () =>
    apiCall('GET', '/users/me', {}),
    
  refreshToken: () =>
    apiCall('POST', '/auth/refresh', {}),
};

// 项目相关API
export const projectApi = {
  list: () =>
    apiCall('GET', '/projects'),
    
  getById: (id: string) =>
    apiCall('GET', `/projects/${id}`),
    
  create: (data: any) =>
    apiCall('POST', '/projects', data),
    
  update: (id: string, data: any) =>
    apiCall('PUT', `/projects/${id}`, data),
    
  delete: (id: string) =>
    apiCall('DELETE', `/projects/${id}`),
};

// 任务相关API
export const taskApi = {
  list: (projectId: string) =>
    apiCall('GET', `/projects/${projectId}/tasks`),
    
  getById: (taskId: string) =>
    apiCall('GET', `/tasks/${taskId}`),
    
  create: async (projectId: string, data: any) => {
    const response = await api.post(`/tasks?project_id=${projectId}`, data);
    return response.data;
  },
    
  update: (taskId: string, data: any) =>
    apiCall('PUT', `/tasks/${taskId}`, data),
    
  delete: (taskId: string) =>
    apiCall('DELETE', `/tasks/${taskId}`),
    
  reorder: (data: any) =>
    apiCall('POST', '/tasks/reorder', data),
};

// 用户相关API
export const userApi = {
  getCurrentUser: () =>
    apiCall('GET', '/users/me'),
    
  list: () =>
    apiCall('GET', '/users'),
};

export default api;