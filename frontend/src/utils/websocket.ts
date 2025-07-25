// WebSocket实时协作客户端
import { User } from '../types';

// 消息类型定义
export enum MessageType {
  TASK_UPDATE = 'task_update',
  TASK_CREATE = 'task_create', 
  TASK_DELETE = 'task_delete',
  USER_JOIN = 'user_join',
  USER_LEAVE = 'user_leave',
  USER_STATUS = 'user_status',
  PROJECT_UPDATE = 'project_update',
  CHAT_MESSAGE = 'chat_message',
  TYPING = 'typing',
  HEARTBEAT = 'heartbeat'
}

// WebSocket消息接口
export interface WSMessage {
  type: MessageType;
  project_id?: number;
  user_id: number;
  username: string;
  avatar: string;
  data: any;
  timestamp: string;
}

// 连接状态
export enum ConnectionStatus {
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  DISCONNECTED = 'disconnected',
  RECONNECTING = 'reconnecting',
  ERROR = 'error'
}

// 事件监听器类型
export type EventListener = (message: WSMessage) => void;
export type StatusListener = (status: ConnectionStatus) => void;

// WebSocket客户端类
export class WebSocketClient {
  private ws: WebSocket | null = null;
  private url: string;
  private user: User;
  private projectId: number;
  private status: ConnectionStatus = ConnectionStatus.DISCONNECTED;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private heartbeatInterval: number | null = null;
  private pingTimeout: number | null = null;
  
  // 事件监听器
  private eventListeners: Map<MessageType, EventListener[]> = new Map();
  private statusListeners: StatusListener[] = [];
  
  constructor(user: User, projectId: number) {
    this.user = user;
    this.projectId = projectId;
    this.url = this.buildWebSocketURL();
    
    // 初始化所有消息类型的监听器数组
    Object.values(MessageType).forEach(type => {
      this.eventListeners.set(type, []);
    });
  }
  
  // 构建WebSocket连接URL
  private buildWebSocketURL(): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = 'localhost:8084'; // WebSocket服务地址
    const params = new URLSearchParams({
      user_id: this.user.id.toString(),
      username: this.user.username || this.user.display_name,
      avatar: this.user.avatar || '',
      project_id: this.projectId.toString()
    });
    
    return `${protocol}//${host}/ws?${params.toString()}`;
  }
  
  // 连接WebSocket
  public connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }
      
      this.setStatus(ConnectionStatus.CONNECTING);
      
      try {
        this.ws = new WebSocket(this.url);
        
        this.ws.onopen = () => {
          console.log('🔗 WebSocket连接已建立');
          this.setStatus(ConnectionStatus.CONNECTED);
          this.reconnectAttempts = 0;
          this.startHeartbeat();
          resolve();
        };
        
        this.ws.onmessage = (event) => {
          this.handleMessage(event);
        };
        
        this.ws.onclose = (event) => {
          console.log('🔌 WebSocket连接已关闭', event.code, event.reason);
          this.setStatus(ConnectionStatus.DISCONNECTED);
          this.stopHeartbeat();
          
          // 如果不是主动关闭，尝试重连
          if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect();
          }
        };
        
        this.ws.onerror = (error) => {
          console.error('❌ WebSocket连接错误:', error);
          this.setStatus(ConnectionStatus.ERROR);
          reject(error);
        };
        
      } catch (error) {
        console.error('❌ WebSocket连接失败:', error);
        this.setStatus(ConnectionStatus.ERROR);
        reject(error);
      }
    });
  }
  
  // 断开连接
  public disconnect(): void {
    this.stopHeartbeat();
    
    if (this.ws) {
      this.ws.close(1000, '主动断开连接');
      this.ws = null;
    }
    
    this.setStatus(ConnectionStatus.DISCONNECTED);
  }
  
  // 发送消息
  public send(type: MessageType, data: any): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('⚠️ WebSocket未连接，无法发送消息');
      return;
    }
    
    const message: WSMessage = {
      type,
      project_id: this.projectId,
      user_id: this.user.id,
      username: this.user.username || this.user.display_name,
      avatar: this.user.avatar || '',
      data,
      timestamp: new Date().toISOString()
    };
    
    try {
      this.ws.send(JSON.stringify(message));
      console.log(`📤 发送消息:`, type, data);
    } catch (error) {
      console.error('❌ 发送消息失败:', error);
    }
  }
  
  // 处理接收到的消息
  private handleMessage(event: MessageEvent): void {
    try {
      const message: WSMessage = JSON.parse(event.data);
      console.log(`📥 收到消息:`, message.type, message.data);
      
      // 触发对应类型的监听器
      const listeners = this.eventListeners.get(message.type);
      if (listeners) {
        listeners.forEach(listener => listener(message));
      }
      
    } catch (error) {
      console.error('❌ 解析消息失败:', error);
    }
  }
  
  // 设置连接状态
  private setStatus(status: ConnectionStatus): void {
    if (this.status !== status) {
      this.status = status;
      console.log(`🔄 WebSocket状态变更: ${status}`);
      
      // 通知状态监听器
      this.statusListeners.forEach(listener => listener(status));
    }
  }
  
  // 计划重连
  private scheduleReconnect(): void {
    this.reconnectAttempts++;
    this.setStatus(ConnectionStatus.RECONNECTING);
    
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    console.log(`🔄 ${delay}ms后尝试第${this.reconnectAttempts}次重连...`);
    
    setTimeout(() => {
      this.connect().catch((error) => {
        console.error('❌ 重连失败:', error);
        
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
          console.error('❌ 达到最大重连次数，停止重连');
          this.setStatus(ConnectionStatus.ERROR);
        }
      });
    }, delay);
  }
  
  // 开始心跳
  private startHeartbeat(): void {
    this.heartbeatInterval = setInterval(() => {
      this.send(MessageType.HEARTBEAT, { timestamp: Date.now() });
    }, 30000); // 每30秒发送一次心跳
  }
  
  // 停止心跳
  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
    
    if (this.pingTimeout) {
      clearTimeout(this.pingTimeout);
      this.pingTimeout = null;
    }
  }
  
  // 添加事件监听器
  public on(type: MessageType, listener: EventListener): void {
    const listeners = this.eventListeners.get(type);
    if (listeners) {
      listeners.push(listener);
    }
  }
  
  // 移除事件监听器
  public off(type: MessageType, listener: EventListener): void {
    const listeners = this.eventListeners.get(type);
    if (listeners) {
      const index = listeners.indexOf(listener);
      if (index > -1) {
        listeners.splice(index, 1);
      }
    }
  }
  
  // 添加状态监听器
  public onStatusChange(listener: StatusListener): void {
    this.statusListeners.push(listener);
  }
  
  // 移除状态监听器
  public offStatusChange(listener: StatusListener): void {
    const index = this.statusListeners.indexOf(listener);
    if (index > -1) {
      this.statusListeners.splice(index, 1);
    }
  }
  
  // 获取当前状态
  public getStatus(): ConnectionStatus {
    return this.status;
  }
  
  // 是否已连接
  public isConnected(): boolean {
    return this.status === ConnectionStatus.CONNECTED;
  }
  
  // 便捷方法：发送任务更新
  public sendTaskUpdate(taskData: any): void {
    this.send(MessageType.TASK_UPDATE, taskData);
  }
  
  // 便捷方法：发送任务创建
  public sendTaskCreate(taskData: any): void {
    this.send(MessageType.TASK_CREATE, taskData);
  }
  
  // 便捷方法：发送任务删除
  public sendTaskDelete(taskId: number): void {
    this.send(MessageType.TASK_DELETE, { task_id: taskId });
  }
  
  // 便捷方法：发送聊天消息
  public sendChatMessage(message: string, messageId?: string): void {
    this.send(MessageType.CHAT_MESSAGE, {
      message,
      message_id: messageId || `msg_${Date.now()}`
    });
  }
  
  // 便捷方法：发送打字状态
  public sendTyping(isTyping: boolean, taskId?: number): void {
    this.send(MessageType.TYPING, {
      is_typing: isTyping,
      task_id: taskId
    });
  }
  
  // 便捷方法：更新用户状态
  public updateUserStatus(status: 'online' | 'away' | 'busy' | 'offline'): void {
    this.send(MessageType.USER_STATUS, {
      status,
      last_active: new Date().toISOString()
    });
  }
}

// WebSocket管理器单例
class WebSocketManager {
  private clients: Map<number, WebSocketClient> = new Map();
  
  // 为项目创建或获取WebSocket客户端
  public getClient(user: User, projectId: number): WebSocketClient {
    const key = projectId;
    
    if (!this.clients.has(key)) {
      const client = new WebSocketClient(user, projectId);
      this.clients.set(key, client);
    }
    
    return this.clients.get(key)!;
  }
  
  // 断开并移除客户端
  public removeClient(projectId: number): void {
    const client = this.clients.get(projectId);
    if (client) {
      client.disconnect();
      this.clients.delete(projectId);
    }
  }
  
  // 断开所有连接
  public disconnectAll(): void {
    this.clients.forEach(client => client.disconnect());
    this.clients.clear();
  }
  
  // 获取所有活跃连接
  public getActiveConnections(): number {
    return Array.from(this.clients.values()).filter(client => 
      client.isConnected()
    ).length;
  }
}

// 导出单例实例
export const wsManager = new WebSocketManager();

// 工具函数：格式化在线用户显示
export const formatOnlineUsers = (users: any[]): string => {
  if (users.length === 0) return '暂无在线成员';
  if (users.length === 1) return `${users[0].username} 在线`;
  if (users.length <= 3) {
    return users.map(u => u.username).join(', ') + ' 在线';
  }
  return `${users.slice(0, 2).map(u => u.username).join(', ')} 等 ${users.length} 人在线`;
};

// 工具函数：获取消息类型的中文显示
export const getMessageTypeLabel = (type: MessageType): string => {
  const labels = {
    [MessageType.TASK_UPDATE]: '任务更新',
    [MessageType.TASK_CREATE]: '创建任务',
    [MessageType.TASK_DELETE]: '删除任务',
    [MessageType.USER_JOIN]: '用户加入',
    [MessageType.USER_LEAVE]: '用户离开',
    [MessageType.USER_STATUS]: '状态变更',
    [MessageType.PROJECT_UPDATE]: '项目更新',
    [MessageType.CHAT_MESSAGE]: '聊天消息',
    [MessageType.TYPING]: '正在打字',
    [MessageType.HEARTBEAT]: '心跳检测'
  };
  
  return labels[type] || '未知消息';
};