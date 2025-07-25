import React, { useState, useEffect, useRef } from 'react';
import { useChatMessages } from '../hooks/useWebSocket';
import { useAuth } from '../contexts/AuthContext';

interface ChatWidgetProps {
  projectId: number;
  isOpen: boolean;
  onToggle: () => void;
  sendChatMessage: (message: string) => void;
}

const ChatWidget: React.FC<ChatWidgetProps> = ({
  projectId,
  isOpen,
  onToggle,
  sendChatMessage
}) => {
  const { user } = useAuth();
  const { messages } = useChatMessages(projectId);
  const [inputMessage, setInputMessage] = useState('');
  const [hasNewMessages, setHasNewMessages] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // 自动滚动到最新消息
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    if (isOpen) {
      scrollToBottom();
      setHasNewMessages(false);
    } else if (messages.length > 0) {
      const lastMessage = messages[messages.length - 1];
      if (lastMessage.user.id !== user?.id) {
        setHasNewMessages(true);
      }
    }
  }, [messages, isOpen, user?.id]);

  const handleSendMessage = () => {
    if (inputMessage.trim()) {
      sendChatMessage(inputMessage.trim());
      setInputMessage('');
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  return (
    <>
      {/* 聊天切换按钮 */}
      <button
        className={`premium-chat-toggle ${hasNewMessages ? 'has-messages' : ''}`}
        onClick={onToggle}
        title="实时协作聊天"
      >
        💬
      </button>

      {/* 聊天容器 */}
      <div className={`premium-chat-container ${isOpen ? 'open' : ''}`}>
        {/* 聊天头部 */}
        <div className="premium-chat-header">
          <div className="premium-chat-title">实时协作聊天</div>
          <button className="premium-chat-close" onClick={onToggle}>
            ×
          </button>
        </div>

        {/* 消息列表 */}
        <div className="premium-chat-messages">
          {messages.length === 0 ? (
            <div style={{ 
              textAlign: 'center', 
              color: '#9ca3af', 
              padding: '2rem',
              fontSize: '0.9rem'
            }}>
              💬 开始实时协作对话...
            </div>
          ) : (
            messages.map((message) => (
              <div
                key={message.id}
                className={`premium-chat-message ${
                  message.user.id === user?.id ? 'own' : ''
                }`}
              >
                <div className="premium-chat-avatar">
                  {message.user.avatar ? (
                    <img src={message.user.avatar} alt={message.user.username} />
                  ) : (
                    message.user.username.charAt(0).toUpperCase()
                  )}
                </div>
                <div className="premium-chat-content">
                  <div className="premium-chat-bubble">
                    {message.message}
                  </div>
                  <div className="premium-chat-time">
                    {message.user.username} • {formatTime(message.timestamp)}
                  </div>
                </div>
              </div>
            ))
          )}
          <div ref={messagesEndRef} />
        </div>

        {/* 输入区域 */}
        <div className="premium-chat-input-container">
          <input
            type="text"
            className="premium-chat-input"
            placeholder="输入消息..."
            value={inputMessage}
            onChange={(e) => setInputMessage(e.target.value)}
            onKeyPress={handleKeyPress}
            maxLength={500}
          />
          <button
            className="premium-chat-send"
            onClick={handleSendMessage}
            disabled={!inputMessage.trim()}
            title="发送消息"
          >
            📤
          </button>
        </div>
      </div>
    </>
  );
};

export default ChatWidget;