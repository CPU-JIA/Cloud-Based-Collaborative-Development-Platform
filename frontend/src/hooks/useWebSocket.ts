// React WebSocket Hook
import { useEffect, useRef, useCallback, useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { WebSocketClient, MessageType, ConnectionStatus, WSMessage, wsManager } from '../utils/websocket';

// Hook选项接口
interface UseWebSocketOptions {
  projectId: number;
  autoConnect?: boolean;
  onMessage?: (message: WSMessage) => void;
  onStatusChange?: (status: ConnectionStatus) => void;
}

// Hook返回值接口
interface UseWebSocketReturn {
  client: WebSocketClient | null;
  status: ConnectionStatus;
  isConnected: boolean;
  connect: () => Promise<void>;
  disconnect: () => void;
  send: (type: MessageType, data: any) => void;
  onlineUsers: any[];
  sendTaskUpdate: (taskData: any) => void;
  sendTaskCreate: (taskData: any) => void;
  sendTaskDelete: (taskId: number) => void;
  sendChatMessage: (message: string) => void;
  sendTyping: (isTyping: boolean, taskId?: number) => void;
}

export const useWebSocket = (options: UseWebSocketOptions): UseWebSocketReturn => {
  const { user } = useAuth();
  const { projectId, autoConnect = true, onMessage, onStatusChange } = options;
  
  const [client, setClient] = useState<WebSocketClient | null>(null);
  const [status, setStatus] = useState<ConnectionStatus>(ConnectionStatus.DISCONNECTED);
  const [onlineUsers, setOnlineUsers] = useState<any[]>([]);
  
  // 使用ref存储回调函数，避免重复注册监听器
  const onMessageRef = useRef(onMessage);
  const onStatusChangeRef = useRef(onStatusChange);
  
  // 更新ref
  useEffect(() => {
    onMessageRef.current = onMessage;
    onStatusChangeRef.current = onStatusChange;
  }, [onMessage, onStatusChange]);
  
  // 初始化WebSocket客户端
  useEffect(() => {
    if (!user || !projectId) return;
    
    const wsClient = wsManager.getClient(user, projectId);
    setClient(wsClient);
    
    // 状态监听器
    const statusListener = (newStatus: ConnectionStatus) => {
      setStatus(newStatus);
      onStatusChangeRef.current?.(newStatus);
    };
    
    // 通用消息监听器
    const messageListener = (message: WSMessage) => {
      onMessageRef.current?.(message);
    };
    
    // 用户状态监听器
    const userStatusListener = (message: WSMessage) => {
      if (message.type === MessageType.USER_STATUS && message.data?.online_users) {
        setOnlineUsers(message.data.online_users);
      }
    };
    
    // 用户加入监听器
    const userJoinListener = (message: WSMessage) => {
      if (message.user_id !== user.id) {
        setOnlineUsers(prev => {
          const exists = prev.find(u => u.user_id === message.user_id);
          if (!exists) {
            return [...prev, {
              user_id: message.user_id,
              username: message.username,
              avatar: message.avatar,
              status: message.data?.status || 'online',
              last_seen: message.data?.last_active || new Date().toISOString()
            }];
          }
          return prev;
        });
      }
    };
    
    // 用户离开监听器
    const userLeaveListener = (message: WSMessage) => {
      setOnlineUsers(prev => prev.filter(u => u.user_id !== message.user_id));
    };
    
    // 注册监听器
    wsClient.onStatusChange(statusListener);
    wsClient.on(MessageType.USER_STATUS, userStatusListener);
    wsClient.on(MessageType.USER_JOIN, userJoinListener);
    wsClient.on(MessageType.USER_LEAVE, userLeaveListener);
    
    // 为所有消息类型注册通用监听器
    Object.values(MessageType).forEach(type => {
      wsClient.on(type, messageListener);
    });
    
    // 自动连接
    if (autoConnect) {
      wsClient.connect().catch(error => {
        console.error('WebSocket自动连接失败:', error);
      });
    }
    
    // 设置初始状态
    setStatus(wsClient.getStatus());
    
    // 清理函数
    return () => {
      wsClient.offStatusChange(statusListener);
      wsClient.off(MessageType.USER_STATUS, userStatusListener);
      wsClient.off(MessageType.USER_JOIN, userJoinListener);
      wsClient.off(MessageType.USER_LEAVE, userLeaveListener);
      
      Object.values(MessageType).forEach(type => {
        wsClient.off(type, messageListener);
      });
    };
  }, [user, projectId, autoConnect]);
  
  // 连接方法
  const connect = useCallback(async () => {
    if (client) {
      await client.connect();
    }
  }, [client]);
  
  // 断开连接方法
  const disconnect = useCallback(() => {
    if (client) {
      client.disconnect();
    }
  }, [client]);
  
  // 发送消息方法
  const send = useCallback((type: MessageType, data: any) => {
    if (client) {
      client.send(type, data);
    }
  }, [client]);
  
  // 便捷方法
  const sendTaskUpdate = useCallback((taskData: any) => {
    client?.sendTaskUpdate(taskData);
  }, [client]);
  
  const sendTaskCreate = useCallback((taskData: any) => {
    client?.sendTaskCreate(taskData);
  }, [client]);
  
  const sendTaskDelete = useCallback((taskId: number) => {
    client?.sendTaskDelete(taskId);
  }, [client]);
  
  const sendChatMessage = useCallback((message: string) => {
    client?.sendChatMessage(message);
  }, [client]);
  
  const sendTyping = useCallback((isTyping: boolean, taskId?: number) => {
    client?.sendTyping(isTyping, taskId);
  }, [client]);
  
  // 组件卸载时清理
  useEffect(() => {
    return () => {
      if (projectId) {
        // 注意：这里不调用removeClient，因为可能有其他组件在使用
        // wsManager.removeClient(projectId);
      }
    };
  }, [projectId]);
  
  return {
    client,
    status,
    isConnected: status === ConnectionStatus.CONNECTED,
    connect,
    disconnect,
    send,
    onlineUsers,
    sendTaskUpdate,
    sendTaskCreate,
    sendTaskDelete,
    sendChatMessage,
    sendTyping,
  };
};

// 特定消息类型的Hook
export const useWebSocketMessage = (
  projectId: number,
  messageType: MessageType,
  onMessage: (message: WSMessage) => void
) => {
  const { user } = useAuth();
  
  useEffect(() => {
    if (!user || !projectId) return;
    
    const client = wsManager.getClient(user, projectId);
    client.on(messageType, onMessage);
    
    return () => {
      client.off(messageType, onMessage);
    };
  }, [user, projectId, messageType, onMessage]);
};

// 任务更新专用Hook
export const useTaskUpdates = (projectId: number) => {
  const [taskUpdates, setTaskUpdates] = useState<any[]>([]);
  
  const handleTaskUpdate = useCallback((message: WSMessage) => {
    setTaskUpdates(prev => [...prev, {
      ...message.data,
      timestamp: message.timestamp,
      user: {
        id: message.user_id,
        username: message.username,
        avatar: message.avatar
      }
    }]);
  }, []);
  
  useWebSocketMessage(projectId, MessageType.TASK_UPDATE, handleTaskUpdate);
  useWebSocketMessage(projectId, MessageType.TASK_CREATE, handleTaskUpdate);
  useWebSocketMessage(projectId, MessageType.TASK_DELETE, handleTaskUpdate);
  
  // 清理过期更新（保留最近50条）
  useEffect(() => {
    if (taskUpdates.length > 50) {
      setTaskUpdates(prev => prev.slice(-50));
    }
  }, [taskUpdates.length]);
  
  return {
    taskUpdates,
    clearUpdates: () => setTaskUpdates([])
  };
};

// 在线用户专用Hook
export const useOnlineUsers = (projectId: number) => {
  const { onlineUsers } = useWebSocket({ projectId });
  return onlineUsers;
};

// 聊天消息专用Hook
export const useChatMessages = (projectId: number) => {
  const [messages, setMessages] = useState<any[]>([]);
  
  const handleChatMessage = useCallback((message: WSMessage) => {
    setMessages(prev => [...prev, {
      id: message.data.message_id || `msg_${Date.now()}`,
      message: message.data.message,
      timestamp: message.timestamp,
      user: {
        id: message.user_id,
        username: message.username,
        avatar: message.avatar
      }
    }]);
  }, []);
  
  useWebSocketMessage(projectId, MessageType.CHAT_MESSAGE, handleChatMessage);
  
  return {
    messages,
    clearMessages: () => setMessages([])
  };
};

// 打字状态专用Hook
export const useTypingStatus = (projectId: number) => {
  const [typingUsers, setTypingUsers] = useState<Map<number, any>>(new Map());
  
  const handleTyping = useCallback((message: WSMessage) => {
    const userId = message.user_id;
    
    setTypingUsers(prev => {
      const newMap = new Map(prev);
      
      if (message.data.is_typing) {
        newMap.set(userId, {
          username: message.username,
          avatar: message.avatar,
          task_id: message.data.task_id,
          timestamp: Date.now()
        });
      } else {
        newMap.delete(userId);
      }
      
      return newMap;
    });
  }, []);
  
  useWebSocketMessage(projectId, MessageType.TYPING, handleTyping);
  
  // 清理过期的打字状态（5秒后自动清除）
  useEffect(() => {
    const interval = setInterval(() => {
      const now = Date.now();
      setTypingUsers(prev => {
        const newMap = new Map();
        prev.forEach((value, key) => {
          if (now - value.timestamp < 5000) {
            newMap.set(key, value);
          }
        });
        return newMap;
      });
    }, 1000);
    
    return () => clearInterval(interval);
  }, []);
  
  return Array.from(typingUsers.values());
};