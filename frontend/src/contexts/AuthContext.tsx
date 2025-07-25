import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { User, LoginRequest, LoginResponse } from '../types';
import { authApi } from '../utils/api';

interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => Promise<void>;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  // 初始化时检查本地存储的用户信息
  useEffect(() => {
    const initAuth = async () => {
      try {
        const storedUser = localStorage.getItem('user');
        const storedToken = localStorage.getItem('access_token');

        if (storedUser && storedToken) {
          try {
            const parsedUser = JSON.parse(storedUser);
            // 验证解析的用户数据完整性
            if (parsedUser && parsedUser.email) {
              setUser(parsedUser);
              console.log('从本地存储恢复用户:', parsedUser);
            } else {
              // 用户数据不完整，清除存储
              console.warn('本地用户数据不完整，清除存储');
              localStorage.removeItem('user');
              localStorage.removeItem('access_token');
              localStorage.removeItem('refresh_token');
              setUser(null);
            }
          } catch (parseError) {
            // JSON解析失败，清除损坏的数据
            console.error('用户数据解析失败:', parseError);
            localStorage.removeItem('user');
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
            setUser(null);
          }
        } else {
          console.log('未找到本地登录信息');
        }
      } catch (error) {
        console.error('初始化认证失败:', error);
      } finally {
        setLoading(false);
      }
    };

    initAuth();
  }, []);

  const login = async (credentials: LoginRequest): Promise<void> => {
    try {
      setLoading(true);
      const response: LoginResponse = await authApi.login(
        credentials.email,
        credentials.password
      );

      // 验证响应数据完整性
      if (response && response.user && response.access_token) {
        // 存储用户信息和token
        localStorage.setItem('user', JSON.stringify(response.user));
        localStorage.setItem('access_token', response.access_token);
        if (response.refresh_token) {
          localStorage.setItem('refresh_token', response.refresh_token);
        }

        setUser(response.user);
        console.log('登录成功，用户信息:', response.user);
      } else {
        throw new Error('登录响应数据不完整');
      }
    } catch (error) {
      console.error('登录失败:', error);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  const logout = async (): Promise<void> => {
    try {
      await authApi.logout();
    } catch (error) {
      console.error('登出失败:', error);
    } finally {
      // 无论API调用是否成功，都清除本地状态
      localStorage.removeItem('user');
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      setUser(null);
    }
  };

  const value: AuthContextType = {
    user,
    loading,
    login,
    logout,
    isAuthenticated: !!user,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};