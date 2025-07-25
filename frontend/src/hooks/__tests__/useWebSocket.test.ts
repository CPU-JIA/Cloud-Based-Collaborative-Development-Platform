import { renderHook, act, waitFor } from '@testing-library/react';
import { jest } from '@jest/globals';

import { useWebSocket, useTaskUpdates, useOnlineUsers } from '../useWebSocket';
import { ConnectionStatus, MessageType } from '../../utils/websocket';

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState: number = MockWebSocket.CONNECTING;
  onopen: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;

  constructor(public url: string) {
    // 模拟异步连接
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN;
      if (this.onopen) {
        this.onopen(new Event('open'));
      }
    }, 10);
  }

  send(data: string) {
    if (this.readyState !== MockWebSocket.OPEN) {
      throw new Error('WebSocket is not open');
    }
    // 在测试中可以通过这个方法验证发送的数据
    (this as any).lastSentData = data;
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) {
      this.onclose(new CloseEvent('close'));
    }
  }

  // 测试辅助方法：模拟接收消息
  simulateMessage(data: any) {
    if (this.onmessage) {
      this.onmessage(new MessageEvent('message', { 
        data: JSON.stringify(data) 
      }));
    }
  }

  // 测试辅助方法：模拟连接错误
  simulateError() {
    if (this.onerror) {
      this.onerror(new Event('error'));
    }
  }
}

// 全局替换WebSocket
(global as any).WebSocket = MockWebSocket;

describe('useWebSocket', () => {
  let mockWebSocket: MockWebSocket;

  beforeEach(() => {
    jest.clearAllMocks();
    // 重置WebSocket mock
    jest.spyOn(global, 'WebSocket').mockImplementation((url: string) => {
      mockWebSocket = new MockWebSocket(url);
      return mockWebSocket as any;
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('成功建立WebSocket连接', async () => {
    const onMessage = jest.fn();
    
    const { result } = renderHook(() => 
      useWebSocket({
        projectId: 1,
        onMessage
      })
    );

    // 初始状态应该是连接中
    expect(result.current.status).toBe(ConnectionStatus.CONNECTING);
    expect(result.current.isConnected).toBe(false);

    // 等待连接建立
    await waitFor(() => {
      expect(result.current.status).toBe(ConnectionStatus.CONNECTED);
    });

    expect(result.current.isConnected).toBe(true);
  });

  it('处理连接失败', async () => {
    const onMessage = jest.fn();
    
    const { result } = renderHook(() => 
      useWebSocket({
        projectId: 1,
        onMessage
      })
    );

    await waitFor(() => {
      expect(mockWebSocket).toBeDefined();
    });

    // 模拟连接错误
    act(() => {
      mockWebSocket.simulateError();
    });

    await waitFor(() => {
      expect(result.current.status).toBe(ConnectionStatus.DISCONNECTED);
    });

    expect(result.current.isConnected).toBe(false);
  });

  it('发送任务更新消息', async () => {
    const onMessage = jest.fn();
    
    const { result } = renderHook(() => 
      useWebSocket({
        projectId: 1,
        onMessage
      })
    );

    // 等待连接建立
    await waitFor(() => {
      expect(result.current.isConnected).toBe(true);
    });

    const taskData = {
      task_id: 1,
      title: '测试任务',
      description: '任务描述',
      status_id: '2',
      priority: 'high',
      assignee_id: 1,
      due_date: '2024-12-31'
    };

    // 发送任务更新
    act(() => {
      result.current.sendTaskUpdate(taskData);
    });

    // 验证发送的数据
    const sentData = JSON.parse((mockWebSocket as any).lastSentData);
    expect(sentData).toEqual({
      type: MessageType.TASK_UPDATE,
      project_id: 1,
      data: taskData,
      timestamp: expect.any(String)
    });
  });

  it('发送任务创建消息', async () => {
    const onMessage = jest.fn();
    
    const { result } = renderHook(() => 
      useWebSocket({
        projectId: 1,
        onMessage
      })
    );

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true);
    });

    const taskData = {
      task_id: 2,
      title: '新任务',
      description: '新任务描述',
      status_id: '1',
      priority: 'medium',
      assignee_id: 2,
      due_date: '2024-11-30'
    };

    act(() => {
      result.current.sendTaskCreate(taskData);
    });

    const sentData = JSON.parse((mockWebSocket as any).lastSentData);
    expect(sentData.type).toBe(MessageType.TASK_CREATE);
    expect(sentData.data).toEqual(taskData);
  });

  it('发送任务删除消息', async () => {
    const onMessage = jest.fn();
    
    const { result } = renderHook(() => 
      useWebSocket({
        projectId: 1,
        onMessage
      })
    );

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true);
    });

    act(() => {
      result.current.sendTaskDelete(123);
    });

    const sentData = JSON.parse((mockWebSocket as any).lastSentData);
    expect(sentData.type).toBe(MessageType.TASK_DELETE);
    expect(sentData.data.task_id).toBe(123);
  });

  it('发送聊天消息', async () => {
    const onMessage = jest.fn();
    
    const { result } = renderHook(() => 
      useWebSocket({
        projectId: 1,
        onMessage
      })
    );

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true);
    });

    act(() => {
      result.current.sendChatMessage('Hello, team!');
    });

    const sentData = JSON.parse((mockWebSocket as any).lastSentData);
    expect(sentData.type).toBe(MessageType.CHAT_MESSAGE);
    expect(sentData.data.message).toBe('Hello, team!');
    expect(sentData.data.timestamp).toBeDefined();
  });

  it('接收并处理消息', async () => {
    const onMessage = jest.fn();
    
    const { result } = renderHook(() => 
      useWebSocket({
        projectId: 1,
        onMessage
      })
    );

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true);
    });

    const incomingMessage = {
      type: MessageType.TASK_UPDATE,
      project_id: 1,
      user_id: 2,
      data: {
        task_id: 1,
        title: '更新的任务',
        status_id: '3'
      },
      timestamp: new Date().toISOString()
    };

    // 模拟接收消息
    act(() => {
      mockWebSocket.simulateMessage(incomingMessage);
    });

    expect(onMessage).toHaveBeenCalledWith(incomingMessage);
  });

  it('自动重连功能', async () => {
    const onMessage = jest.fn();
    
    const { result } = renderHook(() => 
      useWebSocket({
        projectId: 1,
        onMessage
      })
    );

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true);
    });

    // 模拟连接断开
    act(() => {
      mockWebSocket.close();
    });

    await waitFor(() => {
      expect(result.current.status).toBe(ConnectionStatus.DISCONNECTED);
    });

    // 等待自动重连
    await waitFor(() => {
      expect(result.current.status).toBe(ConnectionStatus.RECONNECTING);
    }, { timeout: 6000 });
  });

  it('清理连接', async () => {
    const onMessage = jest.fn();
    
    const { result, unmount } = renderHook(() => 
      useWebSocket({
        projectId: 1,
        onMessage
      })
    );

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true);
    });

    // 卸载组件应该关闭连接
    unmount();

    expect(mockWebSocket.readyState).toBe(MockWebSocket.CLOSED);
  });
});

describe('useTaskUpdates', () => {
  let mockWebSocket: MockWebSocket;

  beforeEach(() => {
    jest.spyOn(global, 'WebSocket').mockImplementation((url: string) => {
      mockWebSocket = new MockWebSocket(url);
      return mockWebSocket as any;
    });
  });

  it('跟踪任务更新', async () => {
    const { result } = renderHook(() => useTaskUpdates(1));

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true);
    });

    const taskUpdate = {
      type: MessageType.TASK_UPDATE,
      project_id: 1,
      user_id: 2,
      data: {
        task_id: 1,
        title: '更新的任务',
        status_id: '2'
      },
      timestamp: new Date().toISOString()
    };

    // 模拟接收任务更新
    act(() => {
      mockWebSocket.simulateMessage(taskUpdate);
    });

    expect(result.current.lastUpdate).toEqual(taskUpdate);
  });

  it('只处理任务相关消息', async () => {
    const { result } = renderHook(() => useTaskUpdates(1));

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true);
    });

    const chatMessage = {
      type: MessageType.CHAT_MESSAGE,
      project_id: 1,
      user_id: 2,
      data: {
        message: '这是聊天消息'
      },
      timestamp: new Date().toISOString()
    };

    // 模拟接收聊天消息
    act(() => {
      mockWebSocket.simulateMessage(chatMessage);
    });

    // lastUpdate应该仍然为null
    expect(result.current.lastUpdate).toBeNull();
  });
});

describe('useOnlineUsers', () => {
  let mockWebSocket: MockWebSocket;

  beforeEach(() => {
    jest.spyOn(global, 'WebSocket').mockImplementation((url: string) => {
      mockWebSocket = new MockWebSocket(url);
      return mockWebSocket as any;
    });
  });

  it('跟踪在线用户', async () => {
    const { result } = renderHook(() => useOnlineUsers(1));

    await waitFor(() => {
      // 初始应该为空数组
      expect(result.current).toEqual([]);
    });

    // 等待WebSocket连接
    await waitFor(() => {
      expect(mockWebSocket.readyState).toBe(MockWebSocket.OPEN);
    });

    const onlineUsers = [
      {
        user_id: 1,
        username: 'user1',
        display_name: '用户1',
        avatar: '',
        status: 'active',
        last_seen: new Date().toISOString()
      },
      {
        user_id: 2,
        username: 'user2',
        display_name: '用户2',
        avatar: '',
        status: 'active',
        last_seen: new Date().toISOString()
      }
    ];

    const userStatusMessage = {
      type: MessageType.USER_STATUS,
      project_id: 1,
      data: {
        online_users: onlineUsers
      },
      timestamp: new Date().toISOString()
    };

    // 模拟接收用户状态更新
    act(() => {
      mockWebSocket.simulateMessage(userStatusMessage);
    });

    await waitFor(() => {
      expect(result.current).toEqual(onlineUsers);
    });
  });

  it('处理用户上线消息', async () => {
    const { result } = renderHook(() => useOnlineUsers(1));

    await waitFor(() => {
      expect(mockWebSocket.readyState).toBe(MockWebSocket.OPEN);
    });

    const userJoinMessage = {
      type: MessageType.USER_JOIN,
      project_id: 1,
      data: {
        user: {
          user_id: 3,
          username: 'user3',
          display_name: '用户3',
          avatar: '',
          status: 'active',
          last_seen: new Date().toISOString()
        }
      },
      timestamp: new Date().toISOString()
    };

    act(() => {
      mockWebSocket.simulateMessage(userJoinMessage);
    });

    await waitFor(() => {
      expect(result.current).toContainEqual(userJoinMessage.data.user);
    });
  });

  it('处理用户离线消息', async () => {
    const { result } = renderHook(() => useOnlineUsers(1));

    await waitFor(() => {
      expect(mockWebSocket.readyState).toBe(MockWebSocket.OPEN);
    });

    // 先添加一些在线用户
    const onlineUsers = [
      {
        user_id: 1,
        username: 'user1',
        display_name: '用户1',
        avatar: '',
        status: 'active',
        last_seen: new Date().toISOString()
      },
      {
        user_id: 2,
        username: 'user2',
        display_name: '用户2',
        avatar: '',
        status: 'active',
        last_seen: new Date().toISOString()
      }
    ];

    act(() => {
      mockWebSocket.simulateMessage({
        type: MessageType.USER_STATUS,
        project_id: 1,
        data: { online_users: onlineUsers },
        timestamp: new Date().toISOString()
      });
    });

    await waitFor(() => {
      expect(result.current).toHaveLength(2);
    });

    // 模拟用户离线
    const userLeaveMessage = {
      type: MessageType.USER_LEAVE,
      project_id: 1,
      data: {
        user_id: 1
      },
      timestamp: new Date().toISOString()
    };

    act(() => {
      mockWebSocket.simulateMessage(userLeaveMessage);
    });

    await waitFor(() => {
      expect(result.current).toHaveLength(1);
      expect(result.current[0].user_id).toBe(2);
    });
  });
});