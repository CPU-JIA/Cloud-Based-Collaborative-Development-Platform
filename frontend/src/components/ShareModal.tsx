import React, { useState } from 'react';

interface ShareModalProps {
  file: any;
  isOpen: boolean;
  onClose: () => void;
  onShare: (shareData: ShareData) => Promise<void>;
}

interface ShareData {
  password?: string;
  expiresAt?: string;
  permission: 'read' | 'write';
}

const ShareModal: React.FC<ShareModalProps> = ({ file, isOpen, onClose, onShare }) => {
  const [shareData, setShareData] = useState<ShareData>({
    permission: 'read',
  });
  const [shareLink, setShareLink] = useState<string>('');
  const [isSharing, setIsSharing] = useState(false);
  const [shareSuccess, setShareSuccess] = useState(false);
  const [passwordEnabled, setPasswordEnabled] = useState(false);
  const [expiryEnabled, setExpiryEnabled] = useState(false);

  const handleShare = async () => {
    setIsSharing(true);
    try {
      const data: ShareData = {
        permission: shareData.permission,
      };
      
      if (passwordEnabled && shareData.password) {
        data.password = shareData.password;
      }
      
      if (expiryEnabled && shareData.expiresAt) {
        data.expiresAt = shareData.expiresAt;
      }

      await onShare(data);
      setShareSuccess(true);
    } catch (error) {
      console.error('分享失败:', error);
    } finally {
      setIsSharing(false);
    }
  };

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      alert('链接已复制到剪贴板');
    } catch (error) {
      console.error('复制失败:', error);
    }
  };

  const resetModal = () => {
    setShareData({ permission: 'read' });
    setShareLink('');
    setShareSuccess(false);
    setPasswordEnabled(false);
    setExpiryEnabled(false);
  };

  const handleClose = () => {
    resetModal();
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center">
      <div className="bg-white rounded-2xl p-6 w-96 max-w-md shadow-2xl">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-xl font-bold text-gray-900">分享文件</h3>
          <button
            onClick={handleClose}
            className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
          >
            ✕
          </button>
        </div>

        {!shareSuccess ? (
          <>
            {/* 文件信息 */}
            <div className="mb-6 p-4 bg-gray-50 rounded-lg">
              <div className="flex items-center gap-3">
                <div className="text-2xl">📄</div>
                <div>
                  <div className="font-medium text-gray-900">{file?.original_name}</div>
                  <div className="text-sm text-gray-600">{file?.formatted_size}</div>
                </div>
              </div>
            </div>

            {/* 权限设置 */}
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                分享权限
              </label>
              <select
                value={shareData.permission}
                onChange={(e) => setShareData(prev => ({ ...prev, permission: e.target.value as 'read' | 'write' }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
              >
                <option value="read">只读（可查看和下载）</option>
                <option value="write">读写（可上传和编辑）</option>
              </select>
            </div>

            {/* 密码保护 */}
            <div className="mb-4">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={passwordEnabled}
                  onChange={(e) => setPasswordEnabled(e.target.checked)}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm font-medium text-gray-700">密码保护</span>
              </label>
              
              {passwordEnabled && (
                <input
                  type="password"
                  placeholder="设置访问密码"
                  value={shareData.password || ''}
                  onChange={(e) => setShareData(prev => ({ ...prev, password: e.target.value }))}
                  className="w-full mt-2 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                />
              )}
            </div>

            {/* 过期时间 */}
            <div className="mb-6">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={expiryEnabled}
                  onChange={(e) => setExpiryEnabled(e.target.checked)}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm font-medium text-gray-700">设置过期时间</span>
              </label>
              
              {expiryEnabled && (
                <input
                  type="datetime-local"
                  value={shareData.expiresAt || ''}
                  onChange={(e) => setShareData(prev => ({ ...prev, expiresAt: e.target.value }))}
                  className="w-full mt-2 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  min={new Date().toISOString().slice(0, 16)}
                />
              )}
            </div>

            {/* 操作按钮 */}
            <div className="flex gap-3">
              <button
                onClick={handleClose}
                className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleShare}
                disabled={isSharing || (passwordEnabled && !shareData.password) || (expiryEnabled && !shareData.expiresAt)}
                className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {isSharing ? '生成中...' : '创建分享链接'}
              </button>
            </div>
          </>
        ) : (
          <>
            {/* 分享成功界面 */}
            <div className="text-center mb-6">
              <div className="text-4xl mb-4">🎉</div>
              <div className="text-lg font-semibold text-gray-900 mb-2">分享链接已创建</div>
              <div className="text-sm text-gray-600">您可以通过以下链接分享文件</div>
            </div>

            {shareLink && (
              <div className="mb-6">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  分享链接
                </label>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={shareLink}
                    readOnly
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg bg-gray-50 text-sm"
                  />
                  <button
                    onClick={() => copyToClipboard(shareLink)}
                    className="px-3 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm"
                  >
                    复制
                  </button>
                </div>
              </div>
            )}

            {/* 分享信息摘要 */}
            <div className="mb-6 p-4 bg-gray-50 rounded-lg text-sm">
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <strong>权限:</strong> {shareData.permission === 'read' ? '只读' : '读写'}
                </div>
                <div>
                  <strong>密码:</strong> {passwordEnabled ? '已设置' : '无'}
                </div>
                {expiryEnabled && shareData.expiresAt && (
                  <div className="col-span-2">
                    <strong>过期时间:</strong> {new Date(shareData.expiresAt).toLocaleString()}
                  </div>
                )}
              </div>
            </div>

            <button
              onClick={handleClose}
              className="w-full px-4 py-2 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition-colors"
            >
              完成
            </button>
          </>
        )}
      </div>
    </div>
  );
};

export default ShareModal;