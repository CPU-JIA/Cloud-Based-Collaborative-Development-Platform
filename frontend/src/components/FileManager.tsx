import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useAuth } from '../contexts/AuthContext';
import ShareModal from './ShareModal';

// 文件和文件夹类型定义
interface FileItem {
  id: number;
  name: string;
  original_name: string;
  size: number;
  mime_type: string;
  extension: string;
  file_type: string;
  formatted_size: string;
  can_preview: boolean;
  preview_url?: string;
  download_url: string;
  folder_id?: number;
  tags: string[];
  description: string;
  uploaded_by: number;
  download_count: number;
  created_at: string;
  updated_at: string;
}

interface Folder {
  id: number;
  name: string;
  path: string;
  description: string;
  parent_id?: number;
  level: number;
  is_public: boolean;
  created_by: number;
  created_at: string;
  updated_at: string;
}

interface FileManagerProps {
  projectId: number;
  isOpen: boolean;
  onClose: () => void;
}

const FileManager: React.FC<FileManagerProps> = ({ projectId, isOpen, onClose }) => {
  const { user } = useAuth();
  const [files, setFiles] = useState<FileItem[]>([]);
  const [folders, setFolders] = useState<Folder[]>([]);
  const [currentFolder, setCurrentFolder] = useState<Folder | null>(null);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  
  // UI状态
  const [view, setView] = useState<'grid' | 'list'>('grid');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedFiles, setSelectedFiles] = useState<Set<number>>(new Set());
  const [sortBy, setSortBy] = useState<'name' | 'size' | 'date'>('name');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc');
  const [filterType, setFilterType] = useState<'all' | 'image' | 'document' | 'code' | 'other'>('all');
  
  // 模态框状态
  const [showCreateFolder, setShowCreateFolder] = useState(false);
  const [showFilePreview, setShowFilePreview] = useState(false);
  const [showShareModal, setShowShareModal] = useState(false);
  const [previewFile, setPreviewFile] = useState<FileItem | null>(null);
  const [shareFile, setShareFile] = useState<FileItem | null>(null);
  const [newFolderName, setNewFolderName] = useState('');
  const [newFolderDescription, setNewFolderDescription] = useState('');
  
  const fileInputRef = useRef<HTMLInputElement>(null);
  const dragDropRef = useRef<HTMLDivElement>(null);

  // 加载文件和文件夹
  const loadFiles = useCallback(async () => {
    setLoading(true);
    try {
      const folderId = currentFolder?.id || null;
      const filesResponse = await fetch(
        `/api/v1/files/project/${projectId}?folder_id=${folderId || 'null'}&search=${searchTerm}&type=${filterType}&order=${sortBy} ${sortOrder}`,
        {
          headers: {
            'X-Tenant-ID': 'default',
          },
        }
      );
      
      const foldersResponse = await fetch(
        `/api/v1/folders/project/${projectId}?parent_id=${folderId || 'null'}`,
        {
          headers: {
            'X-Tenant-ID': 'default',
          },
        }
      );
      
      if (filesResponse.ok) {
        const filesData = await filesResponse.json();
        setFiles(filesData.files || []);
      }
      
      if (foldersResponse.ok) {
        const foldersData = await foldersResponse.json();
        setFolders(foldersData.folders || []);
      }
    } catch (error) {
      console.error('加载文件失败:', error);
    } finally {
      setLoading(false);
    }
  }, [projectId, currentFolder, searchTerm, filterType, sortBy, sortOrder]);

  useEffect(() => {
    if (isOpen) {
      loadFiles();
    }
  }, [isOpen, loadFiles]);

  // 文件上传处理
  const handleFileUpload = async (uploadFiles: FileList) => {
    if (!uploadFiles.length) return;
    
    setUploading(true);
    setUploadProgress(0);
    
    const formData = new FormData();
    formData.append('project_id', projectId.toString());
    if (currentFolder) {
      formData.append('folder_id', currentFolder.id.toString());
    }
    
    Array.from(uploadFiles).forEach(file => {
      formData.append('files', file);
    });
    
    try {
      const xhr = new XMLHttpRequest();
      
      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable) {
          const progress = (e.loaded / e.total) * 100;
          setUploadProgress(progress);
        }
      });
      
      xhr.addEventListener('load', () => {
        if (xhr.status === 200) {
          loadFiles();
          setUploading(false);
          setUploadProgress(0);
          if (fileInputRef.current) {
            fileInputRef.current.value = '';
          }
        }
      });
      
      xhr.addEventListener('error', () => {
        console.error('文件上传失败');
        setUploading(false);
        setUploadProgress(0);
      });
      
      xhr.open('POST', '/api/v1/files/upload');
      xhr.setRequestHeader('X-Tenant-ID', 'default');
      xhr.send(formData);
      
    } catch (error) {
      console.error('文件上传错误:', error);
      setUploading(false);
      setUploadProgress(0);
    }
  };

  // 创建文件夹
  const handleCreateFolder = async () => {
    if (!newFolderName.trim()) return;
    
    try {
      const response = await fetch('/api/v1/folders', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify({
          project_id: projectId,
          name: newFolderName.trim(),
          description: newFolderDescription.trim(),
          parent_id: currentFolder?.id || null,
        }),
      });
      
      if (response.ok) {
        loadFiles();
        setShowCreateFolder(false);
        setNewFolderName('');
        setNewFolderDescription('');
      }
    } catch (error) {
      console.error('创建文件夹失败:', error);
    }
  };

  // 文件预览
  const handleFilePreview = (file: FileItem) => {
    if (file.can_preview) {
      setPreviewFile(file);
      setShowFilePreview(true);
    }
  };

  // 文件下载
  const handleFileDownload = (file: FileItem) => {
    const link = document.createElement('a');
    link.href = file.download_url;
    link.download = file.original_name;
    link.click();
  };

  // 文件分享
  const handleFileShare = (file: FileItem) => {
    setShareFile(file);
    setShowShareModal(true);
  };

  const handleCreateShareLink = async (shareData: any) => {
    if (!shareFile) return;

    try {
      const response = await fetch(`/api/v1/files/${shareFile.id}/share`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default',
        },
        body: JSON.stringify(shareData),
      });

      if (response.ok) {
        const result = await response.json();
        console.log('分享创建成功:', result);
        // 可以在这里更新UI显示分享链接
        loadFiles(); // 重新加载文件列表
      } else {
        throw new Error('创建分享链接失败');
      }
    } catch (error) {
      console.error('分享文件失败:', error);
      throw error;
    }
  };

  // 拖拽上传
  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    
    const droppedFiles = Array.from(e.dataTransfer.files);
    if (droppedFiles.length > 0) {
      const fileList = new DataTransfer();
      droppedFiles.forEach(file => fileList.items.add(file));
      handleFileUpload(fileList.files);
    }
  };

  // 获取文件图标
  const getFileIcon = (file: FileItem) => {
    switch (file.file_type) {
      case 'image': return '🖼️';
      case 'document': return '📄';
      case 'code': return '📝';
      default: return '📁';
    }
  };

  // 格式化时间
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  if (!isOpen) return null;

  return (
    <>
      {/* 文件管理器遮罩 */}
      <div className="fixed inset-0 bg-black bg-opacity-50 z-50" onClick={onClose}>
        <div 
          className="fixed inset-4 bg-white rounded-2xl shadow-2xl flex flex-col"
          onClick={(e) => e.stopPropagation()}
        >
          {/* 头部工具栏 */}
          <div className="flex items-center justify-between p-6 border-b border-gray-200">
            <div className="flex items-center gap-4">
              <h2 className="text-2xl font-bold text-gray-900">📁 文件管理器</h2>
              
              {/* 面包屑导航 */}
              <nav className="flex items-center text-sm text-gray-600">
                <button
                  onClick={() => setCurrentFolder(null)}
                  className="hover:text-blue-600 transition-colors"
                >
                  项目根目录
                </button>
                {currentFolder && (
                  <>
                    <span className="mx-2">/</span>
                    <span className="text-gray-900 font-medium">{currentFolder.name}</span>
                  </>
                )}
              </nav>
            </div>
            
            <button
              onClick={onClose}
              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
            >
              ✕
            </button>
          </div>

          {/* 操作工具栏 */}
          <div className="flex items-center justify-between p-4 border-b border-gray-100">
            <div className="flex items-center gap-3">
              {/* 上传按钮 */}
              <button
                onClick={() => fileInputRef.current?.click()}
                disabled={uploading}
                className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
              >
                📤 上传文件
              </button>
              
              {/* 创建文件夹 */}
              <button
                onClick={() => setShowCreateFolder(true)}
                className="flex items-center gap-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors"
              >
                📁 新建文件夹
              </button>
              
              {/* 视图切换 */}
              <div className="flex border border-gray-300 rounded-lg overflow-hidden">
                <button
                  onClick={() => setView('grid')}
                  className={`px-3 py-2 text-sm ${view === 'grid' ? 'bg-blue-100 text-blue-700' : 'hover:bg-gray-50'}`}
                >
                  网格
                </button>
                <button
                  onClick={() => setView('list')}
                  className={`px-3 py-2 text-sm border-l border-gray-300 ${view === 'list' ? 'bg-blue-100 text-blue-700' : 'hover:bg-gray-50'}`}
                >
                  列表
                </button>
              </div>
            </div>
            
            <div className="flex items-center gap-3">
              {/* 搜索框 */}
              <div className="relative">
                <input
                  type="text"
                  placeholder="搜索文件..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
                <span className="absolute left-3 top-2.5 text-gray-400">🔍</span>
              </div>
              
              {/* 文件类型过滤 */}
              <select
                value={filterType}
                onChange={(e) => setFilterType(e.target.value as any)}
                className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
              >
                <option value="all">所有文件</option>
                <option value="image">图片</option>
                <option value="document">文档</option>
                <option value="code">代码</option>
                <option value="other">其他</option>
              </select>
              
              {/* 排序选择 */}
              <select
                value={`${sortBy}_${sortOrder}`}
                onChange={(e) => {
                  const [sort, order] = e.target.value.split('_');
                  setSortBy(sort as any);
                  setSortOrder(order as any);
                }}
                className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
              >
                <option value="name_asc">名称 ↑</option>
                <option value="name_desc">名称 ↓</option>
                <option value="size_asc">大小 ↑</option>
                <option value="size_desc">大小 ↓</option>
                <option value="date_asc">日期 ↑</option>
                <option value="date_desc">日期 ↓</option>
              </select>
            </div>
          </div>

          {/* 上传进度条 */}
          {uploading && (
            <div className="px-4 py-2 border-b border-gray-100">
              <div className="flex items-center gap-3">
                <span className="text-sm text-gray-600">上传中...</span>
                <div className="flex-1 bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                    style={{ width: `${uploadProgress}%` }}
                  />
                </div>
                <span className="text-sm text-gray-600">{Math.round(uploadProgress)}%</span>
              </div>
            </div>
          )}

          {/* 文件列表区域 */}
          <div 
            ref={dragDropRef}
            className="flex-1 overflow-auto p-4"
            onDragOver={handleDragOver}
            onDrop={handleDrop}
          >
            {loading ? (
              <div className="flex items-center justify-center h-64">
                <div className="loading">加载中...</div>
              </div>
            ) : (
              <>
                {/* 文件夹 */}
                {folders.length > 0 && (
                  <div className="mb-6">
                    <h3 className="text-lg font-semibold text-gray-800 mb-3">文件夹</h3>
                    <div className={view === 'grid' ? 'grid grid-cols-4 gap-4' : 'space-y-2'}>
                      {folders.map((folder) => (
                        <div
                          key={folder.id}
                          className={`p-4 border border-gray-200 rounded-lg hover:bg-gray-50 cursor-pointer transition-colors ${
                            view === 'list' ? 'flex items-center gap-4' : 'text-center'
                          }`}
                          onDoubleClick={() => setCurrentFolder(folder)}
                        >
                          <div className="text-4xl mb-2">📁</div>
                          <div>
                            <div className="font-medium text-gray-900">{folder.name}</div>
                            {folder.description && (
                              <div className="text-sm text-gray-600 mt-1">{folder.description}</div>
                            )}
                            <div className="text-xs text-gray-500 mt-1">
                              {formatDate(folder.created_at)}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* 文件 */}
                {files.length > 0 && (
                  <div>
                    <h3 className="text-lg font-semibold text-gray-800 mb-3">文件</h3>
                    <div className={view === 'grid' ? 'grid grid-cols-4 gap-4' : 'space-y-2'}>
                      {files.map((file) => (
                        <div
                          key={file.id}
                          className={`p-4 border border-gray-200 rounded-lg hover:bg-gray-50 cursor-pointer transition-colors ${
                            view === 'list' ? 'flex items-center gap-4' : 'text-center'
                          } ${selectedFiles.has(file.id) ? 'ring-2 ring-blue-500 bg-blue-50' : ''}`}
                          onClick={() => {
                            const newSelected = new Set(selectedFiles);
                            if (newSelected.has(file.id)) {
                              newSelected.delete(file.id);
                            } else {
                              newSelected.add(file.id);
                            }
                            setSelectedFiles(newSelected);
                          }}
                          onDoubleClick={() => handleFilePreview(file)}
                        >
                          <div className="text-4xl mb-2">{getFileIcon(file)}</div>
                          <div>
                            <div className="font-medium text-gray-900 truncate" title={file.original_name}>
                              {file.original_name}
                            </div>
                            <div className="text-sm text-gray-600">{file.formatted_size}</div>
                            {file.tags.length > 0 && (
                              <div className="flex flex-wrap gap-1 mt-2">
                                {file.tags.map((tag, idx) => (
                                  <span
                                    key={idx}
                                    className="px-2 py-1 bg-blue-100 text-blue-700 text-xs rounded-full"
                                  >
                                    {tag}
                                  </span>
                                ))}
                              </div>
                            )}
                            <div className="text-xs text-gray-500 mt-1">
                              {formatDate(file.created_at)}
                            </div>
                            
                            {/* 文件操作按钮 */}
                            <div className="flex gap-2 mt-2">
                              {file.can_preview && (
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleFilePreview(file);
                                  }}
                                  className="px-2 py-1 bg-blue-100 text-blue-700 text-xs rounded hover:bg-blue-200"
                                >
                                  预览
                                </button>
                              )}
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleFileDownload(file);
                                }}
                                className="px-2 py-1 bg-green-100 text-green-700 text-xs rounded hover:bg-green-200"
                              >
                                下载
                              </button>
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleFileShare(file);
                                }}
                                className="px-2 py-1 bg-purple-100 text-purple-700 text-xs rounded hover:bg-purple-200"
                              >
                                分享
                              </button>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* 空状态 */}
                {!loading && folders.length === 0 && files.length === 0 && (
                  <div className="text-center py-16">
                    <div className="text-6xl mb-4">📁</div>
                    <div className="text-xl text-gray-600 mb-2">暂无文件</div>
                    <div className="text-gray-500">
                      拖拽文件到此处或点击上传按钮添加文件
                    </div>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      </div>

      {/* 隐藏的文件输入 */}
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="hidden"
        onChange={(e) => {
          if (e.target.files) {
            handleFileUpload(e.target.files);
          }
        }}
      />

      {/* 创建文件夹模态框 */}
      {showCreateFolder && (
        <div className="fixed inset-0 bg-black bg-opacity-50 z-60 flex items-center justify-center">
          <div className="bg-white rounded-lg p-6 w-96">
            <h3 className="text-lg font-semibold mb-4">创建新文件夹</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  文件夹名称
                </label>
                <input
                  type="text"
                  value={newFolderName}
                  onChange={(e) => setNewFolderName(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  placeholder="输入文件夹名称..."
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  描述 (可选)
                </label>
                <textarea
                  value={newFolderDescription}
                  onChange={(e) => setNewFolderDescription(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                  rows={3}
                  placeholder="输入文件夹描述..."
                />
              </div>
            </div>
            <div className="flex gap-3 mt-6">
              <button
                onClick={handleCreateFolder}
                disabled={!newFolderName.trim()}
                className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                创建
              </button>
              <button
                onClick={() => {
                  setShowCreateFolder(false);
                  setNewFolderName('');
                  setNewFolderDescription('');
                }}
                className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
              >
                取消
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 文件预览模态框 */}
      {showFilePreview && previewFile && (
        <div className="fixed inset-0 bg-black bg-opacity-75 z-60 flex items-center justify-center">
          <div className="bg-white rounded-lg p-6 max-w-4xl max-h-[90vh] overflow-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold">{previewFile.original_name}</h3>
              <button
                onClick={() => {
                  setShowFilePreview(false);
                  setPreviewFile(null);
                }}
                className="p-2 hover:bg-gray-100 rounded-lg"
              >
                ✕
              </button>
            </div>
            
            {/* 预览内容 */}
            {previewFile.preview_url && (
              <div className="mb-4">
                {previewFile.file_type === 'image' ? (
                  <img
                    src={previewFile.preview_url}
                    alt={previewFile.original_name}
                    className="max-w-full max-h-96 object-contain mx-auto"
                  />
                ) : (
                  <iframe
                    src={previewFile.preview_url}
                    className="w-full h-96 border border-gray-300 rounded"
                    title={previewFile.original_name}
                  />
                )}
              </div>
            )}
            
            {/* 文件信息 */}
            <div className="border-t pt-4 text-sm text-gray-600">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <strong>文件大小:</strong> {previewFile.formatted_size}
                </div>
                <div>
                  <strong>文件类型:</strong> {previewFile.mime_type}
                </div>
                <div>
                  <strong>上传时间:</strong> {formatDate(previewFile.created_at)}
                </div>
                <div>
                  <strong>下载次数:</strong> {previewFile.download_count}
                </div>
              </div>
              {previewFile.description && (
                <div className="mt-2">
                  <strong>描述:</strong> {previewFile.description}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* 文件分享模态框 */}
      <ShareModal
        file={shareFile}
        isOpen={showShareModal}
        onClose={() => {
          setShowShareModal(false);
          setShareFile(null);
        }}
        onShare={handleCreateShareLink}
      />
    </>
  );
};

export default FileManager;